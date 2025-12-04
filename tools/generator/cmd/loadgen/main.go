package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/icegreg/chat-smpl/tools/generator/pkg/client"
)

const (
	DefaultPassword = "Test123!"
	DefaultBaseURL  = "http://localhost:3001"
)

type Config struct {
	BaseURL        string
	UsersFile      string
	ChatsFile      string
	UserCount      int
	Duration       time.Duration
	MessageDelay   time.Duration
	ReadRatio      float64 // Ratio of read operations (0.0-1.0)
	AutoGenUsers   bool
	UserPrefix     string
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

type Stats struct {
	MessagesSent   int64
	MessagesRead   int64
	SendErrors     int64
	ReadErrors     int64
	LoginErrors    int64
	StartTime      time.Time
}

type UserSession struct {
	User  GeneratedUser
	Token string
	Chats []string // Chat IDs this user participates in
}

func main() {
	cfg := parseFlags()

	fmt.Printf("Load Generator\n")
	fmt.Printf("==============\n")
	fmt.Printf("Base URL:      %s\n", cfg.BaseURL)
	fmt.Printf("Users file:    %s\n", cfg.UsersFile)
	fmt.Printf("Chats file:    %s\n", cfg.ChatsFile)
	fmt.Printf("User count:    %d\n", cfg.UserCount)
	fmt.Printf("Duration:      %v\n", cfg.Duration)
	fmt.Printf("Message delay: %v\n", cfg.MessageDelay)
	fmt.Printf("Read ratio:    %.0f%%\n", cfg.ReadRatio*100)
	fmt.Printf("Auto-gen:      %v\n", cfg.AutoGenUsers)
	fmt.Printf("\n")

	c := client.New(cfg.BaseURL)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var users []GeneratedUser
	var chats []GeneratedChat
	var err error

	// Load or generate users
	if cfg.AutoGenUsers {
		fmt.Printf("Generating %d users...\n", cfg.UserCount)
		users, err = generateUsers(ctx, c, cfg.UserCount, cfg.UserPrefix)
		if err != nil {
			fmt.Printf("Error generating users: %v\n", err)
			os.Exit(1)
		}
	} else {
		users, err = loadUsers(cfg.UsersFile)
		if err != nil {
			fmt.Printf("Error loading users: %v\n", err)
			os.Exit(1)
		}
		if len(users) > cfg.UserCount {
			users = users[:cfg.UserCount]
		}
	}

	fmt.Printf("Using %d users\n", len(users))

	// Load or generate chats
	if cfg.ChatsFile != "" {
		chats, err = loadChats(cfg.ChatsFile)
		if err != nil {
			fmt.Printf("Warning: could not load chats file: %v\n", err)
			fmt.Printf("Will create a shared chat for all users\n")
		}
	}

	// Login users and build sessions
	fmt.Printf("\nLogging in users...\n")
	sessions := make([]UserSession, 0, len(users))
	var loginErrors int64

	for i, u := range users {
		resp, err := c.Login(ctx, client.LoginRequest{
			Email:    u.Email,
			Password: u.Password,
		})
		if err != nil {
			atomic.AddInt64(&loginErrors, 1)
			continue
		}

		session := UserSession{
			User:  u,
			Token: resp.AccessToken,
			Chats: []string{},
		}

		// Find chats this user participates in
		for _, chat := range chats {
			for _, pid := range chat.Participants {
				if pid == u.ID {
					session.Chats = append(session.Chats, chat.ID)
					break
				}
			}
		}

		sessions = append(sessions, session)

		if (i+1)%100 == 0 {
			fmt.Printf("Logged in %d/%d users\n", i+1, len(users))
		}
	}

	fmt.Printf("Logged in %d users (errors: %d)\n", len(sessions), loginErrors)

	if len(sessions) == 0 {
		fmt.Printf("Error: no users could be logged in\n")
		os.Exit(1)
	}

	// Create a shared chat if no chats exist
	if len(chats) == 0 {
		fmt.Printf("\nCreating shared load test chat...\n")
		participantIDs := make([]string, 0, len(sessions))
		for _, s := range sessions {
			participantIDs = append(participantIDs, s.User.ID)
		}

		chat, err := c.CreateChat(ctx, sessions[0].Token, client.CreateChatRequest{
			Type:           "group",
			Name:           fmt.Sprintf("Load Test Chat %d", time.Now().Unix()),
			ParticipantIDs: participantIDs,
		})
		if err != nil {
			fmt.Printf("Error creating shared chat: %v\n", err)
			os.Exit(1)
		}

		// Update all sessions to include this chat
		for i := range sessions {
			sessions[i].Chats = []string{chat.ID}
		}

		fmt.Printf("Created shared chat: %s\n", chat.ID)
	}

	// Start load test
	fmt.Printf("\nStarting load test...\n")
	fmt.Printf("Press Ctrl+C to stop\n\n")

	stats := &Stats{
		StartTime: time.Now(),
	}

	var wg sync.WaitGroup
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Start workers (one per user session)
	for i := range sessions {
		wg.Add(1)
		go func(session *UserSession, workerRng *rand.Rand) {
			defer wg.Done()
			runWorker(ctx, c, session, cfg, stats, workerRng)
		}(&sessions[i], rand.New(rand.NewSource(rng.Int63())))
	}

	// Stats reporter
	statsDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				printStats(stats)
			case <-statsDone:
				return
			}
		}
	}()

	// Wait for duration or signal
	select {
	case <-sigChan:
		fmt.Printf("\nReceived shutdown signal...\n")
	case <-time.After(cfg.Duration):
		fmt.Printf("\nDuration reached...\n")
	}

	cancel()
	close(statsDone)
	wg.Wait()

	// Final stats
	fmt.Printf("\n")
	fmt.Printf("Final Results\n")
	fmt.Printf("=============\n")
	printStats(stats)
}

func runWorker(ctx context.Context, c *client.Client, session *UserSession, cfg Config, stats *Stats, rng *rand.Rand) {
	if len(session.Chats) == 0 {
		return
	}

	messages := []string{
		"Hello everyone!",
		"How's it going?",
		"Just checking in...",
		"Anyone there?",
		"Testing the chat system",
		"Lorem ipsum dolor sit amet",
		"The quick brown fox jumps over the lazy dog",
		"Load testing in progress",
		"Message from user " + session.User.Username,
		fmt.Sprintf("Timestamp: %d", time.Now().UnixNano()),
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Pick a random chat
		chatID := session.Chats[rng.Intn(len(session.Chats))]

		// Decide whether to read or write based on ratio
		if rng.Float64() < cfg.ReadRatio {
			// Read messages
			_, err := c.GetMessages(ctx, session.Token, chatID, 1, 20)
			if err != nil {
				if ctx.Err() == nil {
					atomic.AddInt64(&stats.ReadErrors, 1)
				}
			} else {
				atomic.AddInt64(&stats.MessagesRead, 1)
			}
		} else {
			// Send message
			content := messages[rng.Intn(len(messages))]
			_, err := c.SendMessage(ctx, session.Token, chatID, client.SendMessageRequest{
				Content: content,
			})
			if err != nil {
				if ctx.Err() == nil {
					atomic.AddInt64(&stats.SendErrors, 1)
				}
			} else {
				atomic.AddInt64(&stats.MessagesSent, 1)
			}
		}

		// Wait before next operation
		select {
		case <-ctx.Done():
			return
		case <-time.After(cfg.MessageDelay):
		}
	}
}

func printStats(stats *Stats) {
	elapsed := time.Since(stats.StartTime)
	sent := atomic.LoadInt64(&stats.MessagesSent)
	read := atomic.LoadInt64(&stats.MessagesRead)
	sendErr := atomic.LoadInt64(&stats.SendErrors)
	readErr := atomic.LoadInt64(&stats.ReadErrors)

	totalOps := sent + read
	var opsPerSec float64
	if elapsed.Seconds() > 0 {
		opsPerSec = float64(totalOps) / elapsed.Seconds()
	}

	fmt.Printf("[%v] Sent: %d | Read: %d | Total: %d | Rate: %.2f ops/sec | Errors: send=%d read=%d\n",
		elapsed.Truncate(time.Second), sent, read, totalOps, opsPerSec, sendErr, readErr)
}

func generateUsers(ctx context.Context, c *client.Client, count int, prefix string) ([]GeneratedUser, error) {
	users := make([]GeneratedUser, 0, count)
	var mu sync.Mutex
	var wg sync.WaitGroup

	jobs := make(chan int, count)

	// Use 10 workers
	for w := 0; w < 10; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				username := fmt.Sprintf("%s_%d", prefix, i)
				email := fmt.Sprintf("%s_%d@loadtest.local", prefix, i)
				displayName := fmt.Sprintf("Load Test User %d", i)

				req := client.RegisterRequest{
					Username:    username,
					Email:       email,
					Password:    DefaultPassword,
					DisplayName: displayName,
				}

				resp, err := c.Register(ctx, req)
				if err != nil {
					continue
				}

				user := GeneratedUser{
					ID:          resp.User.ID,
					Username:    resp.User.Username,
					Email:       resp.User.Email,
					DisplayName: resp.User.DisplayName,
					Password:    DefaultPassword,
				}

				mu.Lock()
				users = append(users, user)
				mu.Unlock()

				if len(users)%100 == 0 {
					fmt.Printf("Generated %d/%d users\n", len(users), count)
				}
			}
		}()
	}

	for i := 1; i <= count; i++ {
		jobs <- i
	}
	close(jobs)

	wg.Wait()

	return users, nil
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

func loadChats(filename string) ([]GeneratedChat, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var chats []GeneratedChat
	if err := json.Unmarshal(data, &chats); err != nil {
		return nil, err
	}

	return chats, nil
}

func parseFlags() Config {
	cfg := Config{}

	flag.StringVar(&cfg.BaseURL, "url", getEnv("GENERATOR_BASE_URL", DefaultBaseURL), "Base URL of the API")
	flag.StringVar(&cfg.UsersFile, "users", "users.json", "Users JSON file (ignored if -auto-gen)")
	flag.StringVar(&cfg.ChatsFile, "chats", "", "Chats JSON file (optional, creates shared chat if empty)")
	flag.IntVar(&cfg.UserCount, "user-count", 1000, "Number of users to use (default 1000)")
	flag.DurationVar(&cfg.Duration, "duration", 100*time.Minute, "Test duration")
	flag.DurationVar(&cfg.MessageDelay, "delay", 100*time.Millisecond, "Delay between operations per user")
	flag.Float64Var(&cfg.ReadRatio, "read-ratio", 0.7, "Ratio of read operations (0.0-1.0)")
	flag.BoolVar(&cfg.AutoGenUsers, "auto-gen", false, "Auto-generate users instead of loading from file")
	flag.StringVar(&cfg.UserPrefix, "user-prefix", "loadtest", "Username prefix for auto-generated users")

	flag.Parse()

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
