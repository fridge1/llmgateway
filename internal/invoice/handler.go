package invoice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/zhulang/llm-gateway/internal/admin"
	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/storage"
	"github.com/zhulang/llm-gateway/internal/store"
)

// Handler provides invoice API endpoints for users.
type Handler struct {
	store store.Store
	tos   *storage.TOSClient
}

// NewHandler creates a new invoice Handler.
func NewHandler(s store.Store, tos *storage.TOSClient) *Handler {
	return &Handler{store: s, tos: tos}
}

func getUserID(r *http.Request) string {
	userID, _ := r.Context().Value(admin.CtxUserIDKey).(string)
	return userID
}

// HandleListTitles handles GET /api/invoice/titles
func (h *Handler) HandleListTitles(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	titles, err := h.store.ListInvoiceTitlesByUser(userID)
	if err != nil {
		slog.Error("failed to list invoice titles", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list titles", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"titles": titles})
}

// HandleCreateTitle handles POST /api/invoice/titles
func (h *Handler) HandleCreateTitle(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	var req struct {
		Type        string `json:"type"`
		TitleName   string `json:"title_name"`
		TaxNumber   string `json:"tax_number"`
		BankName    string `json:"bank_name"`
		BankAccount string `json:"bank_account"`
		Address     string `json:"address"`
		Phone       string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request", "invalid_request", "bad_json")
		return
	}

	if req.Type != "personal" && req.Type != "enterprise" {
		httputil.WriteError(w, http.StatusBadRequest, "type must be personal or enterprise", "invalid_request", "bad_type")
		return
	}
	if strings.TrimSpace(req.TitleName) == "" {
		httputil.WriteError(w, http.StatusBadRequest, "title_name is required", "invalid_request", "missing_title")
		return
	}
	if req.Type == "enterprise" && strings.TrimSpace(req.TaxNumber) == "" {
		httputil.WriteError(w, http.StatusBadRequest, "tax_number is required for enterprise", "invalid_request", "missing_tax")
		return
	}

	title, err := h.store.CreateInvoiceTitle(userID, req.Type, req.TitleName, req.TaxNumber, req.BankName, req.BankAccount, req.Address, req.Phone)
	if err != nil {
		slog.Error("failed to create invoice title", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "failed to create title", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(title)
}

// HandleUpdateTitle handles PUT /api/invoice/titles/{id}
func (h *Handler) HandleUpdateTitle(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	id, err := extractTitleID(r.URL.Path)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid title id", "invalid_request", "bad_id")
		return
	}

	var req struct {
		Type        string `json:"type"`
		TitleName   string `json:"title_name"`
		TaxNumber   string `json:"tax_number"`
		BankName    string `json:"bank_name"`
		BankAccount string `json:"bank_account"`
		Address     string `json:"address"`
		Phone       string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request", "invalid_request", "bad_json")
		return
	}

	title, err := h.store.UpdateInvoiceTitle(id, userID, req.Type, req.TitleName, req.TaxNumber, req.BankName, req.BankAccount, req.Address, req.Phone)
	if err != nil {
		slog.Error("failed to update invoice title", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "failed to update title", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(title)
}

// HandleDeleteTitle handles DELETE /api/invoice/titles/{id}
func (h *Handler) HandleDeleteTitle(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	id, err := extractTitleID(r.URL.Path)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid title id", "invalid_request", "bad_id")
		return
	}

	if err := h.store.DeleteInvoiceTitle(id, userID); err != nil {
		if strings.Contains(err.Error(), "referenced") {
			httputil.WriteError(w, http.StatusConflict, "title is referenced by invoice requests", "conflict", "title_in_use")
			return
		}
		slog.Error("failed to delete invoice title", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "failed to delete title", "server_error", "db_error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleSetDefaultTitle handles PUT /api/invoice/titles/{id}/default
func (h *Handler) HandleSetDefaultTitle(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	// Path: /api/invoice/titles/123/default → extract 123
	path := strings.TrimSuffix(r.URL.Path, "/default")
	id, err := extractTitleID(path)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid title id", "invalid_request", "bad_id")
		return
	}

	if err := h.store.SetDefaultInvoiceTitle(id, userID); err != nil {
		slog.Error("failed to set default title", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "failed to set default", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleAvailableOrders handles GET /api/invoice/available-orders
func (h *Handler) HandleAvailableOrders(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	orders, err := h.store.ListAvailableOrders(userID)
	if err != nil {
		slog.Error("failed to list available orders", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list orders", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"orders": orders})
}

// HandleCreateRequest handles POST /api/invoice/requests
func (h *Handler) HandleCreateRequest(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	var req struct {
		TitleID     int64    `json:"title_id"`
		InvoiceType string   `json:"invoice_type"`
		OrderIDs    []string `json:"order_ids"`
		Remark      string   `json:"remark"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request", "invalid_request", "bad_json")
		return
	}

	if len(req.OrderIDs) == 0 {
		httputil.WriteError(w, http.StatusBadRequest, "at least one order is required", "invalid_request", "no_orders")
		return
	}
	if req.InvoiceType != "normal" && req.InvoiceType != "special" {
		httputil.WriteError(w, http.StatusBadRequest, "invoice_type must be normal or special", "invalid_request", "bad_invoice_type")
		return
	}

	// Validate title belongs to user and type constraints
	title, err := h.store.GetInvoiceTitle(req.TitleID, userID)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "title not found", "invalid_request", "bad_title")
		return
	}
	if title.Type == "personal" && req.InvoiceType == "special" {
		httputil.WriteError(w, http.StatusBadRequest, "personal title cannot request special invoice", "invalid_request", "personal_no_special")
		return
	}
	if req.InvoiceType == "special" {
		if title.BankName == "" || title.BankAccount == "" || title.Address == "" || title.Phone == "" {
			httputil.WriteError(w, http.StatusBadRequest, "special invoice requires bank_name, bank_account, address, phone", "invalid_request", "incomplete_title")
			return
		}
	}

	invoiceReq, err := h.store.CreateInvoiceRequest(userID, req.TitleID, req.InvoiceType, req.Remark, req.OrderIDs)
	if err != nil {
		slog.Error("failed to create invoice request", "error", err)
		httputil.WriteError(w, http.StatusBadRequest, err.Error(), "invalid_request", "create_failed")
		return
	}

	// Rule-based triage: mark auto_ok / needs_review for the admin queue.
	EvaluateRisk(h.store, invoiceReq)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(invoiceReq)
}

// HandleListRequests handles GET /api/invoice/requests
func (h *Handler) HandleListRequests(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	requests, total, err := h.store.ListInvoiceRequests(userID, size, offset)
	if err != nil {
		slog.Error("failed to list invoice requests", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list requests", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"requests": requests,
		"total":    total,
		"page":     page,
		"size":     size,
	})
}

// HandleGetRequest handles GET /api/invoice/requests/{id}
func (h *Handler) HandleGetRequest(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	id, err := extractRequestID(r.URL.Path)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request id", "invalid_request", "bad_id")
		return
	}

	req, err := h.store.GetInvoiceRequest(id, userID)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "request not found", "not_found", "no_request")
		return
	}

	orders, _ := h.store.GetInvoiceRequestOrders(id)
	title, _ := h.store.GetInvoiceTitleByID(req.TitleID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"request": req,
		"orders":  orders,
		"title":   title,
	})
}

// HandleCancelRequest handles PUT /api/invoice/requests/{id}/cancel
func (h *Handler) HandleCancelRequest(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/cancel")
	id, err := extractRequestID(path)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request id", "invalid_request", "bad_id")
		return
	}

	if err := h.store.CancelInvoiceRequest(id, userID); err != nil {
		slog.Error("failed to cancel invoice request", "error", err)
		httputil.WriteError(w, http.StatusBadRequest, err.Error(), "invalid_request", "cancel_failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
}

// HandleDownload handles GET /api/invoice/requests/{id}/download
func (h *Handler) HandleDownload(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "auth_error", "no_user")
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/download")
	id, err := extractRequestID(path)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request id", "invalid_request", "bad_id")
		return
	}

	req, err := h.store.GetInvoiceRequest(id, userID)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "request not found", "not_found", "no_request")
		return
	}

	if req.Status != "completed" || req.InvoiceFilePath == "" {
		httputil.WriteError(w, http.StatusBadRequest, "invoice not ready", "invalid_request", "not_completed")
		return
	}

	if h.tos == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "storage not configured", "server_error", "no_storage")
		return
	}

	if !strings.HasPrefix(req.InvoiceFilePath, "invoices/") {
		httputil.WriteError(w, http.StatusGone, "invoice file unavailable, please contact admin", "gone", "legacy_file")
		return
	}

	ctx := context.Background()
	signedURL, err := h.tos.PreSignedURL(ctx, req.InvoiceFilePath, 5*time.Minute)
	if err != nil {
		slog.Error("failed to generate presigned URL", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "failed to generate download link", "server_error", "presign_error")
		return
	}

	safeName := sanitizeFilename(req.InvoiceNumber)
	if safeName == "" {
		safeName = fmt.Sprintf("%d", id)
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="invoice-%s.pdf"`, safeName))
	http.Redirect(w, r, signedURL, http.StatusFound)
}

var safeFilenameRe = regexp.MustCompile(`[^a-zA-Z0-9\-_]`)

func sanitizeFilename(s string) string {
	return safeFilenameRe.ReplaceAllString(s, "")
}

// extractTitleID extracts title ID from path like /api/invoice/titles/123
func extractTitleID(path string) (int64, error) {
	parts := strings.Split(strings.TrimRight(path, "/"), "/")
	if len(parts) == 0 {
		return 0, fmt.Errorf("no id in path")
	}
	return strconv.ParseInt(parts[len(parts)-1], 10, 64)
}

// extractRequestID extracts request ID from path like /api/invoice/requests/123
func extractRequestID(path string) (int64, error) {
	return extractTitleID(path) // same logic
}
