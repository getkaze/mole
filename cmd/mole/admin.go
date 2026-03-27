package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/getkaze/mole/internal/config"
	"github.com/getkaze/mole/internal/store"
)

func adminCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Manage dashboard users and roles",
	}

	cmd.AddCommand(
		adminSetRoleCmd(),
		adminListCmd(),
		adminOptInCmd(),
		adminOptOutCmd(),
	)

	return cmd
}

func adminSetRoleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-role <github-user> <role>",
		Short: "Set a user's dashboard role (dev, tech_lead, architect, manager, admin)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			user, role := args[0], args[1]

			validRoles := map[string]bool{
				"dev": true, "tech_lead": true, "architect": true, "manager": true, "admin": true,
			}
			if !validRoles[role] {
				return fmt.Errorf("invalid role %q — must be one of: dev, tech_lead, architect, manager, admin", role)
			}

			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()

			if err := st.UpsertAccess(context.Background(), &store.DashboardAccess{
				GitHubUser: user,
				Role:       role,
			}); err != nil {
				return fmt.Errorf("setting role: %w", err)
			}

			fmt.Printf("Set %s role to %s\n", user, role)
			return nil
		},
	}
}

func adminListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all dashboard users and their roles",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()

			rows, err := st.DB().QueryContext(context.Background(),
				`SELECT github_user, role, individual_visibility FROM dashboard_access ORDER BY github_user`,
			)
			if err != nil {
				return fmt.Errorf("querying users: %w", err)
			}
			defer rows.Close()

			fmt.Printf("%-25s %-15s %s\n", "USER", "ROLE", "VISIBLE")
			fmt.Printf("%-25s %-15s %s\n", "----", "----", "-------")

			for rows.Next() {
				var user, role string
				var visible bool
				if err := rows.Scan(&user, &role, &visible); err != nil {
					return err
				}
				vis := "no"
				if visible {
					vis = "yes"
				}
				fmt.Printf("%-25s %-15s %s\n", user, role, vis)
			}
			return rows.Err()
		},
	}
}

func adminOptInCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "opt-in <github-user>",
		Short: "Allow tech leads and architects to see this user's individual data",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return setVisibility(args[0], true)
		},
	}
}

func adminOptOutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "opt-out <github-user>",
		Short: "Hide this user's individual data from tech leads and architects",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return setVisibility(args[0], false)
		},
	}
}

func setVisibility(user string, visible bool) error {
	st, err := openStore()
	if err != nil {
		return err
	}
	defer st.Close()

	access, err := st.GetAccess(context.Background(), user)
	if err != nil {
		return fmt.Errorf("user %q not found — they need to log in to the dashboard first", user)
	}

	access.IndividualVisibility = visible
	if err := st.UpsertAccess(context.Background(), access); err != nil {
		return fmt.Errorf("updating visibility: %w", err)
	}

	state := "hidden"
	if visible {
		state = "visible"
	}
	fmt.Printf("Set %s individual data to %s\n", user, state)
	return nil
}

func openStore() (*store.MySQLStore, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}
	st, err := store.NewMySQL(cfg.MySQL.DSN())
	if err != nil {
		return nil, fmt.Errorf("mysql: %w", err)
	}
	return st, nil
}
