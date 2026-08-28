package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/zhulang/llm-gateway/internal/money"
)

// Balance represents a user's balance and frozen amount.
type Balance struct {
	UserID    string    `json:"user_id"`
	Balance   float64   `json:"balance"`
	Frozen    float64   `json:"frozen"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GetBalance returns the balance for the given user.
func (s *PgStore) GetBalance(userID string) (*Balance, error) {
	var b Balance
	err := s.db.QueryRow(
		`SELECT user_id, balance, frozen, updated_at
		 FROM balances WHERE user_id = $1`,
		userID,
	).Scan(&b.UserID, &b.Balance, &b.Frozen, &b.UpdatedAt)
	if err == sql.ErrNoRows {
		return &Balance{UserID: userID, Balance: 0, Frozen: 0}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: get balance: %w", err)
	}
	return &b, nil
}

// FreezeBalance atomically moves amount from available balance to frozen.
func (s *PgStore) FreezeBalance(userID string, amount float64) error {
	res, err := s.db.Exec(
		`UPDATE balances SET frozen = frozen + $1, updated_at = NOW()
		 WHERE user_id = $2 AND (balance - frozen) >= $1`,
		amount, userID,
	)
	if err != nil {
		return fmt.Errorf("store: freeze balance: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("store: insufficient balance")
	}
	return nil
}

// UnfreezeBalance atomically moves amount from frozen back.
func (s *PgStore) UnfreezeBalance(userID string, amount float64) error {
	res, err := s.db.Exec(
		`UPDATE balances SET frozen = frozen - $1, updated_at = NOW()
		 WHERE user_id = $2 AND frozen >= $1`,
		amount, userID,
	)
	if err != nil {
		return fmt.Errorf("store: unfreeze balance: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("store: insufficient frozen amount")
	}
	return nil
}

// SettleBilling settles a billing transaction: unfreezes the frozen amount,
// deducts the actual cost from balance, and records the transaction.
func (s *PgStore) SettleBilling(userID string, frozenAmount, actualCost float64, model, requestID string, tokens TokenUsage, apiKeyID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`UPDATE balances SET frozen = frozen - $1, updated_at = NOW() WHERE user_id = $2`,
		frozenAmount, userID,
	)
	if err != nil {
		return fmt.Errorf("store: settle unfreeze: %w", err)
	}

	_, err = tx.Exec(
		`UPDATE balances SET balance = balance - $1, updated_at = NOW() WHERE user_id = $2`,
		actualCost, userID,
	)
	if err != nil {
		return fmt.Errorf("store: settle balance update: %w", err)
	}

	var balanceAfter float64
	err = tx.QueryRow(
		`SELECT balance FROM balances WHERE user_id = $1`,
		userID,
	).Scan(&balanceAfter)
	if err != nil {
		return fmt.Errorf("store: settle get balance: %w", err)
	}

	var apiKeyIDPtr *string
	if apiKeyID != "" {
		apiKeyIDPtr = &apiKeyID
	}

	_, err = tx.Exec(
		`INSERT INTO transactions (user_id, type, amount, balance_after, model, description, request_id,
		  prompt_tokens, completion_tokens, cache_read_tokens,
		  cache_creation_tokens, cache_creation_5m_tokens, cache_creation_1h_tokens, api_key_id)
		 VALUES ($1, 'consumption', $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		userID, actualCost, balanceAfter, model,
		fmt.Sprintf("API call to %s", model), requestID,
		intPtrOrNil(tokens.PromptTokens), intPtrOrNil(tokens.CompletionTokens),
		intPtrOrNil(tokens.CacheReadTokens), intPtrOrNil(tokens.CacheCreationTokens),
		intPtrOrNil(tokens.CacheCreation5mTokens), intPtrOrNil(tokens.CacheCreation1hTokens),
		apiKeyIDPtr,
	)
	if err != nil {
		return fmt.Errorf("store: settle insert transaction: %w", err)
	}

	return tx.Commit()
}

// DirectCharge deducts amount directly from user's balance and records a transaction.
func (s *PgStore) DirectCharge(userID string, amount float64, model, requestID string, tokens TokenUsage, apiKeyID string) error {
	if amount <= 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`UPDATE balances SET balance = balance - $1, updated_at = NOW() WHERE user_id = $2`,
		amount, userID,
	)
	if err != nil {
		return fmt.Errorf("store: direct charge balance update: %w", err)
	}

	var balanceAfter float64
	err = tx.QueryRow(
		`SELECT balance FROM balances WHERE user_id = $1`,
		userID,
	).Scan(&balanceAfter)
	if err != nil {
		return fmt.Errorf("store: direct charge get balance: %w", err)
	}

	var apiKeyIDPtr *string
	if apiKeyID != "" {
		apiKeyIDPtr = &apiKeyID
	}

	_, err = tx.Exec(
		`INSERT INTO transactions (user_id, type, amount, balance_after, model, description, request_id,
		  prompt_tokens, completion_tokens, cache_read_tokens,
		  cache_creation_tokens, cache_creation_5m_tokens, cache_creation_1h_tokens, api_key_id)
		 VALUES ($1, 'consumption', $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		userID, amount, balanceAfter, model,
		fmt.Sprintf("API call to %s", model), requestID,
		intPtrOrNil(tokens.PromptTokens), intPtrOrNil(tokens.CompletionTokens),
		intPtrOrNil(tokens.CacheReadTokens), intPtrOrNil(tokens.CacheCreationTokens),
		intPtrOrNil(tokens.CacheCreation5mTokens), intPtrOrNil(tokens.CacheCreation1hTokens),
		apiKeyIDPtr,
	)
	if err != nil {
		return fmt.Errorf("store: direct charge insert transaction: %w", err)
	}

	return tx.Commit()
}

// DeductForSubscription atomically checks that the user has sufficient balance
// and deducts the amount. Returns an error if the available balance is insufficient.
func (s *PgStore) DeductForSubscription(userID string, amount float64, description string) error {
	if amount <= 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback()

	var bal, frozen float64
	err = tx.QueryRow(
		`SELECT balance, frozen FROM balances WHERE user_id = $1 FOR UPDATE`,
		userID,
	).Scan(&bal, &frozen)
	if err != nil {
		return fmt.Errorf("store: subscription deduct read balances: %w", err)
	}

	available := bal - frozen
	// Tolerant comparison: float noise and legacy half-cent payment gaps
	// must not fail a purchase the user has effectively paid for.
	if !money.GTE(available, amount) {
		return fmt.Errorf("余额不足：可用余额 ¥%.2f，需要 ¥%.2f", available, amount)
	}

	_, err = tx.Exec(
		`UPDATE balances SET balance = GREATEST(balance - $1, 0), updated_at = NOW() WHERE user_id = $2`,
		amount, userID,
	)
	if err != nil {
		return fmt.Errorf("store: subscription deduct update: %w", err)
	}

	var balanceAfter float64
	err = tx.QueryRow(
		`SELECT balance FROM balances WHERE user_id = $1`,
		userID,
	).Scan(&balanceAfter)
	if err != nil {
		return fmt.Errorf("store: subscription deduct get balance: %w", err)
	}

	_, err = tx.Exec(
		`INSERT INTO transactions (user_id, type, amount, balance_after, description)
		 VALUES ($1, 'sub_purchase', $2, $3, $4)`,
		userID, amount, balanceAfter, description,
	)
	if err != nil {
		return fmt.Errorf("store: subscription deduct insert transaction: %w", err)
	}

	return tx.Commit()
}

// Recharge adds amount to the user's balance and records a transaction.
func (s *PgStore) Recharge(userID string, amount float64, description string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: begin recharge tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		`UPDATE balances SET balance = balance + $1, updated_at = NOW() WHERE user_id = $2`,
		amount, userID,
	)
	if err != nil {
		return fmt.Errorf("store: recharge update: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		_, err = tx.Exec(
			`INSERT INTO balances (user_id, balance, frozen) VALUES ($1, $2, 0)`,
			userID, amount,
		)
		if err != nil {
			res, err = tx.Exec(
				`UPDATE balances SET balance = balance + $1, updated_at = NOW() WHERE user_id = $2`,
				amount, userID,
			)
			if err != nil {
				return fmt.Errorf("store: recharge update (retry): %w", err)
			}
			if n, _ = res.RowsAffected(); n == 0 {
				return fmt.Errorf("store: recharge update: no balance row for user %s", userID)
			}
		}
	}

	var balanceAfter float64
	err = tx.QueryRow(
		`SELECT balance FROM balances WHERE user_id = $1`,
		userID,
	).Scan(&balanceAfter)
	if err != nil {
		return fmt.Errorf("store: recharge get balance: %w", err)
	}

	_, err = tx.Exec(
		`INSERT INTO transactions (user_id, type, amount, balance_after, description)
		 VALUES ($1, 'recharge', $2, $3, $4)`,
		userID, amount, balanceAfter, description,
	)
	if err != nil {
		return fmt.Errorf("store: recharge insert transaction: %w", err)
	}

	return tx.Commit()
}

// intPtrOrNil returns a pointer to v if v > 0, otherwise nil (stored as NULL).
func intPtrOrNil(v int) *int {
	if v == 0 {
		return nil
	}
	return &v
}
