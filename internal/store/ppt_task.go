package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// PptTask represents an async PPT generation task.
type PptTask struct {
	ID               int        `json:"id"`
	UserID           string     `json:"user_id"`
	Status           string     `json:"status"`
	Phase            string     `json:"phase"`
	Topic            string     `json:"topic"`
	SlideCount       int        `json:"slide_count"`
	Language         string     `json:"language"`
	Theme            string     `json:"theme"`
	Audience         string     `json:"audience"`
	Tone             string     `json:"tone"`
	Purpose          string     `json:"purpose"`
	Model            string     `json:"model"`
	OutlineOnly      bool       `json:"outline_only"`
	GenerateImages   bool       `json:"generate_images"`
	ContextText      string     `json:"context_text,omitempty"`
	BriefDocument    []byte          `json:"-"`
	StoryArc         []byte          `json:"story_arc,omitempty"`
	SlideBlueprints  []byte          `json:"-"`
	PresentationJSON json.RawMessage `json:"presentation_json,omitempty"`
	TotalTokens      int        `json:"total_tokens"`
	Cost             float64    `json:"cost"`
	ImageCost        float64    `json:"image_cost"`
	ErrorMessage     string     `json:"error_message,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
}

func (s *PgStore) CreatePptTask(ctx context.Context, task *PptTask) (*PptTask, error) {
	query := `
		INSERT INTO ppt_tasks (user_id, status, phase, topic, slide_count, language, theme, audience, tone, purpose, model, outline_only, generate_images, context_text)
		VALUES ($1, 'pending', 'brief_analyst', $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at
	`
	err := s.db.QueryRowContext(ctx, query,
		task.UserID, task.Topic, task.SlideCount, task.Language,
		task.Theme, task.Audience, task.Tone, task.Purpose, task.Model, task.OutlineOnly, task.GenerateImages, task.ContextText,
	).Scan(&task.ID, &task.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create ppt task: %w", err)
	}
	task.Status = "pending"
	task.Phase = "brief_analyst"
	return task, nil
}

func (s *PgStore) GetPptTasks(ctx context.Context, userID string, limit, offset int) ([]PptTask, error) {
	query := `
		SELECT id, user_id, status, phase, topic, slide_count, language, theme,
		       audience, tone, purpose, model, outline_only, generate_images,
		       context_text, presentation_json, total_tokens, cost, image_cost,
		       error_message, created_at, started_at, completed_at
		FROM ppt_tasks
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := s.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query ppt tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]PptTask, 0)
	for rows.Next() {
		var t PptTask
		var errMsg sql.NullString
		var startedAt, completedAt sql.NullTime
		var presentationJSON sql.NullString
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.Status, &t.Phase, &t.Topic, &t.SlideCount,
			&t.Language, &t.Theme, &t.Audience, &t.Tone, &t.Purpose, &t.Model,
			&t.OutlineOnly, &t.GenerateImages, &t.ContextText,
			&presentationJSON, &t.TotalTokens, &t.Cost, &t.ImageCost, &errMsg,
			&t.CreatedAt, &startedAt, &completedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan ppt task: %w", err)
		}
		if errMsg.Valid {
			t.ErrorMessage = errMsg.String
		}
		if startedAt.Valid {
			t.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			t.CompletedAt = &completedAt.Time
		}
		if presentationJSON.Valid {
			t.PresentationJSON = json.RawMessage(presentationJSON.String)
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (s *PgStore) GetPptTask(ctx context.Context, taskID int) (*PptTask, error) {
	query := `
		SELECT id, user_id, status, phase, topic, slide_count, language, theme,
		       audience, tone, purpose, model, outline_only, generate_images, context_text, brief_document, story_arc, slide_blueprints,
		       presentation_json, total_tokens, cost, image_cost, error_message,
		       created_at, started_at, completed_at
		FROM ppt_tasks WHERE id = $1 AND deleted_at IS NULL
	`
	var t PptTask
	var errMsg sql.NullString
	var startedAt, completedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, taskID).Scan(
		&t.ID, &t.UserID, &t.Status, &t.Phase, &t.Topic, &t.SlideCount,
		&t.Language, &t.Theme, &t.Audience, &t.Tone, &t.Purpose, &t.Model,
		&t.OutlineOnly, &t.GenerateImages, &t.ContextText, &t.BriefDocument, &t.StoryArc, &t.SlideBlueprints,
		&t.PresentationJSON, &t.TotalTokens, &t.Cost, &t.ImageCost, &errMsg,
		&t.CreatedAt, &startedAt, &completedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get ppt task: %w", err)
	}
	if errMsg.Valid {
		t.ErrorMessage = errMsg.String
	}
	if startedAt.Valid {
		t.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		t.CompletedAt = &completedAt.Time
	}
	return &t, nil
}

func (s *PgStore) UpdatePptTaskPhase(ctx context.Context, id int, phase, artifactColumn string, artifactJSON []byte) error {
	if artifactColumn != "" && artifactJSON != nil {
		// Whitelist validation to prevent SQL injection
		var query string
		switch artifactColumn {
		case "artifact_url":
			query = `UPDATE ppt_tasks SET phase = $1, artifact_url = $2 WHERE id = $3`
		case "artifact_data":
			query = `UPDATE ppt_tasks SET phase = $1, artifact_data = $2 WHERE id = $3`
		default:
			return fmt.Errorf("invalid artifact column: %s (must be 'artifact_url' or 'artifact_data')", artifactColumn)
		}
		_, err := s.db.ExecContext(ctx, query, phase, artifactJSON, id)
		return err
	}
	query := `UPDATE ppt_tasks SET phase = $1 WHERE id = $2`
	_, err := s.db.ExecContext(ctx, query, phase, id)
	return err
}

func (s *PgStore) CompletePptTask(ctx context.Context, id int, presentationJSON []byte, totalTokens int, cost float64) error {
	query := `
		UPDATE ppt_tasks
		SET status = 'completed', presentation_json = $1, total_tokens = $2, cost = $3, completed_at = NOW()
		WHERE id = $4
	`
	_, err := s.db.ExecContext(ctx, query, presentationJSON, totalTokens, cost, id)
	return err
}

func (s *PgStore) FailPptTask(ctx context.Context, id int, errMsg string) error {
	query := `
		UPDATE ppt_tasks
		SET status = 'failed', error_message = $1, completed_at = NOW()
		WHERE id = $2
	`
	_, err := s.db.ExecContext(ctx, query, errMsg, id)
	return err
}

func (s *PgStore) ClaimPendingPptTask(ctx context.Context) (*PptTask, error) {
	query := `
		UPDATE ppt_tasks
		SET status = 'processing', started_at = NOW()
		WHERE id = (
			SELECT id FROM ppt_tasks
			WHERE status = 'pending'
			ORDER BY created_at
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, user_id, status, phase, topic, slide_count, language, theme,
		          audience, tone, purpose, model, outline_only, generate_images, context_text, brief_document, story_arc,
		          total_tokens, created_at, started_at
	`
	var t PptTask
	var startedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query).Scan(
		&t.ID, &t.UserID, &t.Status, &t.Phase, &t.Topic, &t.SlideCount,
		&t.Language, &t.Theme, &t.Audience, &t.Tone, &t.Purpose, &t.Model,
		&t.OutlineOnly, &t.GenerateImages, &t.ContextText, &t.BriefDocument, &t.StoryArc,
		&t.TotalTokens, &t.CreatedAt, &startedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to claim ppt task: %w", err)
	}
	if startedAt.Valid {
		t.StartedAt = &startedAt.Time
	}
	return &t, nil
}

func (s *PgStore) RecoverStalePptTasks(ctx context.Context) error {
	query := `
		UPDATE ppt_tasks
		SET status = 'pending', started_at = NULL
		WHERE status = 'processing' AND started_at < NOW() - INTERVAL '10 minutes'
	`
	_, err := s.db.ExecContext(ctx, query)
	return err
}

// PausePptTaskForOutline sets the task status to 'outline_ready' so the user can review the story arc.
func (s *PgStore) PausePptTaskForOutline(ctx context.Context, id int, storyArcJSON []byte, totalTokens int) error {
	query := `
		UPDATE ppt_tasks
		SET status = 'outline_ready', phase = 'outline_review', story_arc = $1, total_tokens = $2
		WHERE id = $3
	`
	_, err := s.db.ExecContext(ctx, query, storyArcJSON, totalTokens, id)
	return err
}

// ConfirmPptTaskOutline moves an outline_ready task back to pending so workers pick it up for Agent 3.
func (s *PgStore) ConfirmPptTaskOutline(ctx context.Context, id int) error {
	query := `
		UPDATE ppt_tasks
		SET status = 'pending', phase = 'info_architect', outline_only = false
		WHERE id = $1 AND status = 'outline_ready'
	`
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to confirm outline: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("task not in outline_ready status")
	}
	return nil
}

// UpdatePptPresentation saves user-edited presentation JSON for a completed task.
// UpdatePptTaskImageCost adds image generation cost to a task.
func (s *PgStore) UpdatePptTaskImageCost(ctx context.Context, id int, imageCost float64) error {
	query := `UPDATE ppt_tasks SET image_cost = $1 WHERE id = $2`
	_, err := s.db.ExecContext(ctx, query, imageCost, id)
	return err
}

func (s *PgStore) UpdatePptPresentation(ctx context.Context, id int, userID string, presJSON []byte) error {
	query := `
		UPDATE ppt_tasks
		SET presentation_json = $1
		WHERE id = $2 AND user_id = $3 AND status = 'completed'
	`
	result, err := s.db.ExecContext(ctx, query, presJSON, id, userID)
	if err != nil {
		return fmt.Errorf("failed to update presentation: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("task not found or not completed")
	}
	return nil
}

// DeletePptTask soft-deletes a PPT task with ownership check.
func (s *PgStore) DeletePptTask(ctx context.Context, id int, userID string) error {
	query := `
		UPDATE ppt_tasks
		SET deleted_at = NOW()
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`
	result, err := s.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete ppt task: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("task not found or already deleted")
	}
	return nil
}
