package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/icegreg/chat-smpl/tools/generator/pkg/client"
)

const (
	DefaultPassword = "Test123!"
	DefaultBaseURL  = "http://localhost:3001"
)

type Config struct {
	BaseURL     string
	Count       int
	Prefix      string
	Password    string
	Concurrency int
	OutputFile  string
}

type GeneratedUser struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

func main() {
	cfg := parseFlags()

	fmt.Printf("User Generator\n")
	fmt.Printf("==============\n")
	fmt.Printf("Base URL:    %s\n", cfg.BaseURL)
	fmt.Printf("Count:       %d\n", cfg.Count)
	fmt.Printf("Prefix:      %s\n", cfg.Prefix)
	fmt.Printf("Concurrency: %d\n", cfg.Concurrency)
	fmt.Printf("Output:      %s\n", cfg.OutputFile)
	fmt.Printf("\n")

	c := client.New(cfg.BaseURL)
	ctx := context.Background()

	users := make([]GeneratedUser, 0, cfg.Count)
	var mu sync.Mutex
	var successCount, errorCount int64

	// Create worker pool
	jobs := make(chan int, cfg.Count)
	var wg sync.WaitGroup

	startTime := time.Now()

	// Start workers
	for w := 0; w < cfg.Concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				username := fmt.Sprintf("%s_%d", cfg.Prefix, i)
				email := fmt.Sprintf("%s_%d@test.local", cfg.Prefix, i)
				displayName := fmt.Sprintf("Test User %d", i)

				req := client.RegisterRequest{
					Username:    username,
					Email:       email,
					Password:    cfg.Password,
					DisplayName: displayName,
				}

				resp, err := c.Register(ctx, req)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					fmt.Printf("Error creating user %s: %v\n", username, err)
					continue
				}

				atomic.AddInt64(&successCount, 1)

				user := GeneratedUser{
					ID:          resp.User.ID,
					Username:    resp.User.Username,
					Email:       resp.User.Email,
					DisplayName: resp.User.DisplayName,
					Password:    cfg.Password,
				}

				mu.Lock()
				users = append(users, user)
				mu.Unlock()

				if successCount%100 == 0 {
					fmt.Printf("Progress: %d/%d users created\n", successCount, cfg.Count)
				}
			}
		}()
	}

	// Send jobs
	for i := 1; i <= cfg.Count; i++ {
		jobs <- i
	}
	close(jobs)

	// Wait for completion
	wg.Wait()

	elapsed := time.Since(startTime)

	fmt.Printf("\n")
	fmt.Printf("Results\n")
	fmt.Printf("=======\n")
	fmt.Printf("Success: %d\n", successCount)
	fmt.Printf("Errors:  %d\n", errorCount)
	fmt.Printf("Time:    %v\n", elapsed)
	fmt.Printf("Rate:    %.2f users/sec\n", float64(successCount)/elapsed.Seconds())

	// Save to file
	if cfg.OutputFile != "" && len(users) > 0 {
		data, err := json.MarshalIndent(users, "", "  ")
		if err != nil {
			fmt.Printf("Error marshaling users: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(cfg.OutputFile, data, 0644); err != nil {
			fmt.Printf("Error writing file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\nUsers saved to: %s\n", cfg.OutputFile)
	}
}

func parseFlags() Config {
	cfg := Config{}

	flag.StringVar(&cfg.BaseURL, "url", getEnv("GENERATOR_BASE_URL", DefaultBaseURL), "Base URL of the API")
	flag.IntVar(&cfg.Count, "count", 1000, "Number of users to generate")
	flag.StringVar(&cfg.Prefix, "prefix", "loadtest", "Username prefix")
	flag.StringVar(&cfg.Password, "password", DefaultPassword, "Password for all users")
	flag.IntVar(&cfg.Concurrency, "concurrency", 10, "Number of concurrent workers")
	flag.StringVar(&cfg.OutputFile, "output", "users.json", "Output file for generated users")

	flag.Parse()

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
