package main

import (
	"context"
	"os"
	"strconv"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/icegreg/chat-smpl/pkg/logger"
	"github.com/icegreg/chat-smpl/pkg/postgres"
	filesPb "github.com/icegreg/chat-smpl/proto/files"
	"github.com/icegreg/chat-smpl/services/chat/internal/repository"
)

type Config struct {
	DatabaseURL       string
	FilesServiceAddr  string
	RetentionDays     int
	BatchSize         int
	DryRun            bool
}

func loadConfig() Config {
	retentionDays := 30
	if days := os.Getenv("MESSAGE_RETENTION_DAYS"); days != "" {
		if d, err := strconv.Atoi(days); err == nil && d > 0 {
			retentionDays = d
		}
	}

	batchSize := 100
	if size := os.Getenv("CLEANUP_BATCH_SIZE"); size != "" {
		if s, err := strconv.Atoi(size); err == nil && s > 0 {
			batchSize = s
		}
	}

	dryRun := os.Getenv("CLEANUP_DRY_RUN") == "true"

	return Config{
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://chatapp:secret@localhost:5432/chatapp?sslmode=disable"),
		FilesServiceAddr: getEnv("FILES_SERVICE_ADDR", "localhost:50053"),
		RetentionDays:    retentionDays,
		BatchSize:        batchSize,
		DryRun:           dryRun,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	logger.InitDefault()
	defer logger.Sync()

	cfg := loadConfig()

	logger.Info("starting message cleanup job",
		zap.Int("retention_days", cfg.RetentionDays),
		zap.Int("batch_size", cfg.BatchSize),
		zap.Bool("dry_run", cfg.DryRun),
	)

	ctx := context.Background()

	// Connect to database
	pool, err := postgres.NewPool(ctx, postgres.DefaultConfig(cfg.DatabaseURL))
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer postgres.Close(pool)

	// Connect to files service
	var filesClient filesPb.FilesServiceClient
	filesConn, err := grpc.NewClient(cfg.FilesServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Warn("failed to connect to files service", zap.Error(err))
	} else {
		defer filesConn.Close()
		filesClient = filesPb.NewFilesServiceClient(filesConn)
		logger.Info("connected to files service", zap.String("addr", cfg.FilesServiceAddr))
	}

	// Initialize repository
	chatRepo := repository.NewChatRepository(pool)

	// Calculate cutoff time
	cutoffTime := time.Now().AddDate(0, 0, -cfg.RetentionDays)
	logger.Info("cleanup cutoff time", zap.Time("cutoff", cutoffTime))

	// Statistics
	var totalMessages, totalFileLinks, totalFilesDeleted int64

	// Process in batches
	for {
		// Get batch of deleted messages older than cutoff
		messageIDs, err := chatRepo.GetDeletedMessagesOlderThan(ctx, cutoffTime, cfg.BatchSize)
		if err != nil {
			logger.Error("failed to get deleted messages", zap.Error(err))
			break
		}

		if len(messageIDs) == 0 {
			logger.Info("no more messages to process")
			break
		}

		logger.Info("processing batch", zap.Int("count", len(messageIDs)))

		for _, msgID := range messageIDs {
			// Get file link IDs for this message
			fileLinkIDs, err := chatRepo.GetFileLinkIDsForMessage(ctx, msgID)
			if err != nil {
				logger.Warn("failed to get file links for message", zap.String("message_id", msgID.String()), zap.Error(err))
			}

			if cfg.DryRun {
				logger.Info("DRY RUN: would delete message",
					zap.String("message_id", msgID.String()),
					zap.Int("file_links", len(fileLinkIDs)),
				)
				totalMessages++
				totalFileLinks += int64(len(fileLinkIDs))
				continue
			}

			// Permanently delete file links through files service
			if filesClient != nil && len(fileLinkIDs) > 0 {
				linkIDStrings := make([]string, len(fileLinkIDs))
				for i, id := range fileLinkIDs {
					linkIDStrings[i] = id.String()
				}

				resp, err := filesClient.PermanentlyDeleteLinks(ctx, &filesPb.PermanentlyDeleteLinksRequest{
					LinkIds: linkIDStrings,
				})
				if err != nil {
					logger.Warn("failed to permanently delete file links",
						zap.String("message_id", msgID.String()),
						zap.Error(err),
					)
				} else {
					totalFileLinks += int64(resp.DeletedLinks)
					totalFilesDeleted += int64(resp.DeletedFiles)
					logger.Debug("deleted file links",
						zap.String("message_id", msgID.String()),
						zap.Int32("links_deleted", resp.DeletedLinks),
						zap.Int32("files_deleted", resp.DeletedFiles),
					)
				}
			}

			// Permanently delete the message
			if err := chatRepo.PermanentlyDeleteMessage(ctx, msgID); err != nil {
				logger.Error("failed to permanently delete message",
					zap.String("message_id", msgID.String()),
					zap.Error(err),
				)
				continue
			}

			totalMessages++
			logger.Debug("permanently deleted message", zap.String("message_id", msgID.String()))
		}

		// Small delay between batches to reduce database load
		time.Sleep(100 * time.Millisecond)
	}

	logger.Info("cleanup job completed",
		zap.Int64("messages_deleted", totalMessages),
		zap.Int64("file_links_deleted", totalFileLinks),
		zap.Int64("files_deleted", totalFilesDeleted),
		zap.Bool("dry_run", cfg.DryRun),
	)
}
