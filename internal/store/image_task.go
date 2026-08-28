package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// ImageTask represents an async image generation/edit task.
type ImageTask struct {
	ID              int            `json:"id"`
	UserID          string         `json:"user_id"`
	Type            string         `json:"type"`
	Status          string         `json:"status"`
	Model           string         `json:"model"`
	Prompt          string         `json:"prompt"`
	Size            string         `json:"size"`
	ImageCount      int            `json:"image_count"`
	Params          map[string]any `json:"params,omitempty"`
	InputImages     []string       `json:"input_images,omitempty"`
	InputMask       []byte         `json:"-"`
	ResultURLs      []string       `json:"result_urls"`
	Cost            float64        `json:"cost"`
	ErrorMessage    string         `json:"error_message,omitempty"`
	ImageShareKeyID *string        `json:"image_share_key_id,omitempty"`
	// Billing identity for async settlement. Empty for legacy/JWT/image-share
	// tasks (which settle by UserID). Populated for API-key public tasks so the
	// detached worker charges the right entity.
	TenantID     string     `json:"-"`
	TenantKeyID  string     `json:"-"`
	SubUserID    string     `json:"-"`
	SubUserKeyID string     `json:"-"`
	APIKeyID     string     `json:"-"`
	CreatedAt    time.Time  `json:"created_at"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

// nullableString returns nil for empty strings so they persist as SQL NULL.
func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (s *PgStore) CreateImageTask(ctx context.Context, task *ImageTask) (*ImageTask, error) {
	var inputImagesJSON []byte
	if len(task.InputImages) > 0 {
		var err error
		inputImagesJSON, err = json.Marshal(task.InputImages)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal input images: %w", err)
		}
	}

	var paramsJSON []byte
	if len(task.Params) > 0 {
		var err error
		paramsJSON, err = json.Marshal(task.Params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
	}

	query := `
		INSERT INTO image_tasks (user_id, type, status, model, prompt, size, image_count, input_images, input_mask, params, image_share_key_id,
		                         tenant_id, tenant_key_id, sub_user_id, sub_user_key_id, api_key_id)
		VALUES ($1, $2, 'pending', $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_at
	`
	err := s.db.QueryRowContext(ctx, query,
		task.UserID, task.Type, task.Model, task.Prompt,
		task.Size, task.ImageCount, inputImagesJSON, task.InputMask, paramsJSON, task.ImageShareKeyID,
		nullableString(task.TenantID), nullableString(task.TenantKeyID),
		nullableString(task.SubUserID), nullableString(task.SubUserKeyID), nullableString(task.APIKeyID),
	).Scan(&task.ID, &task.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create image task: %w", err)
	}
	task.Status = "pending"
	return task, nil
}

func (s *PgStore) GetImageTasks(ctx context.Context, userID string, limit, offset int) ([]ImageTask, error) {
	query := `
		SELECT id, user_id, type, status, model, prompt, size, image_count,
		       result_urls, cost, error_message, created_at, started_at, completed_at, params, image_share_key_id
		FROM image_tasks
		WHERE user_id = $1 AND image_share_key_id IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	return s.queryImageTasks(ctx, query, userID, limit, offset)
}

// GetImageTasksByShareKey returns tasks belonging to the given image-share key.
func (s *PgStore) GetImageTasksByShareKey(ctx context.Context, imageShareKeyID string, limit, offset int) ([]ImageTask, error) {
	query := `
		SELECT id, user_id, type, status, model, prompt, size, image_count,
		       result_urls, cost, error_message, created_at, started_at, completed_at, params, image_share_key_id
		FROM image_tasks
		WHERE image_share_key_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	return s.queryImageTasks(ctx, query, imageShareKeyID, limit, offset)
}

func (s *PgStore) queryImageTasks(ctx context.Context, query string, args ...any) ([]ImageTask, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query image tasks: %w", err)
	}
	defer rows.Close()

	var tasks []ImageTask
	for rows.Next() {
		var t ImageTask
		var resultURLsJSON []byte
		var paramsJSON []byte
		var errMsg sql.NullString
		var startedAt, completedAt sql.NullTime
		var shareKeyID sql.NullString
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.Type, &t.Status, &t.Model, &t.Prompt,
			&t.Size, &t.ImageCount, &resultURLsJSON, &t.Cost, &errMsg,
			&t.CreatedAt, &startedAt, &completedAt, &paramsJSON, &shareKeyID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan image task: %w", err)
		}
		if len(resultURLsJSON) > 0 {
			json.Unmarshal(resultURLsJSON, &t.ResultURLs)
		}
		if len(paramsJSON) > 0 {
			json.Unmarshal(paramsJSON, &t.Params)
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
		if shareKeyID.Valid {
			s := shareKeyID.String
			t.ImageShareKeyID = &s
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (s *PgStore) GetImageTask(ctx context.Context, taskID int) (*ImageTask, error) {
	query := `
		SELECT id, user_id, type, status, model, prompt, size, image_count,
		       input_images, input_mask, result_urls, cost, error_message,
		       created_at, started_at, completed_at, params, image_share_key_id,
		       tenant_id, tenant_key_id, sub_user_id, sub_user_key_id, api_key_id
		FROM image_tasks WHERE id = $1
	`
	var t ImageTask
	var inputImagesJSON, resultURLsJSON, paramsJSON []byte
	var errMsg sql.NullString
	var startedAt, completedAt sql.NullTime
	var shareKeyID sql.NullString
	var tenantID, tenantKeyID, subUserID, subUserKeyID, apiKeyID sql.NullString

	err := s.db.QueryRowContext(ctx, query, taskID).Scan(
		&t.ID, &t.UserID, &t.Type, &t.Status, &t.Model, &t.Prompt,
		&t.Size, &t.ImageCount, &inputImagesJSON, &t.InputMask,
		&resultURLsJSON, &t.Cost, &errMsg,
		&t.CreatedAt, &startedAt, &completedAt, &paramsJSON, &shareKeyID,
		&tenantID, &tenantKeyID, &subUserID, &subUserKeyID, &apiKeyID,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get image task: %w", err)
	}
	if len(inputImagesJSON) > 0 {
		json.Unmarshal(inputImagesJSON, &t.InputImages)
	}
	if len(resultURLsJSON) > 0 {
		json.Unmarshal(resultURLsJSON, &t.ResultURLs)
	}
	if len(paramsJSON) > 0 {
		json.Unmarshal(paramsJSON, &t.Params)
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
	if shareKeyID.Valid {
		v := shareKeyID.String
		t.ImageShareKeyID = &v
	}
	t.TenantID = tenantID.String
	t.TenantKeyID = tenantKeyID.String
	t.SubUserID = subUserID.String
	t.SubUserKeyID = subUserKeyID.String
	t.APIKeyID = apiKeyID.String
	return &t, nil
}

func (s *PgStore) ClaimPendingTask(ctx context.Context) (*ImageTask, error) {
	query := `
		UPDATE image_tasks
		SET status = 'processing', started_at = NOW()
		WHERE id = (
			SELECT id FROM image_tasks
			WHERE status = 'pending'
			ORDER BY created_at
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, user_id, type, status, model, prompt, size, image_count,
		          input_images, input_mask, created_at, started_at, params, image_share_key_id,
		          tenant_id, tenant_key_id, sub_user_id, sub_user_key_id, api_key_id
	`
	var t ImageTask
	var inputImagesJSON, paramsJSON []byte
	var startedAt sql.NullTime
	var shareKeyID sql.NullString
	var tenantID, tenantKeyID, subUserID, subUserKeyID, apiKeyID sql.NullString

	err := s.db.QueryRowContext(ctx, query).Scan(
		&t.ID, &t.UserID, &t.Type, &t.Status, &t.Model, &t.Prompt,
		&t.Size, &t.ImageCount, &inputImagesJSON, &t.InputMask,
		&t.CreatedAt, &startedAt, &paramsJSON, &shareKeyID,
		&tenantID, &tenantKeyID, &subUserID, &subUserKeyID, &apiKeyID,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to claim task: %w", err)
	}
	if len(inputImagesJSON) > 0 {
		json.Unmarshal(inputImagesJSON, &t.InputImages)
	}
	if len(paramsJSON) > 0 {
		json.Unmarshal(paramsJSON, &t.Params)
	}
	if startedAt.Valid {
		t.StartedAt = &startedAt.Time
	}
	if shareKeyID.Valid {
		v := shareKeyID.String
		t.ImageShareKeyID = &v
	}
	t.TenantID = tenantID.String
	t.TenantKeyID = tenantKeyID.String
	t.SubUserID = subUserID.String
	t.SubUserKeyID = subUserKeyID.String
	t.APIKeyID = apiKeyID.String
	return &t, nil
}

func (s *PgStore) CompleteTask(ctx context.Context, taskID int, resultURLs []string, cost float64) error {
	urlsJSON, err := json.Marshal(resultURLs)
	if err != nil {
		return fmt.Errorf("failed to marshal result URLs: %w", err)
	}
	query := `
		UPDATE image_tasks
		SET status = 'completed', result_urls = $1, cost = $2, completed_at = NOW()
		WHERE id = $3
	`
	_, err = s.db.ExecContext(ctx, query, urlsJSON, cost, taskID)
	return err
}

// AppendResultURL appends a single URL into the JSONB result_urls array of a task.
// Used by progressive image generation: each completed image is written immediately
// so that the frontend can render it without waiting for the entire batch.
func (s *PgStore) AppendResultURL(ctx context.Context, taskID int, url string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE image_tasks
		SET result_urls = COALESCE(result_urls, '[]'::jsonb) || to_jsonb($2::text)
		WHERE id = $1
	`, taskID, url)
	if err != nil {
		return fmt.Errorf("failed to append result url: %w", err)
	}
	return nil
}

// FinalizeTask marks a task as completed and records the final cost without touching result_urls
// (which is written progressively by AppendResultURL during execution).
func (s *PgStore) FinalizeTask(ctx context.Context, taskID int, cost float64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE image_tasks
		SET status = 'completed', cost = $1, completed_at = NOW()
		WHERE id = $2
	`, cost, taskID)
	if err != nil {
		return fmt.Errorf("failed to finalize task: %w", err)
	}
	return nil
}

func (s *PgStore) FailTask(ctx context.Context, taskID int, errMsg string) error {
	query := `
		UPDATE image_tasks
		SET status = 'failed', error_message = $1, completed_at = NOW()
		WHERE id = $2
	`
	_, err := s.db.ExecContext(ctx, query, errMsg, taskID)
	return err
}

func (s *PgStore) RecoverStaleTasks(ctx context.Context, timeout time.Duration) (int, error) {
	query := `
		UPDATE image_tasks
		SET status = 'pending', started_at = NULL
		WHERE status = 'processing' AND started_at < NOW() - $1::interval
	`
	res, err := s.db.ExecContext(ctx, query, fmt.Sprintf("%d seconds", int(timeout.Seconds())))
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// DeleteImageTask removes an image task row by ID. Returns sql.ErrNoRows if no row was deleted.
func (s *PgStore) DeleteImageTask(ctx context.Context, taskID int) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM image_tasks WHERE id = $1`, taskID)
	if err != nil {
		return fmt.Errorf("failed to delete image task: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read rows affected: %w", err)
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// UpdateTaskResultURLs replaces the result_urls and image_count of a task.
func (s *PgStore) UpdateTaskResultURLs(ctx context.Context, taskID int, urls []string) error {
	urlsJSON, err := json.Marshal(urls)
	if err != nil {
		return fmt.Errorf("failed to marshal result URLs: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE image_tasks SET result_urls = $1, image_count = $2 WHERE id = $3`,
		urlsJSON, len(urls), taskID,
	)
	if err != nil {
		return fmt.Errorf("failed to update result urls: %w", err)
	}
	return nil
}
