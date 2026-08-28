package store

import (
	"encoding/json"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

const publicLotteryWinnersCountQuery = `SELECT COUNT(*)
		 FROM lottery_records r
		 INNER JOIN lottery_prizes p ON p.id = r.prize_id
		 INNER JOIN users u ON u.id::text = r.user_id
		 WHERE r.event_id = $1 AND p.prize_type <> 'none'`

const publicLotteryWinnersListQuery = `SELECT r.id, u.phone, p.name, p.prize_type,
		        CASE WHEN p.prize_type = 'match_recharge' THEN r.recharge_amount ELSE p.prize_value END,
		        r.created_at
		 FROM lottery_records r
		 INNER JOIN lottery_prizes p ON p.id = r.prize_id
		 INNER JOIN users u ON u.id::text = r.user_id
		 WHERE r.event_id = $1 AND p.prize_type <> 'none'
		 ORDER BY r.created_at DESC, r.id DESC
		 LIMIT $2 OFFSET $3`

const publicLotteryWinnersAllCountQuery = `SELECT COUNT(*)
		 FROM lottery_records r
		 INNER JOIN lottery_prizes p ON p.id = r.prize_id
		 INNER JOIN users u ON u.id::text = r.user_id
		 WHERE p.prize_type <> 'none'`

const publicLotteryWinnersAllListQuery = `SELECT r.id, u.phone, p.name, p.prize_type,
		        CASE WHEN p.prize_type = 'match_recharge' THEN r.recharge_amount ELSE p.prize_value END,
		        r.created_at
		 FROM lottery_records r
		 INNER JOIN lottery_prizes p ON p.id = r.prize_id
		 INNER JOIN users u ON u.id::text = r.user_id
		 WHERE p.prize_type <> 'none'
		 ORDER BY r.created_at DESC, r.id DESC
		 LIMIT $1 OFFSET $2`

func TestMaskLotteryPhone(t *testing.T) {
	tests := []struct {
		name  string
		phone string
		want  string
	}{
		{name: "valid phone", phone: "13812345678", want: "138****5678"},
		{name: "short value", phone: "1381234", want: "****"},
		{name: "non-digit value", phone: "1381234abcd", want: "****"},
		{name: "empty value", phone: "", want: "****"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := maskLotteryPhone(tt.phone); got != tt.want {
				t.Fatalf("maskLotteryPhone(%q) = %q, want %q", tt.phone, got, tt.want)
			}
		})
	}
}

func TestListPublicLotteryWinners(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	store := &PgStore{db: db}
	createdAt := time.Date(2026, time.July, 27, 12, 30, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta(publicLotteryWinnersCountQuery)).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(11))

	mock.ExpectQuery(regexp.QuoteMeta(publicLotteryWinnersListQuery)).
		WithArgs(42, 10, 10).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "phone", "name", "prize_type", "prize_value", "created_at",
		}).AddRow(99, "13812345678", "充值同额返还", "match_recharge", 200.0, createdAt))

	records, total, err := store.ListPublicLotteryWinners(42, 10, 10)
	if err != nil {
		t.Fatalf("ListPublicLotteryWinners() error = %v", err)
	}
	if records == nil {
		t.Fatal("ListPublicLotteryWinners() records is nil, want non-nil slice")
	}
	if total != 11 {
		t.Fatalf("ListPublicLotteryWinners() total = %d, want 11", total)
	}
	if len(records) != 1 {
		t.Fatalf("ListPublicLotteryWinners() returned %d records, want 1", len(records))
	}

	record := records[0]
	if record.ID != 99 || record.MaskedPhone != "138****5678" ||
		record.PrizeName != "充值同额返还" || record.PrizeType != "match_recharge" ||
		record.PrizeValue != 200 || !record.CreatedAt.Equal(createdAt) {
		t.Fatalf("ListPublicLotteryWinners() record = %+v", record)
	}

	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("json.Marshal(record) error = %v", err)
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("json.Unmarshal(record) error = %v", err)
	}
	for _, field := range []string{"phone", "user_id", "order_no", "recharge_amount", "event_id"} {
		if _, ok := fields[field]; ok {
			t.Errorf("public record exposes forbidden JSON field %q", field)
		}
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestListPublicLotteryWinnersReturnsEmptyArray(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	store := &PgStore{db: db}
	mock.ExpectQuery(regexp.QuoteMeta(publicLotteryWinnersCountQuery)).
		WithArgs(42).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta(publicLotteryWinnersListQuery)).
		WithArgs(42, 10, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "phone", "name", "prize_type", "prize_value", "created_at",
		}))

	records, total, err := store.ListPublicLotteryWinners(42, 10, 0)
	if err != nil {
		t.Fatalf("ListPublicLotteryWinners() error = %v", err)
	}
	if total != 0 {
		t.Fatalf("ListPublicLotteryWinners() total = %d, want 0", total)
	}
	if records == nil {
		t.Fatal("ListPublicLotteryWinners() records is nil, want non-nil slice")
	}
	if len(records) != 0 {
		t.Fatalf("ListPublicLotteryWinners() returned %d records, want 0", len(records))
	}

	encoded, err := json.Marshal(records)
	if err != nil {
		t.Fatalf("json.Marshal(records) error = %v", err)
	}
	if string(encoded) != "[]" {
		t.Fatalf("json.Marshal(records) = %s, want []", encoded)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestListAllPublicLotteryWinners(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	store := &PgStore{db: db}
	createdAt := time.Date(2026, time.August, 24, 12, 30, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta(publicLotteryWinnersAllCountQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(11))

	mock.ExpectQuery(regexp.QuoteMeta(publicLotteryWinnersAllListQuery)).
		WithArgs(10, 10).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "phone", "name", "prize_type", "prize_value", "created_at",
		}).AddRow(99, "13812345678", "充值同额返还", "match_recharge", 200.0, createdAt))

	records, total, err := store.ListAllPublicLotteryWinners(10, 10)
	if err != nil {
		t.Fatalf("ListAllPublicLotteryWinners() error = %v", err)
	}
	if records == nil {
		t.Fatal("ListAllPublicLotteryWinners() records is nil, want non-nil slice")
	}
	if total != 11 {
		t.Fatalf("ListAllPublicLotteryWinners() total = %d, want 11", total)
	}
	if len(records) != 1 {
		t.Fatalf("ListAllPublicLotteryWinners() returned %d records, want 1", len(records))
	}

	record := records[0]
	if record.ID != 99 || record.MaskedPhone != "138****5678" ||
		record.PrizeName != "充值同额返还" || record.PrizeType != "match_recharge" ||
		record.PrizeValue != 200 || !record.CreatedAt.Equal(createdAt) {
		t.Fatalf("ListAllPublicLotteryWinners() record = %+v", record)
	}

	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("json.Marshal(record) error = %v", err)
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("json.Unmarshal(record) error = %v", err)
	}
	for _, field := range []string{"phone", "user_id", "order_no", "recharge_amount", "event_id"} {
		if _, ok := fields[field]; ok {
			t.Errorf("public record exposes forbidden JSON field %q", field)
		}
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestListAllPublicLotteryWinnersReturnsEmptyArray(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	store := &PgStore{db: db}
	mock.ExpectQuery(regexp.QuoteMeta(publicLotteryWinnersAllCountQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta(publicLotteryWinnersAllListQuery)).
		WithArgs(10, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "phone", "name", "prize_type", "prize_value", "created_at",
		}))

	records, total, err := store.ListAllPublicLotteryWinners(10, 0)
	if err != nil {
		t.Fatalf("ListAllPublicLotteryWinners() error = %v", err)
	}
	if total != 0 {
		t.Fatalf("ListAllPublicLotteryWinners() total = %d, want 0", total)
	}
	if records == nil {
		t.Fatal("ListAllPublicLotteryWinners() records is nil, want non-nil slice")
	}
	if len(records) != 0 {
		t.Fatalf("ListAllPublicLotteryWinners() returned %d records, want 0", len(records))
	}

	encoded, err := json.Marshal(records)
	if err != nil {
		t.Fatalf("json.Marshal(records) error = %v", err)
	}
	if string(encoded) != "[]" {
		t.Fatalf("json.Marshal(records) = %s, want []", encoded)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}
