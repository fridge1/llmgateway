package lottery

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zhulang/llm-gateway/internal/store"
)

type handlerStore struct {
	store.Store

	records    []store.PublicLotteryRecord
	total      int
	recordsErr error

	winnerCalled bool
	limit        int
	offset       int
}

func (s *handlerStore) ListAllPublicLotteryWinners(limit, offset int) ([]store.PublicLotteryRecord, int, error) {
	s.winnerCalled = true
	s.limit = limit
	s.offset = offset
	return s.records, s.total, s.recordsErr
}

func TestHandleUserGetRecordsListsWinners(t *testing.T) {
	createdAt := time.Date(2026, time.July, 27, 12, 0, 0, 0, time.UTC)
	s := &handlerStore{
		records: []store.PublicLotteryRecord{
			{
				ID:          99,
				MaskedPhone: "138****5678",
				PrizeName:   "充值同额返还",
				PrizeType:   "match_recharge",
				PrizeValue:  200,
				CreatedAt:   createdAt,
			},
		},
		total: 11,
	}
	h := NewHandler(s)
	req := httptest.NewRequest(http.MethodGet, "/api/lottery/records?page=2&size=10", nil)
	w := httptest.NewRecorder()

	h.HandleUserGetRecords(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HandleUserGetRecords() status = %d, want %d; body = %q", w.Code, http.StatusOK, w.Body.String())
	}
	if !s.winnerCalled {
		t.Fatal("ListAllPublicLotteryWinners() was not called")
	}
	if s.limit != 10 || s.offset != 10 {
		t.Fatalf("ListAllPublicLotteryWinners() args = (%d, %d), want (10, 10)", s.limit, s.offset)
	}

	body := w.Body.Bytes()
	var response struct {
		Records []store.PublicLotteryRecord `json:"records"`
		Total   int                         `json:"total"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Total != 11 || len(response.Records) != 1 {
		t.Fatalf("response = %+v, want one record and total 11", response)
	}
	if response.Records[0].MaskedPhone != "138****5678" {
		t.Fatalf("masked_phone = %q, want %q", response.Records[0].MaskedPhone, "138****5678")
	}

	for _, sensitive := range []string{`"user_id"`, `"order_no"`, `"recharge_amount"`, "13812345678"} {
		if strings.Contains(string(body), sensitive) {
			t.Errorf("response contains sensitive value %q: %s", sensitive, body)
		}
	}
}

func TestHandleUserGetRecordsReturnsEmptyArrayWhenNoWinners(t *testing.T) {
	s := &handlerStore{records: []store.PublicLotteryRecord{}}
	h := NewHandler(s)
	req := httptest.NewRequest(http.MethodGet, "/api/lottery/records", nil)
	w := httptest.NewRecorder()

	h.HandleUserGetRecords(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HandleUserGetRecords() status = %d, want %d; body = %q", w.Code, http.StatusOK, w.Body.String())
	}
	if !s.winnerCalled {
		t.Fatal("ListAllPublicLotteryWinners() was not called")
	}
	var response struct {
		Records []store.PublicLotteryRecord `json:"records"`
		Total   int                         `json:"total"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Records == nil {
		t.Fatal("records is nil, want non-nil empty slice")
	}
	if len(response.Records) != 0 || response.Total != 0 {
		t.Fatalf("response = %+v, want empty records and total 0", response)
	}
}

func TestHandleUserGetRecordsReturnsInternalErrorWhenWinnerLookupFails(t *testing.T) {
	s := &handlerStore{recordsErr: errors.New("winner lookup failed")}
	h := NewHandler(s)
	req := httptest.NewRequest(http.MethodGet, "/api/lottery/records", nil)
	w := httptest.NewRecorder()

	h.HandleUserGetRecords(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("HandleUserGetRecords() status = %d, want %d; body = %q", w.Code, http.StatusInternalServerError, w.Body.String())
	}
	if !s.winnerCalled {
		t.Fatal("ListAllPublicLotteryWinners() was not called")
	}
}
