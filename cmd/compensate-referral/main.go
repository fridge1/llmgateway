package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/zhulang/llm-gateway/internal/config"
	"github.com/zhulang/llm-gateway/internal/store"
)

func main() {
	configPath := flag.String("config", "config.yaml", "config file path")
	dryRun := flag.Bool("dry-run", false, "preview only, do not grant rewards")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadFromFile(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	slog.Info("Referral reward compensation tool",
		"inviter_bonus", cfg.Promotion.ReferralInviterBonusCNY,
		"invitee_bonus", cfg.Promotion.ReferralInviteeBonusCNY,
		"dry_run", *dryRun)

	if cfg.Promotion.ReferralInviterBonusCNY == 0 && cfg.Promotion.ReferralInviteeBonusCNY == 0 {
		log.Fatal("Both inviter and invitee bonuses are 0, nothing to compensate")
	}

	// Connect to database
	pgStore, err := store.OpenPostgres(cfg.Database.DSN, 0, 0, 0)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Query eligible users: those with referred_by set, reward not granted, and at least one paid order
	query := `
		SELECT DISTINCT u.id, u.phone, u.referred_by
		FROM users u
		INNER JOIN orders o ON o.user_id = u.id AND o.status = 'paid'
		WHERE u.referred_by IS NOT NULL
		  AND u.referral_reward_granted = FALSE
		ORDER BY u.id
	`

	rows, err := pgStore.DB().Query(query)
	if err != nil {
		log.Fatalf("Failed to query eligible users: %v", err)
	}
	defer rows.Close()

	var eligible []struct {
		InvitedUserID string
		Phone         string
		InviterID     string
	}

	for rows.Next() {
		var item struct {
			InvitedUserID string
			Phone         string
			InviterID     string
		}
		if err := rows.Scan(&item.InvitedUserID, &item.Phone, &item.InviterID); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}
		eligible = append(eligible, item)
	}

	if err := rows.Err(); err != nil {
		log.Fatalf("Error iterating rows: %v", err)
	}

	slog.Info("Found eligible users", "count", len(eligible))

	if len(eligible) == 0 {
		slog.Info("No users need compensation")
		return
	}

	// Preview mode
	if *dryRun {
		fmt.Print("\n=== DRY RUN MODE - No changes will be made ===\n\n")
		for i, item := range eligible {
			var inviterPhone string
			err := pgStore.DB().QueryRow("SELECT phone FROM users WHERE id = $1", item.InviterID).Scan(&inviterPhone)
			if err != nil && err != sql.ErrNoRows {
				slog.Warn("Failed to get inviter phone", "inviter_id", item.InviterID, "error", err)
				inviterPhone = "unknown"
			}

			fmt.Printf("%d. Invited: %s (phone: %s)\n", i+1, item.InvitedUserID, item.Phone)
			fmt.Printf("   Inviter: %s (phone: %s)\n", item.InviterID, inviterPhone)
			fmt.Printf("   Would grant: invitee ¥%.2f, inviter ¥%.2f\n\n",
				cfg.Promotion.ReferralInviteeBonusCNY,
				cfg.Promotion.ReferralInviterBonusCNY)
		}
		fmt.Printf("Total: %d users would receive compensation\n", len(eligible))
		return
	}

	// Actual compensation
	fmt.Print("\n=== ACTUAL COMPENSATION - Changes will be made ===\n\n")
	successCount := 0
	errorCount := 0

	for i, item := range eligible {
		granted, err := pgStore.GrantReferralReward(
			item.InvitedUserID,
			cfg.Promotion.ReferralInviterBonusCNY,
			cfg.Promotion.ReferralInviteeBonusCNY,
		)

		if err != nil {
			slog.Error("Failed to grant reward",
				"invited_user_id", item.InvitedUserID,
				"phone", item.Phone,
				"error", err)
			errorCount++
			continue
		}

		if granted {
			slog.Info("Compensation granted",
				"index", i+1,
				"invited_user_id", item.InvitedUserID,
				"phone", item.Phone,
				"inviter_id", item.InviterID,
				"invitee_bonus", cfg.Promotion.ReferralInviteeBonusCNY,
				"inviter_bonus", cfg.Promotion.ReferralInviterBonusCNY)
			successCount++
		} else {
			slog.Warn("Reward not granted (may have been granted already or no referrer)",
				"invited_user_id", item.InvitedUserID,
				"phone", item.Phone)
		}
	}

	fmt.Printf("\n=== SUMMARY ===\n")
	fmt.Printf("Total eligible: %d\n", len(eligible))
	fmt.Printf("Successfully compensated: %d\n", successCount)
	fmt.Printf("Errors: %d\n", errorCount)
	fmt.Printf("Skipped (already granted): %d\n", len(eligible)-successCount-errorCount)

	if errorCount > 0 {
		os.Exit(1)
	}
}
