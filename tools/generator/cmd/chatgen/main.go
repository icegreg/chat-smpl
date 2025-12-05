package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
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
	BaseURL          string
	Count            int
	UsersFile        string
	ParticipantsMin  int
	ParticipantsMax  int
	ChatType         string
	Concurrency      int
	OutputFile       string
}

type GeneratedUser struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

type GeneratedChat struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	ChatType     string   `json:"chat_type"`
	CreatedBy    string   `json:"created_by"`
	Participants []string `json:"participants"`
}

func main() {
	cfg := parseFlags()

	fmt.Printf("Chat Generator\n")
	fmt.Printf("==============\n")
	fmt.Printf("Base URL:     %s\n", cfg.BaseURL)
	fmt.Printf("Count:        %d\n", cfg.Count)
	fmt.Printf("Users file:   %s\n", cfg.UsersFile)
	fmt.Printf("Participants: %d-%d\n", cfg.ParticipantsMin, cfg.ParticipantsMax)
	fmt.Printf("Chat type:    %s\n", cfg.ChatType)
	fmt.Printf("Concurrency:  %d\n", cfg.Concurrency)
	fmt.Printf("Output:       %s\n", cfg.OutputFile)
	fmt.Printf("\n")

	// Load users
	users, err := loadUsers(cfg.UsersFile)
	if err != nil {
		fmt.Printf("Error loading users: %v\n", err)
		os.Exit(1)
	}

	if len(users) < cfg.ParticipantsMin {
		fmt.Printf("Error: not enough users (%d) for minimum participants (%d)\n", len(users), cfg.ParticipantsMin)
		os.Exit(1)
	}

	fmt.Printf("Loaded %d users\n\n", len(users))

	c := client.New(cfg.BaseURL)
	ctx := context.Background()

	// Login all users and cache tokens
	tokens := make(map[string]string)
	fmt.Printf("Logging in users...\n")
	for i, u := range users {
		resp, err := c.Login(ctx, client.LoginRequest{
			Email:    u.Email,
			Password: u.Password,
		})
		if err != nil {
			fmt.Printf("Error logging in user %s: %v\n", u.Username, err)
			continue
		}
		tokens[u.ID] = resp.AccessToken

		if (i+1)%100 == 0 {
			fmt.Printf("Logged in %d/%d users\n", i+1, len(users))
		}
	}

	fmt.Printf("Logged in %d users\n\n", len(tokens))

	chats := make([]GeneratedChat, 0, cfg.Count)
	var mu sync.Mutex
	var successCount, errorCount int64

	// Create worker pool
	jobs := make(chan int, cfg.Count)
	var wg sync.WaitGroup

	startTime := time.Now()
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Start workers
	for w := 0; w < cfg.Concurrency; w++ {
		wg.Add(1)
		go func(workerRng *rand.Rand) {
			defer wg.Done()
			for i := range jobs {
				// Select random creator
				creatorIdx := workerRng.Intn(len(users))
				creator := users[creatorIdx]
				creatorToken := tokens[creator.ID]

				if creatorToken == "" {
					atomic.AddInt64(&errorCount, 1)
					continue
				}

				// Select random participants
				numParticipants := cfg.ParticipantsMin
				if cfg.ParticipantsMax > cfg.ParticipantsMin {
					numParticipants = cfg.ParticipantsMin + workerRng.Intn(cfg.ParticipantsMax-cfg.ParticipantsMin+1)
				}

				participantIDs := make([]string, 0, numParticipants)
				participantIDs = append(participantIDs, creator.ID)

				// Randomly select other participants
				usedIndices := map[int]bool{creatorIdx: true}
				for len(participantIDs) < numParticipants && len(usedIndices) < len(users) {
					idx := workerRng.Intn(len(users))
					if !usedIndices[idx] {
						usedIndices[idx] = true
						participantIDs = append(participantIDs, users[idx].ID)
					}
				}

				chatName := fmt.Sprintf("Chat %d", i)

				req := client.CreateChatRequest{
					Type:           cfg.ChatType,
					Name:           chatName,
					ParticipantIDs: participantIDs,
				}

				resp, err := c.CreateChat(ctx, creatorToken, req)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					fmt.Printf("Error creating chat %s: %v\n", chatName, err)
					continue
				}

				atomic.AddInt64(&successCount, 1)

				chat := GeneratedChat{
					ID:           resp.ID,
					Name:         resp.Name,
					ChatType:     cfg.ChatType,
					CreatedBy:    creator.ID,
					Participants: participantIDs,
				}

				mu.Lock()
				chats = append(chats, chat)
				mu.Unlock()

				if successCount%100 == 0 {
					fmt.Printf("Progress: %d/%d chats created\n", successCount, cfg.Count)
				}
			}
		}(rand.New(rand.NewSource(rng.Int63())))
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
	fmt.Printf("Rate:    %.2f chats/sec\n", float64(successCount)/elapsed.Seconds())

	// Save to file
	if cfg.OutputFile != "" && len(chats) > 0 {
		data, err := json.MarshalIndent(chats, "", "  ")
		if err != nil {
			fmt.Printf("Error marshaling chats: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(cfg.OutputFile, data, 0644); err != nil {
			fmt.Printf("Error writing file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\nChats saved to: %s\n", cfg.OutputFile)
	}
}

func loadUsers(filename string) ([]GeneratedUser, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var users []GeneratedUser
	if err := json.Unmarshal(data, &users); err != nil {
		return nil, err
	}

	return users, nil
}

func parseFlags() Config {
	cfg := Config{}

	flag.StringVar(&cfg.BaseURL, "url", getEnv("GENERATOR_BASE_URL", DefaultBaseURL), "Base URL of the API")
	flag.IntVar(&cfg.Count, "count", 100, "Number of chats to generate")
	flag.StringVar(&cfg.UsersFile, "users", "users.json", "Users JSON file")
	flag.IntVar(&cfg.ParticipantsMin, "participants-min", 2, "Minimum participants per chat")
	flag.IntVar(&cfg.ParticipantsMax, "participants-max", 10, "Maximum participants per chat")
	flag.StringVar(&cfg.ChatType, "type", "group", "Chat type: group, private, channel")
	flag.IntVar(&cfg.Concurrency, "concurrency", 5, "Number of concurrent workers")
	flag.StringVar(&cfg.OutputFile, "output", "chats.json", "Output file for generated chats")

	flag.Parse()

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
