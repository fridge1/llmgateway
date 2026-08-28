package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrTaskNotCompleted is returned when claiming a task that isn't completed.
var ErrTaskNotCompleted = errors.New("task not completed")

// ErrTaskAlreadyClaimed is returned when a task reward was already claimed.
var ErrTaskAlreadyClaimed = errors.New("task reward already claimed")

// TaskDefinition is a growth-task definition with the current user's progress.
type TaskDefinition struct {
	Code               string     `json:"code"`
	Title              string     `json:"title"`
	Description        string     `json:"description"`
	RewardCNY          float64    `json:"reward_cny"`
	RewardLotteryDraws int        `json:"reward_lottery_draws"`
	SortOrder          int        `json:"sort_order"`
	Status             string     `json:"status"` // pending | completed | claimed
	CompletedAt        *time.Time `json:"completed_at,omitempty"`
	ClaimedAt          *time.Time `json:"claimed_at,omitempty"`
}

// ListUserTasks returns all active tasks joined with the user's progress.
// Before reading, it reconciles usage-derived tasks (first_api_call,
// daily_spend_1) from the transactions table, so these complete without
// touching any hot path — the check runs only when the user opens the page.
func (s *PgStore) ListUserTasks(userID string) ([]TaskDefinition, error) {
	s.reconcileUsageTasks(userID)

	rows, err := s.db.Query(
		`SELECT d.code, d.title, d.description, d.reward_cny, d.reward_lottery_draws, d.sort_order,
		        COALESCE(p.status, 'pending'), p.completed_at, p.claimed_at
		 FROM task_definitions d
		 LEFT JOIN user_task_progress p ON p.task_code = d.code AND p.user_id = $1
		 WHERE d.is_active = TRUE
		 ORDER BY d.sort_order, d.id`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list user tasks: %w", err)
	}
	defer rows.Close()

	var out []TaskDefinition
	for rows.Next() {
		var t TaskDefinition
		if err := rows.Scan(
			&t.Code, &t.Title, &t.Description, &t.RewardCNY, &t.RewardLotteryDraws, &t.SortOrder,
			&t.Status, &t.CompletedAt, &t.ClaimedAt,
		); err != nil {
			return nil, fmt.Errorf("store: scan user task: %w", err)
		}
		out = append(out, t)
	}
	if out == nil {
		out = []TaskDefinition{}
	}
	return out, rows.Err()
}

// reconcileUsageTasks marks usage-derived tasks complete based on the
// transactions table, without requiring hot-path hooks. Best-effort: errors
// are swallowed so a transient failure never blocks the task listing.
func (s *PgStore) reconcileUsageTasks(userID string) {
	// first_api_call: any consumption/subscription_usage transaction exists.
	var hasCall bool
	_ = s.db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM transactions
		 WHERE user_id = $1 AND type IN ('consumption','subscription_usage'))`,
		userID,
	).Scan(&hasCall)
	if hasCall {
		_ = s.MarkTaskCompleted(userID, "first_api_call")
	}

	// daily_spend_1: total spend on any single calendar day reached ¥1.
	var hasBigDay bool
	_ = s.db.QueryRow(
		`SELECT EXISTS(
		   SELECT 1 FROM transactions
		   WHERE user_id = $1 AND type IN ('consumption','subscription_usage')
		   GROUP BY created_at::date
		   HAVING SUM(amount) >= 1.0)`,
		userID,
	).Scan(&hasBigDay)
	if hasBigDay {
		_ = s.MarkTaskCompleted(userID, "daily_spend_1")
	}

	// try_image: a completed image task exists for the user.
	var hasImage bool
	_ = s.db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM image_tasks WHERE user_id = $1 AND status = 'completed')`,
		userID,
	).Scan(&hasImage)
	if hasImage {
		_ = s.MarkTaskCompleted(userID, "try_image")
	}
}

// MarkTaskCompleted idempotently marks a task as completed for a user. It is a
// no-op if the task is already completed or claimed, or if the code is unknown.
func (s *PgStore) MarkTaskCompleted(userID, code string) error {
	// Ignore unknown / inactive task codes.
	var exists bool
	if err := s.db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM task_definitions WHERE code = $1 AND is_active = TRUE)`, code,
	).Scan(&exists); err != nil {
		return fmt.Errorf("store: task exists check: %w", err)
	}
	if !exists {
		return nil
	}
	_, err := s.db.Exec(
		`INSERT INTO user_task_progress (user_id, task_code, status, completed_at)
		 VALUES ($1, $2, 'completed', NOW())
		 ON CONFLICT (user_id, task_code) DO UPDATE
		   SET status = 'completed', completed_at = COALESCE(user_task_progress.completed_at, NOW())
		   WHERE user_task_progress.status = 'pending'`,
		userID, code,
	)
	if err != nil {
		return fmt.Errorf("store: mark task completed: %w", err)
	}
	return nil
}

// ClaimTaskReward atomically claims a completed task's reward, crediting the
// CNY reward to the user's balance. Returns the credited amount and lottery
// draws granted. Concurrency-safe: the status transition guards double-claims.
func (s *PgStore) ClaimTaskReward(userID, code string) (rewardCNY float64, lotteryDraws int, err error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, 0, fmt.Errorf("store: begin claim tx: %w", err)
	}
	defer tx.Rollback()

	// Lock the progress row and verify it's completed-but-unclaimed.
	var status string
	err = tx.QueryRow(
		`SELECT status FROM user_task_progress WHERE user_id = $1 AND task_code = $2 FOR UPDATE`,
		userID, code,
	).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, ErrTaskNotCompleted
	}
	if err != nil {
		return 0, 0, fmt.Errorf("store: claim lookup: %w", err)
	}
	switch status {
	case "claimed":
		return 0, 0, ErrTaskAlreadyClaimed
	case "completed":
		// proceed
	default:
		return 0, 0, ErrTaskNotCompleted
	}

	if err = tx.QueryRow(
		`SELECT reward_cny, reward_lottery_draws FROM task_definitions WHERE code = $1`, code,
	).Scan(&rewardCNY, &lotteryDraws); err != nil {
		return 0, 0, fmt.Errorf("store: claim reward lookup: %w", err)
	}

	if _, err = tx.Exec(
		`UPDATE user_task_progress SET status = 'claimed', claimed_at = NOW()
		 WHERE user_id = $1 AND task_code = $2`,
		userID, code,
	); err != nil {
		return 0, 0, fmt.Errorf("store: claim update: %w", err)
	}

	if rewardCNY > 0 {
		if _, err = tx.Exec(
			`UPDATE balances SET balance = balance + $1, updated_at = NOW() WHERE user_id = $2`,
			rewardCNY, userID,
		); err != nil {
			return 0, 0, fmt.Errorf("store: claim credit: %w", err)
		}
		if _, err = tx.Exec(
			`INSERT INTO transactions (user_id, type, amount, balance_after, description)
			 SELECT $1, 'task_reward', $2, balance, $3 FROM balances WHERE user_id = $1`,
			userID, rewardCNY, fmt.Sprintf("任务奖励：%s", code),
		); err != nil {
			return 0, 0, fmt.Errorf("store: claim transaction: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("store: commit claim: %w", err)
	}
	return rewardCNY, lotteryDraws, nil
}
