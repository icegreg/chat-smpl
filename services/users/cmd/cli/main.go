package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"

	"github.com/icegreg/chat-smpl/pkg/jwt"
	"github.com/icegreg/chat-smpl/pkg/logger"
	"github.com/icegreg/chat-smpl/pkg/postgres"
	"github.com/icegreg/chat-smpl/services/users/internal/model"
	"github.com/icegreg/chat-smpl/services/users/internal/repository"
	"github.com/icegreg/chat-smpl/services/users/internal/service"
)

var (
	databaseURL string
	jwtSecret   string
)

func main() {
	logger.InitDefault()

	rootCmd := &cobra.Command{
		Use:   "users-cli",
		Short: "CLI tool for managing users",
		Long:  "A command line tool for managing users in the chat application",
	}

	rootCmd.PersistentFlags().StringVar(&databaseURL, "database-url", getEnv("DATABASE_URL", "postgres://chatapp:secret@localhost:5432/chatapp?sslmode=disable"), "Database connection URL")
	rootCmd.PersistentFlags().StringVar(&jwtSecret, "jwt-secret", getEnv("JWT_SECRET", "your-super-secret-jwt-key"), "JWT secret key")

	rootCmd.AddCommand(userCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getService() (service.UserService, func(), error) {
	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, postgres.DefaultConfig(databaseURL))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	jwtManager := jwt.NewManager(jwt.DefaultConfig(jwtSecret))
	userRepo := repository.NewUserRepository(pool)
	userService := service.NewUserService(userRepo, jwtManager)

	cleanup := func() {
		postgres.Close(pool)
	}

	return userService, cleanup, nil
}

func userCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "User management commands",
	}

	cmd.AddCommand(addUserCmd())
	cmd.AddCommand(deleteUserCmd())
	cmd.AddCommand(listUsersCmd())
	cmd.AddCommand(setRoleCmd())
	cmd.AddCommand(getUserCmd())
	cmd.AddCommand(generateAvatarsCmd())

	return cmd
}

func addUserCmd() *cobra.Command {
	var (
		username string
		email    string
		password string
		role     string
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new user",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, cleanup, err := getService()
			if err != nil {
				return err
			}
			defer cleanup()

			r := model.Role(role)
			if !r.IsValid() {
				return fmt.Errorf("invalid role: %s (must be one of: owner, moderator, user, guest)", role)
			}

			user, err := svc.Create(cmd.Context(), model.CreateUserRequest{
				Username: username,
				Email:    email,
				Password: password,
				Role:     r,
			})
			if err != nil {
				return fmt.Errorf("failed to create user: %w", err)
			}

			fmt.Printf("User created successfully:\n")
			fmt.Printf("  ID:       %s\n", user.ID)
			fmt.Printf("  Username: %s\n", user.Username)
			fmt.Printf("  Email:    %s\n", user.Email)
			fmt.Printf("  Role:     %s\n", user.Role)

			return nil
		},
	}

	cmd.Flags().StringVar(&username, "username", "", "Username (required)")
	cmd.Flags().StringVar(&email, "email", "", "Email address (required)")
	cmd.Flags().StringVar(&password, "password", "", "Password (required)")
	cmd.Flags().StringVar(&role, "role", "user", "User role (owner, moderator, user, guest)")

	cmd.MarkFlagRequired("username")
	cmd.MarkFlagRequired("email")
	cmd.MarkFlagRequired("password")

	return cmd
}

func deleteUserCmd() *cobra.Command {
	var id string

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a user",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, cleanup, err := getService()
			if err != nil {
				return err
			}
			defer cleanup()

			userID, err := uuid.Parse(id)
			if err != nil {
				return fmt.Errorf("invalid user ID: %w", err)
			}

			if err := svc.Delete(cmd.Context(), userID); err != nil {
				return fmt.Errorf("failed to delete user: %w", err)
			}

			fmt.Printf("User %s deleted successfully\n", id)
			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "User ID (required)")
	cmd.MarkFlagRequired("id")

	return cmd
}

func listUsersCmd() *cobra.Command {
	var (
		page  int
		count int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all users",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, cleanup, err := getService()
			if err != nil {
				return err
			}
			defer cleanup()

			resp, err := svc.List(cmd.Context(), page, count)
			if err != nil {
				return fmt.Errorf("failed to list users: %w", err)
			}

			fmt.Printf("Users (page %d/%d, total: %d):\n\n", resp.Pagination.Page, resp.Pagination.TotalPages, resp.Pagination.Total)

			for _, user := range resp.Data {
				fmt.Printf("  ID:       %s\n", user.ID)
				fmt.Printf("  Username: %s\n", user.Username)
				fmt.Printf("  Email:    %s\n", user.Email)
				fmt.Printf("  Role:     %s\n", user.Role)
				fmt.Printf("  Created:  %s\n", user.CreatedAt.Format("2006-01-02 15:04:05"))
				fmt.Println()
			}

			if len(resp.Data) == 0 {
				fmt.Println("  No users found")
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&count, "count", 20, "Items per page")

	return cmd
}

func setRoleCmd() *cobra.Command {
	var (
		id   string
		role string
	)

	cmd := &cobra.Command{
		Use:   "set-role",
		Short: "Set user role",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, cleanup, err := getService()
			if err != nil {
				return err
			}
			defer cleanup()

			userID, err := uuid.Parse(id)
			if err != nil {
				return fmt.Errorf("invalid user ID: %w", err)
			}

			r := model.Role(role)
			if !r.IsValid() {
				return fmt.Errorf("invalid role: %s (must be one of: owner, moderator, user, guest)", role)
			}

			user, err := svc.UpdateRole(cmd.Context(), userID, r)
			if err != nil {
				return fmt.Errorf("failed to update role: %w", err)
			}

			fmt.Printf("User role updated:\n")
			fmt.Printf("  ID:       %s\n", user.ID)
			fmt.Printf("  Username: %s\n", user.Username)
			fmt.Printf("  Role:     %s\n", user.Role)

			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "User ID (required)")
	cmd.Flags().StringVar(&role, "role", "", "New role (required)")
	cmd.MarkFlagRequired("id")
	cmd.MarkFlagRequired("role")

	return cmd
}

func getUserCmd() *cobra.Command {
	var id string

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get user by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, cleanup, err := getService()
			if err != nil {
				return err
			}
			defer cleanup()

			userID, err := uuid.Parse(id)
			if err != nil {
				return fmt.Errorf("invalid user ID: %w", err)
			}

			user, err := svc.GetByID(cmd.Context(), userID)
			if err != nil {
				return fmt.Errorf("failed to get user: %w", err)
			}

			fmt.Printf("User:\n")
			fmt.Printf("  ID:       %s\n", user.ID)
			fmt.Printf("  Username: %s\n", user.Username)
			fmt.Printf("  Email:    %s\n", user.Email)
			fmt.Printf("  Role:     %s\n", user.Role)
			fmt.Printf("  Created:  %s\n", user.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("  Updated:  %s\n", user.UpdatedAt.Format("2006-01-02 15:04:05"))

			return nil
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "User ID (required)")
	cmd.MarkFlagRequired("id")

	return cmd
}

// generateCatAvatarURL generates a random cat avatar URL based on user ID
func generateCatAvatarURL(userID uuid.UUID) string {
	seed := strings.ReplaceAll(userID.String(), "-", "")[:8]
	return fmt.Sprintf("https://cataas.com/cat?width=128&height=128&%s", seed)
}

func generateAvatarsCmd() *cobra.Command {
	var overwrite bool

	cmd := &cobra.Command{
		Use:   "generate-avatars",
		Short: "Generate cat avatars for all users without avatar",
		Long:  "Generate random cat avatar URLs for all users. Use --overwrite to regenerate for users who already have avatars.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			pool, err := postgres.NewPool(ctx, postgres.DefaultConfig(databaseURL))
			if err != nil {
				return fmt.Errorf("failed to connect to database: %w", err)
			}
			defer postgres.Close(pool)

			return generateAvatars(ctx, pool, overwrite)
		},
	}

	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "Overwrite existing avatars")

	return cmd
}

func generateAvatars(ctx context.Context, pool *pgxpool.Pool, overwrite bool) error {
	// Select users that need avatar update
	var query string
	if overwrite {
		query = `SELECT id, username FROM con_test.users`
	} else {
		query = `SELECT id, username FROM con_test.users WHERE avatar_url IS NULL OR avatar_url = ''`
	}

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	type userInfo struct {
		ID       uuid.UUID
		Username string
	}

	var users []userInfo
	for rows.Next() {
		var u userInfo
		if err := rows.Scan(&u.ID, &u.Username); err != nil {
			return fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, u)
	}

	if len(users) == 0 {
		fmt.Println("No users to update")
		return nil
	}

	fmt.Printf("Updating avatars for %d users...\n", len(users))

	updateQuery := `UPDATE con_test.users SET avatar_url = $2, updated_at = NOW() WHERE id = $1`
	updated := 0

	for _, u := range users {
		avatarURL := generateCatAvatarURL(u.ID)
		_, err := pool.Exec(ctx, updateQuery, u.ID, avatarURL)
		if err != nil {
			fmt.Printf("  Failed to update %s: %v\n", u.Username, err)
			continue
		}
		updated++
		fmt.Printf("  Updated %s: %s\n", u.Username, avatarURL)
	}

	fmt.Printf("\nDone! Updated %d/%d users\n", updated, len(users))
	return nil
}
