package admin

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/zhulang/llm-gateway/internal/config"
	"github.com/zhulang/llm-gateway/internal/email"
	"github.com/zhulang/llm-gateway/internal/httputil"
	"github.com/zhulang/llm-gateway/internal/sms"
	"github.com/zhulang/llm-gateway/internal/store"
)

type loginAttempt struct {
	failures int
	lockedAt time.Time
}

// AuthHandler handles login, registration, and SMS/Email verification.
type AuthHandler struct {
	store                 store.Store
	jwtSecret             []byte
	smsSender             sms.Sender
	smsCodeStore          *sms.CodeStore
	smsTemplates          config.SMSTemplates
	emailSender           email.Sender
	emailCodeStore        *email.CodeStore
	emailTemplates        config.EmailTemplates
	trialCreditsCNY       float64
	firstRechargeBonusCNY float64
	adminInitToken        string

	loginMu       sync.Mutex
	loginAttempts map[string]*loginAttempt
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(
	s store.Store,
	jwtSecret string,
	smsSender sms.Sender,
	smsCodeStore *sms.CodeStore,
	smsTemplates config.SMSTemplates,
	emailSender email.Sender,
	emailCodeStore *email.CodeStore,
	emailTemplates config.EmailTemplates,
	trialCreditsCNY, firstRechargeBonusCNY float64,
	adminInitToken string,
) *AuthHandler {
	return &AuthHandler{
		store:                 s,
		jwtSecret:             []byte(jwtSecret),
		smsSender:             smsSender,
		smsCodeStore:          smsCodeStore,
		smsTemplates:          smsTemplates,
		emailSender:           emailSender,
		emailCodeStore:        emailCodeStore,
		emailTemplates:        emailTemplates,
		trialCreditsCNY:       trialCreditsCNY,
		firstRechargeBonusCNY: firstRechargeBonusCNY,
		adminInitToken:        adminInitToken,
		loginAttempts:         make(map[string]*loginAttempt),
	}
}

type loginRequest struct {
	Phone      string `json:"phone,omitempty"`
	Email      string `json:"email,omitempty"`
	Identifier string `json:"identifier,omitempty"` // 统一标识符（自动识别手机号或邮箱）
	Password   string `json:"password"`
	Remember   bool   `json:"remember"`
}

type loginResponse struct {
	Token string `json:"token"`
	Phone string `json:"phone,omitempty"`
	Email string `json:"email,omitempty"`
}

// HandleLogin authenticates and returns a JWT in a cookie.
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}

	// 统一标识符（优先使用 identifier，否则使用 phone 或 email）
	identifier := req.Identifier
	if identifier == "" {
		if req.Email != "" {
			identifier = req.Email
		} else if req.Phone != "" {
			identifier = req.Phone
		}
	}

	if identifier == "" {
		httputil.WriteError(w, http.StatusBadRequest, "请提供手机号或邮箱", "invalid_request_error", "missing_identifier")
		return
	}

	// Check account lockout.
	h.loginMu.Lock()
	attempt, exists := h.loginAttempts[identifier]
	if exists && attempt.failures >= 5 && time.Since(attempt.lockedAt) < 15*time.Minute {
		h.loginMu.Unlock()
		httputil.WriteError(w, http.StatusTooManyRequests, "登录失败次数过多，请15分钟后再试", "auth_error", "account_locked")
		return
	}
	if exists && time.Since(attempt.lockedAt) >= 15*time.Minute {
		delete(h.loginAttempts, identifier)
	}
	h.loginMu.Unlock()

	user, err := h.store.AuthenticateByIdentifier(identifier, req.Password)
	if err != nil {
		h.loginMu.Lock()
		a, ok := h.loginAttempts[identifier]
		if !ok {
			a = &loginAttempt{}
			h.loginAttempts[identifier] = a
		}
		a.failures++
		a.lockedAt = time.Now()
		h.loginMu.Unlock()
		slog.Warn("login failed", "identifier", identifier)
		httputil.WriteError(w, http.StatusUnauthorized, "账号或密码错误", "auth_error", "invalid_credentials")
		return
	}

	// Reset on successful login.
	h.loginMu.Lock()
	delete(h.loginAttempts, identifier)
	h.loginMu.Unlock()

	expiration := 24 * time.Hour
	if req.Remember {
		expiration = 30 * 24 * time.Hour
	}

	// 构建 JWT Claims（包含 phone 和 email，哪个有值就包含哪个）
	claims := jwt.MapClaims{
		"sub":  user.ID,
		"role": user.Role,
		"exp":  time.Now().Add(expiration).Unix(),
	}
	if user.Phone != "" {
		claims["phone"] = user.Phone
	}
	if user.Email != "" {
		claims["email"] = user.Email
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(h.jwtSecret)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to generate token", "server_error", "jwt_error")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    tokenStr,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(expiration.Seconds()),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loginResponse{
		Token: tokenStr,
		Phone: user.Phone,
		Email: user.Email,
	})
}

// HandleLogout clears the auth cookie.
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleMe returns the current user info from JWT.
func (h *AuthHandler) HandleMe(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(CtxUserIDKey).(string)
	resp := map[string]any{
		"phone":   r.Context().Value(CtxPhoneKey),
		"email":   r.Context().Value(CtxEmailKey),
		"role":    r.Context().Value(CtxRoleKey),
		"user_id": userID,
	}
	if userID != "" {
		if u, err := h.store.GetUserByID(userID); err == nil {
			resp["first_recharge_bonus_granted"] = u.FirstRechargeBonusGranted
			resp["image_share_enabled"] = u.ImageShareEnabled
			resp["email_verified"] = u.EmailVerified
		}
	}
	if h.firstRechargeBonusCNY > 0 {
		resp["first_recharge_bonus_cny"] = h.firstRechargeBonusCNY
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// CtxKey is a context key type.
type CtxKey string

const (
	// CtxUsernameKey is kept for backward compatibility.
	CtxUsernameKey CtxKey = "username"
	// CtxPhoneKey is the context key for the authenticated phone.
	CtxPhoneKey CtxKey = "phone"
	// CtxEmailKey is the context key for the authenticated email.
	CtxEmailKey CtxKey = "email"
	// CtxRoleKey is the context key for the authenticated role.
	CtxRoleKey CtxKey = "role"
	// CtxUserIDKey is the context key for the authenticated user ID.
	CtxUserIDKey CtxKey = "user_id"
	// CtxKeyIDKey is the context key for the API key ID.
	CtxKeyIDKey CtxKey = "key_id"
	// CtxTenantMemberKey is the context key for the authenticated tenant member.
	CtxTenantMemberKey CtxKey = "tenant_member"
	// CtxImageShareKeyID is the context key for the image-share key id (when role==image_share).
	CtxImageShareKeyID CtxKey = "image_share_key_id"
	// CtxImageShareOwnerID is the context key for the image-share key owner user id.
	CtxImageShareOwnerID CtxKey = "image_share_owner_id"
)

// phoneRegex validates Chinese mobile phone numbers.
var phoneRegex = regexp.MustCompile(`^1[3-9]\d{9}$`)

// emailRegex validates email addresses.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

type sendCodeRequest struct {
	Phone   string `json:"phone,omitempty"`
	Email   string `json:"email,omitempty"`
	Purpose string `json:"purpose"` // register, reset_password
}

type registerRequest struct {
	Phone        string `json:"phone,omitempty"`
	Email        string `json:"email,omitempty"`
	Code         string `json:"code"`
	Password     string `json:"password"`
	AdminToken   string `json:"admin_token,omitempty"`
	AcceptAUP    bool   `json:"accept_aup"`
	ReferralCode string `json:"referral_code,omitempty"`
}

type resetPasswordRequest struct {
	Phone       string `json:"phone,omitempty"`
	Email       string `json:"email,omitempty"`
	Code        string `json:"code"`
	NewPassword string `json:"new_password"`
}

// HandleSendCode sends a verification code (SMS or Email) to the given phone/email.
// POST /api/sms/send  body: {"phone": "...", "email": "...", "purpose": "register|reset_password"}
func (h *AuthHandler) HandleSendCode(w http.ResponseWriter, r *http.Request) {
	var req sendCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}

	// 判断是手机号还是邮箱
	if req.Email != "" {
		h.handleSendEmailCode(w, req.Email, req.Purpose)
	} else if req.Phone != "" {
		h.handleSendSMSCode(w, req.Phone, req.Purpose)
	} else {
		httputil.WriteError(w, http.StatusBadRequest, "请提供手机号或邮箱", "invalid_request_error", "missing_identifier")
	}
}

// handleSendSMSCode sends SMS verification code
func (h *AuthHandler) handleSendSMSCode(w http.ResponseWriter, phone, purpose string) {
	if !phoneRegex.MatchString(phone) {
		httputil.WriteError(w, http.StatusBadRequest, "手机号格式不正确", "invalid_request_error", "invalid_phone")
		return
	}

	// Resolve template ID and param key from purpose.
	var templateID, paramKey string
	switch purpose {
	case "reset_password":
		templateID = h.smsTemplates.ResetPassword.ID
		paramKey = h.smsTemplates.ResetPassword.ParamKey
	case "register", "":
		templateID = h.smsTemplates.Register.ID
		paramKey = h.smsTemplates.Register.ParamKey
	default:
		httputil.WriteError(w, http.StatusBadRequest, "无效的用途类型", "invalid_request_error", "invalid_purpose")
		return
	}

	code := sms.GenerateCode()

	// Rate limit check (1 per 60 seconds per phone).
	if err := h.smsCodeStore.Set(phone, code); err != nil {
		httputil.WriteError(w, http.StatusTooManyRequests, err.Error(), "rate_limit_error", "too_frequent")
		return
	}

	// Send the code via SMS.
	if err := h.smsSender.SendCode(phone, code, templateID, paramKey); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "短信发送失败", "server_error", "sms_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
}

// handleSendEmailCode sends email verification code
func (h *AuthHandler) handleSendEmailCode(w http.ResponseWriter, emailAddr, purpose string) {
	if !emailRegex.MatchString(emailAddr) {
		httputil.WriteError(w, http.StatusBadRequest, "邮箱格式不正确", "invalid_request_error", "invalid_email")
		return
	}

	// Resolve template from purpose.
	var template config.EmailTemplate
	switch purpose {
	case "reset_password":
		template = h.emailTemplates.ResetPassword
	case "register", "":
		template = h.emailTemplates.Register
	default:
		httputil.WriteError(w, http.StatusBadRequest, "无效的用途类型", "invalid_request_error", "invalid_purpose")
		return
	}

	code := email.GenerateCode()

	// Rate limit check (1 per 60 seconds per email).
	if err := h.emailCodeStore.Set(emailAddr, code); err != nil {
		httputil.WriteError(w, http.StatusTooManyRequests, err.Error(), "rate_limit_error", "too_frequent")
		return
	}

	// Send the code via Email.
	if err := h.emailSender.SendCode(emailAddr, code, template.Subject, template.TemplateID, template.ParamKey); err != nil {
		slog.Error("email send failed", "error", err, "email", emailAddr)
		httputil.WriteError(w, http.StatusInternalServerError, "邮件发送失败", "server_error", "email_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
}

// HandleRegister registers a new user after verifying the verification code.
// POST /api/register
func (h *AuthHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}

	// 互斥验证：phone 和 email 只能提供其中一个
	if req.Phone != "" && req.Email != "" {
		httputil.WriteError(w, http.StatusBadRequest, "手机号和邮箱只能提供其中一个", "invalid_request_error", "conflicting_identifiers")
		return
	}
	if req.Phone == "" && req.Email == "" {
		httputil.WriteError(w, http.StatusBadRequest, "请提供手机号或邮箱", "invalid_request_error", "missing_identifier")
		return
	}

	// 验证格式
	isEmailReg := req.Email != ""
	if isEmailReg {
		if !emailRegex.MatchString(req.Email) {
			httputil.WriteError(w, http.StatusBadRequest, "邮箱格式不正确", "invalid_request_error", "invalid_email")
			return
		}
	} else {
		if !phoneRegex.MatchString(req.Phone) {
			httputil.WriteError(w, http.StatusBadRequest, "手机号格式不正确", "invalid_request_error", "invalid_phone")
			return
		}
	}

	if len(req.Password) < 8 {
		httputil.WriteError(w, http.StatusBadRequest, "密码长度不能少于8位", "invalid_request_error", "weak_password")
		return
	}

	if req.Code == "" {
		httputil.WriteError(w, http.StatusBadRequest, "验证码不能为空", "invalid_request_error", "missing_code")
		return
	}

	// 验证验证码
	var codeValid bool
	if isEmailReg {
		codeValid = h.emailCodeStore.Verify(req.Email, req.Code)
	} else {
		codeValid = h.smsCodeStore.Verify(req.Phone, req.Code)
	}
	if !codeValid {
		httputil.WriteError(w, http.StatusBadRequest, "验证码错误或已过期", "invalid_request_error", "invalid_code")
		return
	}

	// Require AUP acceptance.
	if !req.AcceptAUP {
		httputil.WriteError(w, http.StatusBadRequest, "请阅读并同意使用条款", "invalid_request_error", "aup_not_accepted")
		return
	}

	// 检查用户是否已存在
	if isEmailReg {
		if _, err := h.store.GetUserByEmail(req.Email); err == nil {
			httputil.WriteError(w, http.StatusConflict, "该邮箱已注册", "invalid_request_error", "email_exists")
			return
		}
	} else {
		if _, err := h.store.GetUserByPhone(req.Phone); err == nil {
			httputil.WriteError(w, http.StatusConflict, "该手机号已注册", "invalid_request_error", "phone_exists")
			return
		}
	}

	// Determine role: first user can become admin only with correct init_token.
	role := "user"
	count, err := h.store.UserCount()
	if err == nil && count == 0 {
		if h.adminInitToken != "" && req.AdminToken == h.adminInitToken {
			role = "admin"
			identifier := req.Phone
			if isEmailReg {
				identifier = req.Email
			}
			slog.Info("first user registered as admin via init_token", "identifier", identifier)
		} else if h.adminInitToken == "" {
			slog.Warn("first user registered without admin role (admin.init_token not configured)")
		} else {
			slog.Info("first user registered as normal user (admin_token not provided or incorrect)")
		}
	}

	// Create user.
	var user *store.User
	if isEmailReg {
		user, err = h.store.CreateUserWithEmail(req.Email, req.Password, role)
		if err == nil {
			// 邮箱注册后标记为已验证（因为验证码已验证）
			if err := h.store.MarkEmailVerified(user.ID); err != nil {
				slog.Error("failed to mark email verified", "user_id", user.ID, "error", err)
			}
		}
	} else {
		user, err = h.store.CreateUser(req.Phone, req.Password, role)
	}
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "注册失败", "server_error", "db_error")
		return
	}

	// Record AUP acceptance.
	if err := h.store.MarkAUPAccepted(user.ID); err != nil {
		slog.Error("failed to record AUP acceptance", "user_id", user.ID, "error", err)
	}

	// Link referral relationship if a valid invite code was provided.
	if req.ReferralCode != "" {
		if referrerID, err := h.store.GetUserIDByReferralCode(strings.ToUpper(strings.TrimSpace(req.ReferralCode))); err == nil {
			if err := h.store.SetReferredBy(user.ID, referrerID); err != nil {
				slog.Warn("failed to set referred_by", "user_id", user.ID, "error", err)
			}
		} else {
			slog.Info("register with unknown referral code", "code", req.ReferralCode)
		}
	}

	// Grant trial credits if configured.
	if h.trialCreditsCNY > 0 {
		if err := h.store.Recharge(user.ID, h.trialCreditsCNY, "注册赠送试用额度"); err != nil {
			slog.Error("failed to grant trial credits", "user_id", user.ID, "error", err)
		}
	}

	// Generate JWT for the new user.
	claims := jwt.MapClaims{
		"sub":  user.ID,
		"role": user.Role,
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
	}
	if user.Phone != "" {
		claims["phone"] = user.Phone
	}
	if user.Email != "" {
		claims["email"] = user.Email
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(h.jwtSecret)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "failed to generate token", "server_error", "jwt_error")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    tokenStr,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"token": tokenStr,
		"user": map[string]any{
			"id":    user.ID,
			"phone": user.Phone,
			"email": user.Email,
			"role":  user.Role,
		},
	})
}

// HandleResetPassword resets the user's password after verifying the verification code.
// POST /api/reset-password
func (h *AuthHandler) HandleResetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid request body", "invalid_request_error", "bad_json")
		return
	}

	// 互斥验证：phone 和 email 只能提供其中一个
	if req.Phone != "" && req.Email != "" {
		httputil.WriteError(w, http.StatusBadRequest, "手机号和邮箱只能提供其中一个", "invalid_request_error", "conflicting_identifiers")
		return
	}
	if req.Phone == "" && req.Email == "" {
		httputil.WriteError(w, http.StatusBadRequest, "请提供手机号或邮箱", "invalid_request_error", "missing_identifier")
		return
	}

	isEmailReset := req.Email != ""
	if isEmailReset {
		if !emailRegex.MatchString(req.Email) {
			httputil.WriteError(w, http.StatusBadRequest, "邮箱格式不正确", "invalid_request_error", "invalid_email")
			return
		}
	} else {
		if !phoneRegex.MatchString(req.Phone) {
			httputil.WriteError(w, http.StatusBadRequest, "手机号格式不正确", "invalid_request_error", "invalid_phone")
			return
		}
	}

	if len(req.NewPassword) < 6 {
		httputil.WriteError(w, http.StatusBadRequest, "密码长度不能少于6位", "invalid_request_error", "weak_password")
		return
	}

	if req.Code == "" {
		httputil.WriteError(w, http.StatusBadRequest, "验证码不能为空", "invalid_request_error", "missing_code")
		return
	}

	// 验证验证码
	var codeValid bool
	if isEmailReset {
		codeValid = h.emailCodeStore.Verify(req.Email, req.Code)
	} else {
		codeValid = h.smsCodeStore.Verify(req.Phone, req.Code)
	}
	if !codeValid {
		httputil.WriteError(w, http.StatusBadRequest, "验证码错误或已过期", "invalid_request_error", "invalid_code")
		return
	}

	// Check if the user exists.
	if isEmailReset {
		if _, err := h.store.GetUserByEmail(req.Email); err != nil {
			httputil.WriteError(w, http.StatusNotFound, "该邮箱未注册", "invalid_request_error", "user_not_found")
			return
		}
	} else {
		if _, err := h.store.GetUserByPhone(req.Phone); err != nil {
			httputil.WriteError(w, http.StatusNotFound, "该手机号未注册", "invalid_request_error", "user_not_found")
			return
		}
	}

	// Update the password.
	var err error
	if isEmailReset {
		err = h.store.UpdatePasswordByEmail(req.Email, req.NewPassword)
	} else {
		err = h.store.UpdatePassword(req.Phone, req.NewPassword)
	}
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "密码重置失败", "server_error", "db_error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandlePromotionRules returns the current promotion/reward rules.
// GET /api/promotion/rules (authenticated)
func (h *AuthHandler) HandlePromotionRules(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rules := []map[string]any{}

	if h.trialCreditsCNY > 0 {
		rules = append(rules, map[string]any{
			"type":        "trial_credits",
			"title":       "注册赠送",
			"description": fmt.Sprintf("新用户注册即赠送 ¥%.2f 试用额度", h.trialCreditsCNY),
			"amount":      h.trialCreditsCNY,
		})
	}

	if h.firstRechargeBonusCNY > 0 {
		rules = append(rules, map[string]any{
			"type":        "first_recharge_bonus",
			"title":       "首充奖励",
			"description": fmt.Sprintf("首次充值赠送 %.0f%%（充多少送多少）", h.firstRechargeBonusCNY*100),
			"amount":      h.firstRechargeBonusCNY,
		})
	}

	if promo, err := h.store.GetCurrentActiveRechargePromotion(time.Now()); err == nil && promo != nil {
		desc := fmt.Sprintf("活动期间充值赠送 %s%%", trimTrailingZeros(promo.BonusRatio*100))
		if promo.MinRechargeAmount > 0 {
			desc += fmt.Sprintf("，最低充值 ¥%.2f", promo.MinRechargeAmount)
		}
		rules = append(rules, map[string]any{
			"type":          "recharge_bonus",
			"title":         promo.Name,
			"description":   desc,
			"amount":        promo.BonusRatio * 100,
			"bonus_ratio":   promo.BonusRatio,
			"starts_at":     promo.StartsAt,
			"ends_at":       promo.EndsAt,
			"min_recharge":  promo.MinRechargeAmount,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"rules": rules})
}

// trimTrailingZeros formats a float for display, dropping trailing zero decimals
// (10.0 → "10", 12.5 → "12.5").
func trimTrailingZeros(v float64) string {
	s := strconv.FormatFloat(v, 'f', 2, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" {
		return "0"
	}
	return s
}

// HandleSetupStatus returns system setup readiness.
// GET /api/system/setup-status (public, no auth required)
func (h *AuthHandler) HandleSetupStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	count, err := h.store.UserCount()
	needsAdminSetup := err == nil && count == 0 && h.adminInitToken != ""

	models, _ := h.store.ListModels()
	pricing, _ := h.store.ListActivePricing()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"needs_admin_setup":  needsAdminSetup,
		"models_configured":  len(models) > 0,
		"pricing_configured": len(pricing) > 0,
	})
}
