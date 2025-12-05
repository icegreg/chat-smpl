package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID        string
	Username  string
	AvatarURL *string
}

func main() {
	var (
		dbURL       = flag.String("db", "postgres://chatapp:secret@localhost:5435/chatapp?sslmode=disable", "Database URL")
		storagePath = flag.String("storage", "", "Path to files storage (required)")
		baseURL     = flag.String("base-url", "/avatars", "Base URL for avatar URLs in database")
		dryRun      = flag.Bool("dry-run", false, "Don't make any changes, just show what would be done")
	)
	flag.Parse()

	if *storagePath == "" {
		fmt.Println("Error: -storage is required")
		fmt.Println("Usage: avatargen -storage /path/to/storage")
		os.Exit(1)
	}

	ctx := context.Background()

	// Connect to database
	pool, err := pgxpool.New(ctx, *dbURL)
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Create avatars directory
	avatarsDir := filepath.Join(*storagePath, "avatars")
	if !*dryRun {
		if err := os.MkdirAll(avatarsDir, 0755); err != nil {
			fmt.Printf("Failed to create avatars directory: %v\n", err)
			os.Exit(1)
		}
	}
	fmt.Printf("Avatars directory: %s\n", avatarsDir)

	// Get all users with cataas.com avatar URLs
	rows, err := pool.Query(ctx, `
		SELECT id, username, avatar_url
		FROM con_test.users
		WHERE avatar_url IS NOT NULL AND avatar_url LIKE 'https://cataas.com%'
	`)
	if err != nil {
		fmt.Printf("Failed to query users: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.AvatarURL); err != nil {
			fmt.Printf("Failed to scan user: %v\n", err)
			continue
		}
		users = append(users, u)
	}

	fmt.Printf("Found %d users with cataas.com avatars\n", len(users))

	if *dryRun {
		fmt.Println("Dry run mode - not making any changes")
		for _, u := range users {
			fmt.Printf("  Would download avatar for %s (%s)\n", u.Username, u.ID)
		}
		return
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	successCount := 0
	failCount := 0

	for i, u := range users {
		fmt.Printf("[%d/%d] Downloading avatar for %s... ", i+1, len(users), u.Username)

		// Download from cataas.com
		resp, err := client.Get(*u.AvatarURL)
		if err != nil {
			fmt.Printf("FAILED (download): %v\n", err)
			failCount++
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			fmt.Printf("FAILED (status %d)\n", resp.StatusCode)
			failCount++
			continue
		}

		// Save to local file
		avatarPath := filepath.Join(avatarsDir, u.ID+".jpg")
		file, err := os.Create(avatarPath)
		if err != nil {
			resp.Body.Close()
			fmt.Printf("FAILED (create file): %v\n", err)
			failCount++
			continue
		}

		_, err = io.Copy(file, resp.Body)
		resp.Body.Close()
		file.Close()

		if err != nil {
			os.Remove(avatarPath)
			fmt.Printf("FAILED (write): %v\n", err)
			failCount++
			continue
		}

		// Update database
		newURL := fmt.Sprintf("%s/%s", *baseURL, u.ID)
		_, err = pool.Exec(ctx, `UPDATE con_test.users SET avatar_url = $1 WHERE id = $2`, newURL, u.ID)
		if err != nil {
			fmt.Printf("FAILED (db update): %v\n", err)
			failCount++
			continue
		}

		fmt.Println("OK")
		successCount++

		// Small delay to be nice to cataas.com
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Printf("\nDone! Success: %d, Failed: %d\n", successCount, failCount)
}
