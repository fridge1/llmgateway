package invoice

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/storage"
	"github.com/zhulang/llm-gateway/internal/store"
)

// AdminHandler provides invoice admin API endpoints.
type AdminHandler struct {
	store store.Store
	tos   *storage.TOSClient
}

// NewAdminHandler creates a new admin invoice Handler.
func NewAdminHandler(s store.Store, tos *storage.TOSClient) *AdminHandler {
	return &AdminHandler{store: s, tos: tos}
}

// HandleAdminListRequests handles GET /api/admin/invoice/requests
func (h *AdminHandler) HandleAdminListRequests(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size < 1 || size > 100 {
		size = 20
	}
	status := r.URL.Query().Get("status")
	offset := (page - 1) * size

	details, total, err := h.store.AdminListInvoiceRequests(status, size, offset)
	if err != nil {
		slog.Error("failed to list admin invoice requests", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list requests", "server_error", "db_error")
		return
	}

	// Enrich with title info
	for i := range details {
		title, err := h.store.GetInvoiceTitleByID(details[i].TitleID)
		if err == nil {
			details[i].Title = *title
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"requests": details,
		"total":    total,
		"page":     page,
		"size":     size,
	})
}

// HandleAdminGetRequest handles GET /api/admin/invoice/requests/{id}
func (h *AdminHandler) HandleAdminGetRequest(w http.ResponseWriter, r *http.Request) {
	id, err := extractRequestID(r.URL.Path)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request id", "invalid_request", "bad_id")
		return
	}

	req, err := h.store.GetInvoiceRequest(id, "") // empty userID = admin
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "request not found", "not_found", "no_request")
		return
	}

	orders, _ := h.store.GetInvoiceRequestOrders(id)
	title, _ := h.store.GetInvoiceTitleByID(req.TitleID)
	user, _ := h.store.GetUserByID(req.UserID)

	result := map[string]any{
		"request": req,
		"orders":  orders,
		"title":   title,
	}
	if user != nil {
		result["user"] = map[string]string{
			"id":    user.ID,
			"phone": user.Phone,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// HandleAdminProcess handles PUT /api/admin/invoice/requests/{id}/process
func (h *AdminHandler) HandleAdminProcess(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/process")
	id, err := extractRequestID(path)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request id", "invalid_request", "bad_id")
		return
	}

	if err := h.store.UpdateInvoiceRequestStatus(id, "processing"); err != nil {
		slog.Error("failed to process invoice request", "error", err)
		httputil.WriteError(w, http.StatusBadRequest, err.Error(), "invalid_request", "process_failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "processing"})
}

// HandleAdminComplete handles PUT /api/admin/invoice/requests/{id}/complete (multipart)
func (h *AdminHandler) HandleAdminComplete(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/complete")
	id, err := extractRequestID(path)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request id", "invalid_request", "bad_id")
		return
	}

	if h.tos == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "storage not configured", "server_error", "no_storage")
		return
	}

	// Parse multipart: max 10MB
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "file too large or invalid form", "invalid_request", "bad_form")
		return
	}

	invoiceNumber := r.FormValue("invoice_number")
	if strings.TrimSpace(invoiceNumber) == "" {
		httputil.WriteError(w, http.StatusBadRequest, "invoice_number is required", "invalid_request", "missing_number")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "file is required", "invalid_request", "missing_file")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		slog.Error("failed to read uploaded file", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "failed to read file", "server_error", "read_error")
		return
	}

	now := time.Now()
	key := fmt.Sprintf("invoices/%d/%d%02d%02d_%d.pdf", id, now.Year(), now.Month(), now.Day(), now.UnixMilli())

	ctx := context.Background()
	if _, err := h.tos.UploadFile(ctx, data, key, "application/pdf"); err != nil {
		slog.Error("failed to upload invoice to TOS", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "failed to upload file", "server_error", "upload_error")
		return
	}

	if err := h.store.CompleteInvoiceRequest(id, key, invoiceNumber); err != nil {
		slog.Error("failed to complete invoice request", "error", err)
		httputil.WriteError(w, http.StatusBadRequest, err.Error(), "invalid_request", "complete_failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "completed"})
}

// HandleAdminReject handles PUT /api/admin/invoice/requests/{id}/reject
func (h *AdminHandler) HandleAdminReject(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/reject")
	id, err := extractRequestID(path)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request id", "invalid_request", "bad_id")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request", "invalid_request", "bad_json")
		return
	}

	if strings.TrimSpace(req.Reason) == "" {
		httputil.WriteError(w, http.StatusBadRequest, "reason is required", "invalid_request", "missing_reason")
		return
	}

	if err := h.store.RejectInvoiceRequest(id, req.Reason); err != nil {
		slog.Error("failed to reject invoice request", "error", err)
		httputil.WriteError(w, http.StatusBadRequest, err.Error(), "invalid_request", "reject_failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "rejected"})
}

// HandleAdminBatchApprove moves a batch of pending requests to processing.
// Only requests flagged auto_ok are safe to include; requests that are not
// pending are silently skipped (their IDs are excluded from the response).
// POST /api/admin/invoice/requests/batch-approve  body: {ids: []int64}
func (h *AdminHandler) HandleAdminBatchApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request_error", "method_not_allowed")
		return
	}
	var req struct {
		IDs []int64 `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.IDs) == 0 {
		httputil.WriteError(w, http.StatusBadRequest, "ids is required", "invalid_request_error", "bad_ids")
		return
	}
	if len(req.IDs) > 200 {
		httputil.WriteError(w, http.StatusBadRequest, "最多一次批量处理 200 条", "invalid_request_error", "too_many")
		return
	}
	updated, err := h.store.BatchUpdateInvoiceRequestStatus(req.IDs, "pending", "processing")
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "batch approve failed", "server_error", "db_error")
		return
	}
	if updated == nil {
		updated = []int64{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "ok", "approved": updated, "count": len(updated)})
}
