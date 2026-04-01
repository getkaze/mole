package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	ghinstall "github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/spf13/cobra"

	"github.com/getkaze/mole/internal/aggregator"
	"github.com/getkaze/mole/internal/config"
	"github.com/getkaze/mole/internal/dashboard"
	"github.com/getkaze/mole/internal/git"
	ghclient "github.com/getkaze/mole/internal/github"
	"github.com/getkaze/mole/internal/llm"
	"github.com/getkaze/mole/internal/migrate"
	"github.com/getkaze/mole/internal/queue"
	"github.com/getkaze/mole/internal/review"
	"github.com/getkaze/mole/internal/server"
	"github.com/getkaze/mole/internal/store"
	"github.com/getkaze/mole/internal/updater"
	"github.com/getkaze/mole/internal/worker"
)

var (
	version    = "dev"
	configPath string
)

func main() {
	root := &cobra.Command{
		Use:     "mole",
		Short:   "AI-powered PR reviewer — digs deep into code, elevates those who write it",
		Version: version,
	}

	root.CompletionOptions.DisableDefaultCmd = true
	root.PersistentFlags().StringVar(&configPath, "config", "mole.yaml", "path to config file")

	root.AddCommand(
		serveCmd(),
		migrateCmd(),
		healthCmd(),
		reviewCmd(),
		initCmd(),
		adminCmd(),
		syncCmd(),
		updateCmd(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start webhook server and worker pool",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			setupLogging(cfg.Log.Level)

			st, err := store.NewMySQL(cfg.MySQL.DSN())
			if err != nil {
				return fmt.Errorf("mysql: %w", err)
			}
			defer st.Close()

			applied, err := migrate.Run(st.DB())
			if err != nil {
				return fmt.Errorf("auto-migrate: %w", err)
			}
			if applied > 0 {
				slog.Info("auto-migrate", "applied", applied)
			}

			q, err := queue.New(cfg.Valkey.Addr())
			if err != nil {
				return fmt.Errorf("valkey: %w", err)
			}
			defer q.Close()

			var gwFactory ghclient.GatewayFactory
			if cfg.Server.Environment == "development" {
				// Dev mode: no GitHub App, use a no-op gateway for workers
				gwFactory = ghclient.NewLocalGatewayFactory(".")
			} else {
				ghFactory := ghclient.NewClientFactory(cfg.GitHub.AppID, cfg.GitHub.PrivateKeyPath)
				gwFactory = ghclient.NewRemoteGatewayFactory(ghFactory)
			}
			provider := llm.NewClaude(cfg.LLM.APIKey)

			// Exploration dependencies
			var repoMgr *git.RepoManager
			var explorer *llm.Explorer
			if cfg.Repos.BasePath != "" {
				tokenFunc := newTokenFunc(cfg.GitHub.AppID, cfg.GitHub.PrivateKeyPath)
				repoMgr = git.NewRepoManager(cfg.Repos.BasePath, tokenFunc)
				if repoMgr.IsAvailable() {
					explorer = llm.NewExplorer(cfg.LLM.APIKey, cfg.Exploration.MaxTurns, cfg.Exploration.Model)
					repoMgr.CleanupStale()
					slog.Info("exploration enabled", "base_path", cfg.Repos.BasePath, "max_turns", cfg.Exploration.MaxTurns)
				} else {
					slog.Warn("exploration: git binary not found, exploration disabled")
					repoMgr = nil
				}
			}

			svc := review.NewService(gwFactory, provider, explorer, repoMgr, st, cfg.LLM.ReviewModel, cfg.LLM.DeepReviewModel, cfg.Defaults.Language, cfg.Defaults.Personality)

			pool := worker.NewPool(q, svc.Execute, cfg.Worker.Count)

			var extras []server.RouteRegistrar
			isDev := cfg.Server.Environment == "development"
			if cfg.Dashboard.Enabled() || isDev {
				sessionSecret := cfg.Dashboard.SessionSecret
				if sessionSecret == "" && isDev {
					sessionSecret = "dev-secret-not-for-production"
				}
				dash, err := dashboard.New(st, dashboard.Config{
					GitHubClientID:     cfg.Dashboard.GitHubClientID,
					GitHubClientSecret: cfg.Dashboard.GitHubClientSecret,
					SessionSecret:      sessionSecret,
					BaseURL:            cfg.Dashboard.BaseURL,
					AllowedOrg:         cfg.Dashboard.AllowedOrg,
					Pricing:            cfg.LLM.Pricing,
					Version:            version,
					Environment:        cfg.Server.Environment,
				})
				if err != nil {
					return fmt.Errorf("dashboard: %w", err)
				}
				extras = append(extras, dash)
				slog.Info("dashboard enabled", "base_url", cfg.Dashboard.BaseURL)
			}

			srv := server.New(cfg.Server.Port, cfg.GitHub.WebhookSecret, q, st, extras...)

			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			pool.Start(ctx)

			if cfg.Server.Environment != "development" {
				ghFactory := ghclient.NewClientFactory(cfg.GitHub.AppID, cfg.GitHub.PrivateKeyPath)
				reactionSyncer := aggregator.NewReactionSyncer(st, ghFactory)
				agg := aggregator.New(st, time.Hour, aggregator.WithReactionSyncer(reactionSyncer))
				go agg.Run(ctx)
			} else {
				agg := aggregator.New(st, time.Hour)
				go agg.Run(ctx)
			}

			go func() {
				if err := srv.Start(); err != nil {
					slog.Error("server error", "error", err)
					stop()
				}
			}()

			startupAttrs := []any{
				"port", cfg.Server.Port,
				"workers", cfg.Worker.Count,
				"review_model", cfg.LLM.ReviewModel,
				"deep_review_model", cfg.LLM.DeepReviewModel,
			}
			if explorer != nil {
				startupAttrs = append(startupAttrs,
					"exploration_model", cfg.Exploration.Model,
					"exploration_max_turns", cfg.Exploration.MaxTurns,
				)
			}
			slog.Info("mole is running", startupAttrs...)

			<-ctx.Done()

			shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			srv.Shutdown(shutdownCtx)
			pool.Stop()

			slog.Info("mole stopped")
			return nil
		},
	}
}

func migrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			st, err := store.NewMySQL(cfg.MySQL.DSN())
			if err != nil {
				return fmt.Errorf("mysql: %w", err)
			}
			defer st.Close()

			applied, err := migrate.Run(st.DB())
			if err != nil {
				return fmt.Errorf("migration: %w", err)
			}

			if applied == 0 {
				fmt.Println("No new migrations")
			} else {
				fmt.Printf("Applied %d migrations\n", applied)
			}
			return nil
		},
	}

	cmd.AddCommand(migrateCleanCmd())
	return cmd
}

func migrateCleanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clean",
		Short: "Drop all tables and re-run migrations from scratch",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			st, err := store.NewMySQL(cfg.MySQL.DSN())
			if err != nil {
				return fmt.Errorf("mysql: %w", err)
			}
			defer st.Close()

			applied, err := migrate.Clean(st.DB())
			if err != nil {
				return fmt.Errorf("migration clean: %w", err)
			}

			fmt.Printf("Database cleaned and %d migrations applied\n", applied)
			return nil
		},
	}
}

func healthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check connectivity to MySQL, Valkey, and GitHub",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			ctx := context.Background()

			fmt.Print("MySQL... ")
			st, err := store.NewMySQL(cfg.MySQL.DSN())
			if err != nil {
				fmt.Printf("FAIL (%s)\n", err)
			} else {
				if err := st.Ping(ctx); err != nil {
					fmt.Printf("FAIL (%s)\n", err)
				} else {
					fmt.Println("OK")
				}
				st.Close()
			}

			fmt.Print("Valkey... ")
			q, err := queue.New(cfg.Valkey.Addr())
			if err != nil {
				fmt.Printf("FAIL (%s)\n", err)
			} else {
				if err := q.Ping(ctx); err != nil {
					fmt.Printf("FAIL (%s)\n", err)
				} else {
					fmt.Println("OK")
				}
				q.Close()
			}

			fmt.Println("GitHub... OK (credentials loaded)")
			return nil
		},
	}
}

func syncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync reactions from GitHub, recalculate scores, and update metrics",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			setupLogging(cfg.Log.Level)

			st, err := store.NewMySQL(cfg.MySQL.DSN())
			if err != nil {
				return fmt.Errorf("mysql: %w", err)
			}
			defer st.Close()

			ghFactory := ghclient.NewClientFactory(cfg.GitHub.AppID, cfg.GitHub.PrivateKeyPath)
			reactionSyncer := aggregator.NewReactionSyncer(st, ghFactory)
			agg := aggregator.New(st, time.Hour, aggregator.WithReactionSyncer(reactionSyncer))

			fmt.Println("Syncing reactions from GitHub...")
			_, recalculated := agg.SyncOnce(context.Background())

			fmt.Printf("Done. Recalculated %d review scores. Metrics updated.\n", recalculated)
			return nil
		},
	}
}

func reviewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "review [owner/repo#pr]",
		Short: "Review a pull request",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deep, _ := cmd.Flags().GetBool("deep")
			digFlag, _ := cmd.Flags().GetBool("dig")
			localDir, _ := cmd.Flags().GetString("local")

			jobType := "standard"
			if digFlag {
				jobType = "dig"
			} else if deep {
				jobType = "deep"
			}

			// Local mode: read PR data from fixtures, skip GitHub
			if localDir != "" {
				cfg, err := config.LoadLocal(configPath)
				if err != nil {
					return err
				}

				setupLogging(cfg.Log.Level)

				st, err := store.NewMySQL(cfg.MySQL.DSN())
				if err != nil {
					return fmt.Errorf("mysql: %w", err)
				}
				defer st.Close()

				gwFactory := ghclient.NewLocalGatewayFactory(localDir)
				provider := llm.NewClaude(cfg.LLM.APIKey)

				// Local mode: exploration uses localDir as the worktree directly, no git clone needed
				var explorer *llm.Explorer
				if cfg.LLM.APIKey != "" {
					explorer = llm.NewExplorer(cfg.LLM.APIKey, cfg.Exploration.MaxTurns, cfg.Exploration.Model)
				}

				svc := review.NewService(gwFactory, provider, explorer, nil, st, cfg.LLM.ReviewModel, cfg.LLM.DeepReviewModel, cfg.Defaults.Language, cfg.Defaults.Personality)

				// Read repo and PR number from fixtures if available
				gw := ghclient.NewLocalGateway(localDir)
				prInfo, err := gw.GetPRInfo(context.Background(), "", 0)
				if err != nil {
					return fmt.Errorf("reading pr.json: %w", err)
				}

				repo := "local/review"
				prNumber := 1
				if prInfo.Repo != "" {
					repo = prInfo.Repo
				}
				if prInfo.PRNumber > 0 {
					prNumber = prInfo.PRNumber
				}

				job := queue.Job{
					ID:       fmt.Sprintf("local-%s", localDir),
					Type:     jobType,
					Repo:     repo,
					PRNumber: prNumber,
				}

				fmt.Printf("Reviewing from %s (%s)...\n", localDir, jobType)
				if err := svc.Execute(context.Background(), job); err != nil {
					return fmt.Errorf("review failed: %w", err)
				}
				return nil
			}

			// Remote mode: full config required
			if len(args) == 0 {
				return fmt.Errorf("usage: mole review owner/repo#pr  or  mole review --local <dir>")
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			setupLogging(cfg.Log.Level)

			owner, repo, prNumber, err := parsePRRef(args[0])
			if err != nil {
				return err
			}

			st, err := store.NewMySQL(cfg.MySQL.DSN())
			if err != nil {
				return fmt.Errorf("mysql: %w", err)
			}
			defer st.Close()

			ghFactory := ghclient.NewClientFactory(cfg.GitHub.AppID, cfg.GitHub.PrivateKeyPath)
			gwFactory := ghclient.NewRemoteGatewayFactory(ghFactory)
			provider := llm.NewClaude(cfg.LLM.APIKey)

			var repoMgr *git.RepoManager
			var explorer *llm.Explorer
			if cfg.Repos.BasePath != "" {
				tokenFunc := newTokenFunc(cfg.GitHub.AppID, cfg.GitHub.PrivateKeyPath)
				repoMgr = git.NewRepoManager(cfg.Repos.BasePath, tokenFunc)
				if repoMgr.IsAvailable() {
					explorer = llm.NewExplorer(cfg.LLM.APIKey, cfg.Exploration.MaxTurns, cfg.Exploration.Model)
				} else {
					repoMgr = nil
				}
			}

			svc := review.NewService(gwFactory, provider, explorer, repoMgr, st, cfg.LLM.ReviewModel, cfg.LLM.DeepReviewModel, cfg.Defaults.Language, cfg.Defaults.Personality)

			installID, _ := cmd.Flags().GetInt64("install-id")

			job := queue.Job{
				ID:        fmt.Sprintf("cli-%s/%s#%d", owner, repo, prNumber),
				Type:      jobType,
				Repo:      fmt.Sprintf("%s/%s", owner, repo),
				PRNumber:  prNumber,
				InstallID: installID,
			}

			fmt.Printf("Reviewing %s/%s#%d (%s)...\n", owner, repo, prNumber, jobType)
			if err := svc.Execute(context.Background(), job); err != nil {
				return fmt.Errorf("review failed: %w", err)
			}

			fmt.Println("Review posted successfully.")
			return nil
		},
	}
	cmd.Flags().Bool("deep", false, "use Claude Opus for deep review")
	cmd.Flags().Bool("dig", false, "clone repo, explore codebase with Haiku, then review with Opus")
	cmd.Flags().String("local", "", "read PR data from local fixtures directory (no GitHub needed)")
	cmd.Flags().Int64("install-id", 0, "GitHub App installation ID")
	return cmd
}

// parsePRRef parses "owner/repo#123" into components.
func parsePRRef(ref string) (owner, repo string, pr int, err error) {
	parts := strings.SplitN(ref, "#", 2)
	if len(parts) != 2 {
		return "", "", 0, fmt.Errorf("invalid format: use owner/repo#pr (got %q)", ref)
	}

	pr, err = strconv.Atoi(parts[1])
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid PR number: %q", parts[1])
	}

	repoParts := strings.SplitN(parts[0], "/", 2)
	if len(repoParts) != 2 {
		return "", "", 0, fmt.Errorf("invalid repo format: use owner/repo (got %q)", parts[0])
	}

	return repoParts[0], repoParts[1], pr, nil
}

func updateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update mole to the latest version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("==> checking for updates...")

			result, err := updater.Check(version)
			if err != nil {
				return fmt.Errorf("check failed: %w", err)
			}

			if !result.Available {
				fmt.Printf("  ✓ already up to date (%s)\n", version)
				return nil
			}

			fmt.Printf("==> new version available: %s (current: %s)\n", result.Latest, result.Current)
			fmt.Printf("==> downloading mole %s...\n", result.Latest)

			tmpPath, err := updater.Download(result.Latest)
			if err != nil {
				return fmt.Errorf("download failed: %w", err)
			}
			defer os.Remove(tmpPath)

			if err := updater.Replace(tmpPath); err != nil {
				return fmt.Errorf("replace failed: %w\n\nhint: try running with sudo", err)
			}

			fmt.Printf("  ✓ updated to %s\n", result.Latest)
			return nil
		},
	}
}

// newTokenFunc creates a git.TokenFunc that generates GitHub App installation
// tokens using the ghinstallation library.
func newTokenFunc(appID int64, privateKeyPath string) git.TokenFunc {
	return func(ctx context.Context, installID int64) (string, error) {
		transport, err := ghinstall.NewKeyFromFile(
			http.DefaultTransport,
			appID,
			installID,
			privateKeyPath,
		)
		if err != nil {
			return "", fmt.Errorf("creating github transport: %w", err)
		}
		return transport.Token(ctx)
	}
}

func setupLogging(level string) {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	slog.SetDefault(slog.New(handler))
}
