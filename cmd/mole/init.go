package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/getkaze/mole/internal/config"
	"github.com/getkaze/mole/internal/llm"
	"github.com/getkaze/mole/internal/scan"
)

func initCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init <path>",
		Short: "Analyze a repository and generate .mole/ context files",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoPath, err := filepath.Abs(args[0])
			if err != nil {
				return fmt.Errorf("resolving path: %w", err)
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			dryRun, _ := cmd.Flags().GetBool("dry-run")
			language, _ := cmd.Flags().GetString("language")

			// Step 1: Local scan
			fmt.Printf("Scanning %s...\n", repoPath)
			result, err := scan.Run(repoPath)
			if err != nil {
				return fmt.Errorf("scan: %w", err)
			}

			report := result.Format()
			slog.Debug("scan complete", "language", result.Language, "framework", result.Framework, "samples", len(result.Samples))

			if dryRun {
				fmt.Println("\n--- Scan Report (would be sent to LLM) ---")
				fmt.Println(report)
				return nil
			}

			// Step 2: Generate docs via LLM
			fmt.Println("Generating documentation...")
			provider := llm.NewClaude(cfg.LLM.APIKey)

			raw, err := provider.Generate(cmd.Context(), llm.GenerateRequest{
				System: scan.BuildInitPrompt(language),
				User:   report,
				Model:  cfg.LLM.ReviewModel,
			})
			if err != nil {
				return fmt.Errorf("LLM generate: %w", err)
			}

			output, err := scan.ParseInitResponse(raw)
			if err != nil {
				return fmt.Errorf("parsing LLM response: %w", err)
			}

			// Step 3: Write .mole/ files
			moleDir := filepath.Join(repoPath, ".mole")
			if err := os.MkdirAll(moleDir, 0o755); err != nil {
				return fmt.Errorf("creating .mole/: %w", err)
			}

			cfgLang := "en"
			if language == "pt-BR" || language == "pt" {
				cfgLang = language
			}

			files := map[string]string{
				"architecture.md": output.Architecture,
				"conventions.md":  output.Conventions,
				"config.yaml":    fmt.Sprintf("# Mole per-repository configuration\nlanguage: %s\n", cfgLang),
			}

			for name, content := range files {
				path := filepath.Join(moleDir, name)
				// Don't overwrite config.yaml if it already exists
				if name == "config.yaml" {
					if _, err := os.Stat(path); err == nil {
						fmt.Printf("  skip %s (already exists)\n", name)
						continue
					}
				}
				if err := os.WriteFile(path, []byte(content+"\n"), 0o644); err != nil {
					return fmt.Errorf("writing %s: %w", name, err)
				}
				fmt.Printf("  wrote %s\n", name)
			}

			fmt.Println("\nDone. Review the generated files and commit them to your repo.")
			return nil
		},
	}

	cmd.Flags().Bool("dry-run", false, "show scan report without calling LLM or writing files")
	cmd.Flags().String("language", "en", "language for generated docs (en, pt-BR)")
	return cmd
}
