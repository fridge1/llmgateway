package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// ImageGeneration represents an image generation record.
type ImageGeneration struct {
	ID         int       `json:"id"`
	SessionID  int       `json:"session_id"`
	UserID     string    `json:"user_id"`
	Model      string    `json:"model"`
	Prompt     string    `json:"prompt"`
	Size       string    `json:"size"`
	ImageCount int       `json:"image_count"`
	ImageURLs  []string  `json:"image_urls"`
	Cost       float64   `json:"cost"`
	CreatedAt  time.Time `json:"created_at"`
}

// CreateImageGeneration creates a new image generation record.
func (s *PgStore) CreateImageGeneration(ctx context.Context, gen *ImageGeneration) (*ImageGeneration, error) {
	// 将 ImageURLs 转换为 JSON
	imageURLsJSON, err := json.Marshal(gen.ImageURLs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal image URLs: %w", err)
	}

	query := `
		INSERT INTO image_generations (session_id, user_id, model, prompt, size, image_count, image_urls, cost, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		RETURNING id, created_at
	`

	err = s.db.QueryRowContext(ctx, query,
		gen.SessionID,
		gen.UserID,
		gen.Model,
		gen.Prompt,
		gen.Size,
		gen.ImageCount,
		imageURLsJSON,
		gen.Cost,
	).Scan(&gen.ID, &gen.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create image generation: %w", err)
	}

	return gen, nil
}

// GetImageGenerations retrieves generation records for a session with pagination.
func (s *PgStore) GetImageGenerations(ctx context.Context, sessionID, limit, offset int) ([]ImageGeneration, error) {
	query := `
		SELECT id, session_id, user_id, model, prompt, size, image_count, image_urls, cost, created_at
		FROM image_generations
		WHERE session_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.QueryContext(ctx, query, sessionID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query image generations: %w", err)
	}
	defer rows.Close()

	var generations []ImageGeneration
	for rows.Next() {
		var gen ImageGeneration
		var imageURLsJSON []byte

		if err := rows.Scan(
			&gen.ID,
			&gen.SessionID,
			&gen.UserID,
			&gen.Model,
			&gen.Prompt,
			&gen.Size,
			&gen.ImageCount,
			&imageURLsJSON,
			&gen.Cost,
			&gen.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan image generation: %w", err)
		}

		// 解析 JSON 数组
		if err := json.Unmarshal(imageURLsJSON, &gen.ImageURLs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal image URLs: %w", err)
		}

		generations = append(generations, gen)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return generations, nil
}

// GetImageGeneration retrieves a single generation record by ID.
func (s *PgStore) GetImageGeneration(ctx context.Context, genID int) (*ImageGeneration, error) {
	query := `
		SELECT id, session_id, user_id, model, prompt, size, image_count, image_urls, cost, created_at
		FROM image_generations
		WHERE id = $1
	`

	var gen ImageGeneration
	var imageURLsJSON []byte

	err := s.db.QueryRowContext(ctx, query, genID).Scan(
		&gen.ID,
		&gen.SessionID,
		&gen.UserID,
		&gen.Model,
		&gen.Prompt,
		&gen.Size,
		&gen.ImageCount,
		&imageURLsJSON,
		&gen.Cost,
		&gen.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("generation not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get image generation: %w", err)
	}

	// 解析 JSON 数组
	if err := json.Unmarshal(imageURLsJSON, &gen.ImageURLs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal image URLs: %w", err)
	}

	return &gen, nil
}
