package sms

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// Sender is the interface for sending SMS verification codes.
type Sender interface {
	// SendCode sends a verification code to the given phone number.
	// templateID is the SMS template ID; paramKey is the template variable name (e.g. "code").
	SendCode(phone, code, templateID, paramKey string) error
}

// codeEntry stores a verification code with TTL and rate-limit tracking.
type codeEntry struct {
	Code      string
	ExpiresAt time.Time
	SentAt    time.Time
	Attempts  int
}

// CodeStore is a thread-safe in-memory store for SMS verification codes
// with 5-minute TTL and 1-per-60-second rate limiting per phone.
type CodeStore struct {
	mu     sync.Mutex
	codes  map[string]*codeEntry // phone -> entry
	stopCh chan struct{}
}

// NewCodeStore creates a new CodeStore.
func NewCodeStore() *CodeStore {
	cs := &CodeStore{
		codes:  make(map[string]*codeEntry),
		stopCh: make(chan struct{}),
	}
	// Background cleanup every 2 minutes.
	go cs.cleanup()
	return cs
}

// Stop terminates the background cleanup goroutine.
func (cs *CodeStore) Stop() {
	close(cs.stopCh)
}

// GenerateCode returns a random 6-digit code string.
func GenerateCode() string {
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}

// Set stores a code for a phone number. It returns an error if a code was
// sent to the same phone within the last 60 seconds (rate limit).
func (cs *CodeStore) Set(phone, code string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	now := time.Now()

	if entry, ok := cs.codes[phone]; ok {
		// Rate limit: 1 code per 60 seconds per phone.
		if now.Before(entry.SentAt.Add(60 * time.Second)) {
			return fmt.Errorf("请等待 60 秒后再发送验证码")
		}
	}

	cs.codes[phone] = &codeEntry{
		Code:      code,
		ExpiresAt: now.Add(5 * time.Minute),
		SentAt:    now,
	}
	return nil
}

// Verify checks the code for the given phone. Returns true if valid, then deletes it.
func (cs *CodeStore) Verify(phone, code string) bool {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	entry, ok := cs.codes[phone]
	if !ok {
		return false
	}

	if time.Now().After(entry.ExpiresAt) {
		delete(cs.codes, phone)
		return false
	}

	if entry.Attempts >= 5 {
		delete(cs.codes, phone)
		return false
	}

	if entry.Code != code {
		entry.Attempts++
		return false
	}

	// Code is valid; consume it.
	delete(cs.codes, phone)
	return true
}

// cleanup periodically removes expired entries.
func (cs *CodeStore) cleanup() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			cs.mu.Lock()
			now := time.Now()
			for phone, entry := range cs.codes {
				if now.After(entry.ExpiresAt) {
					delete(cs.codes, phone)
				}
			}
			cs.mu.Unlock()
		case <-cs.stopCh:
			return
		}
	}
}
