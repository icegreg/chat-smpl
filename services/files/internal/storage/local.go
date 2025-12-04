package storage

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type Storage interface {
	Save(filename string, reader io.Reader) (string, error)
	Get(path string) (io.ReadCloser, error)
	Delete(path string) error
	Exists(path string) bool
}

type LocalStorage struct {
	basePath string
}

func NewLocalStorage(basePath string) (*LocalStorage, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &LocalStorage{
		basePath: basePath,
	}, nil
}

func (s *LocalStorage) Save(filename string, reader io.Reader) (string, error) {
	// Generate unique filename with date-based directory structure
	now := time.Now()
	dateDir := filepath.Join(s.basePath, now.Format("2006/01/02"))

	if err := os.MkdirAll(dateDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create date directory: %w", err)
	}

	// Generate random prefix to avoid collisions
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	randomPrefix := hex.EncodeToString(randomBytes)

	// Clean filename
	cleanFilename := filepath.Base(filename)
	ext := filepath.Ext(cleanFilename)
	name := cleanFilename[:len(cleanFilename)-len(ext)]

	// Create unique filename
	uniqueFilename := fmt.Sprintf("%s_%s%s", randomPrefix, name, ext)
	fullPath := filepath.Join(dateDir, uniqueFilename)

	// Create file
	file, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy content
	if _, err := io.Copy(file, reader); err != nil {
		os.Remove(fullPath)
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// Return relative path from base
	relPath, err := filepath.Rel(s.basePath, fullPath)
	if err != nil {
		return fullPath, nil
	}

	return relPath, nil
}

func (s *LocalStorage) Get(path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.basePath, path)

	// Security check: ensure path is within base directory
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	absBase, err := filepath.Abs(s.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute base path: %w", err)
	}

	if len(absPath) < len(absBase) || absPath[:len(absBase)] != absBase {
		return nil, fmt.Errorf("path traversal attempt detected")
	}

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

func (s *LocalStorage) Delete(path string) error {
	fullPath := filepath.Join(s.basePath, path)

	// Security check
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	absBase, err := filepath.Abs(s.basePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute base path: %w", err)
	}

	if len(absPath) < len(absBase) || absPath[:len(absBase)] != absBase {
		return fmt.Errorf("path traversal attempt detected")
	}

	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

func (s *LocalStorage) Exists(path string) bool {
	fullPath := filepath.Join(s.basePath, path)
	_, err := os.Stat(fullPath)
	return err == nil
}

func (s *LocalStorage) GetFullPath(path string) string {
	return filepath.Join(s.basePath, path)
}
