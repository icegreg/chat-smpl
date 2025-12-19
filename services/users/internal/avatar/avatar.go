package avatar

import (
	"crypto/md5"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

const (
	// AvatarSize is the size of generated avatars in pixels
	AvatarSize = 128
	// GridSize is the size of the identicon grid (5x5)
	GridSize = 5
	// BlockSize is the size of each block in pixels
	BlockSize = AvatarSize / GridSize
)

// Generator handles avatar generation and storage
type Generator struct {
	basePath string
}

// NewGenerator creates a new avatar generator with the specified storage path
func NewGenerator(basePath string) *Generator {
	return &Generator{basePath: basePath}
}

// EnsureDir creates the avatars directory if it doesn't exist
func (g *Generator) EnsureDir() error {
	return os.MkdirAll(g.basePath, 0755)
}

// GetFilename returns the filename for a user's avatar
func GetFilename(userID uuid.UUID) string {
	return fmt.Sprintf("%s.png", userID.String())
}

// GetPath returns the full path to a user's avatar file
func (g *Generator) GetPath(userID uuid.UUID) string {
	return filepath.Join(g.basePath, GetFilename(userID))
}

// Exists checks if an avatar file exists for the given user
func (g *Generator) Exists(userID uuid.UUID) bool {
	path := g.GetPath(userID)
	_, err := os.Stat(path)
	return err == nil
}

// Generate creates an identicon avatar for the given user ID and saves it to disk
func (g *Generator) Generate(userID uuid.UUID) error {
	if err := g.EnsureDir(); err != nil {
		return fmt.Errorf("failed to create avatars directory: %w", err)
	}

	// Generate the identicon image
	img := generateIdenticon(userID)

	// Save to file
	path := g.GetPath(userID)
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create avatar file: %w", err)
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		return fmt.Errorf("failed to encode avatar: %w", err)
	}

	return nil
}

// Delete removes an avatar file
func (g *Generator) Delete(userID uuid.UUID) error {
	path := g.GetPath(userID)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// generateIdenticon creates an identicon image based on user ID
func generateIdenticon(userID uuid.UUID) image.Image {
	// Create hash from user ID for deterministic generation
	hash := md5.Sum([]byte(userID.String()))

	// Extract colors from hash
	foreground := color.RGBA{
		R: hash[0],
		G: hash[1],
		B: hash[2],
		A: 255,
	}

	// Lighter background based on foreground
	background := color.RGBA{
		R: 240,
		G: 240,
		B: 245,
		A: 255,
	}

	// Create image
	img := image.NewRGBA(image.Rect(0, 0, AvatarSize, AvatarSize))

	// Fill background
	for y := 0; y < AvatarSize; y++ {
		for x := 0; x < AvatarSize; x++ {
			img.Set(x, y, background)
		}
	}

	// Generate symmetric pattern (only need to generate half, then mirror)
	for row := 0; row < GridSize; row++ {
		for col := 0; col <= GridSize/2; col++ {
			// Use hash bytes to determine if cell is filled
			hashIndex := row*GridSize + col
			if hashIndex >= len(hash) {
				hashIndex = hashIndex % len(hash)
			}

			if hash[hashIndex]%2 == 0 {
				// Fill this cell and its mirror
				fillBlock(img, col, row, foreground)
				// Mirror horizontally
				mirrorCol := GridSize - 1 - col
				if mirrorCol != col {
					fillBlock(img, mirrorCol, row, foreground)
				}
			}
		}
	}

	return img
}

// fillBlock fills a grid block with the specified color
func fillBlock(img *image.RGBA, gridX, gridY int, c color.RGBA) {
	startX := gridX * BlockSize
	startY := gridY * BlockSize

	for y := startY; y < startY+BlockSize; y++ {
		for x := startX; x < startX+BlockSize; x++ {
			img.Set(x, y, c)
		}
	}
}
