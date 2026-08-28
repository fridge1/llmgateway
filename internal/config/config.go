package config

import (
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"
	"time"

	"gopkg.in/yaml.v3"
)

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port                int             `yaml:"port"`
	AuthTokens          []string        `yaml:"auth_tokens"`
	AdminToken          string          `yaml:"admin_token"`
	RequestTimeout      time.Duration   `yaml:"request_timeout"`
	MaxRequestBodyBytes int64           `yaml:"max_request_body_bytes"`
	ShutdownTimeout     time.Duration   `yaml:"shutdown_timeout"`
	CORSOrigins         []string        `yaml:"cors_origins"`
	// CORSMode controls CORS behaviour: "permissive" (default) allows all when cors_origins is empty,
	// "strict" rejects cross-origin requests when cors_origins is empty.
	CORSMode            string          `yaml:"cors_mode"`
	LogLevel            string          `yaml:"log_level"`
	RateLimit           RateLimitConfig `yaml:"rate_limit"`
	// NoPricingStrategy controls behaviour when a model has no pricing configured.
	// "allow" = serve without billing (current default), "reject" = return 403, "warn" = allow + warn log.
	NoPricingStrategy string `yaml:"no_pricing_strategy"`
	// ImageBandwidth controls concurrent image response delivery to avoid bandwidth saturation.
	ImageBandwidth ImageBandwidthConfig `yaml:"image_bandwidth"`
	// ImageUpstreamMaxConcurrency caps the number of concurrent in-flight upstream
	// image generation/edit calls across all async workers. 0 disables the gate.
	ImageUpstreamMaxConcurrency int `yaml:"image_upstream_max_concurrency"`
}

// ImageBandwidthConfig limits how many image responses can be written to clients simultaneously.
type ImageBandwidthConfig struct {
	MaxConcurrent int           `yaml:"max_concurrent"` // 0 = disabled
	QueueTimeout  time.Duration `yaml:"queue_timeout"`  // default 5m
}

// RateLimitConfig holds rate limiting settings.
type RateLimitConfig struct {
	RequestsPerSecond float64       `yaml:"requests_per_second"`
	Burst             int           `yaml:"burst"`
	CleanupInterval   time.Duration `yaml:"cleanup_interval"`
	// Per-dimension rate limits (0 = disabled).
	PerUserRPS float64 `yaml:"per_user_rps"`
	PerUserBurst int   `yaml:"per_user_burst"`
	PerKeyRPS  float64 `yaml:"per_key_rps"`
	PerKeyBurst int    `yaml:"per_key_burst"`
}

// UpstreamConfig describes a single upstream LLM provider endpoint.
//
// Provider 既用于 UI 分组（OpenAI/Anthropic/Gemini 等），也是路由协议的
// 默认推断来源。Protocols 是该上游支持的协议数组（如 ["openai","anthropic"]），
// 入口路由层据此挑选同协议上游做透传，或退到 openai/openai-compatible 上游做协议转换兜底。
// 旧的单值 Protocol 字段保留：当 Protocols 为空时回退到 Protocol（兼容老数据）。
type UpstreamConfig struct {
	Provider         string   `yaml:"provider"`
	Protocol         string   `yaml:"protocol,omitempty"`
	Protocols        []string `yaml:"protocols,omitempty,flow"`
	UpstreamProvider string   `yaml:"upstream_provider,omitempty"`
	UpstreamName     string   `yaml:"upstream_name,omitempty"`
	BaseURL          string   `yaml:"base_url"`
	APIKey           string   `yaml:"api_key"`
	ModelOverride    string   `yaml:"model_override,omitempty"`
	Weight           int      `yaml:"weight"`
}

// ModelConfig maps a logical model name to one or more upstream providers.
type ModelConfig struct {
	Name        string           `yaml:"name"`
	DisplayName string           `yaml:"display_name,omitempty"`
	Upstreams   []UpstreamConfig `yaml:"upstreams"`
}

// TenantModelConfig is a tenant-specific upstream override for one model.
// Requests authenticated with that tenant's keys route exclusively to these
// upstreams (no fallback to the global pool). Populated from the database
// only, never from YAML.
type TenantModelConfig struct {
	TenantID  string           `yaml:"-"`
	ModelName string           `yaml:"-"`
	Upstreams []UpstreamConfig `yaml:"-"`
}

// CircuitBreakerConfig holds circuit-breaker tuning parameters.
type CircuitBreakerConfig struct {
	FailureThreshold    int           `yaml:"failure_threshold"`
	RecoveryTimeout     time.Duration `yaml:"recovery_timeout"`
	HalfOpenMaxRequests int           `yaml:"half_open_max_requests"`

	// Retry configuration
	MaxRetries           int           `yaml:"max_retries"`            // Default 3
	RetryableStatusCodes []int         `yaml:"retryable_status_codes"` // Default [429, 502, 503, 504]
	RetryBaseDelay       time.Duration `yaml:"retry_base_delay"`       // Default 100ms
	RetryMaxDelay        time.Duration `yaml:"retry_max_delay"`        // Default 5s
}

// DatabaseConfig holds database settings.
type DatabaseConfig struct {
	DSN             string        `yaml:"dsn"`
	Path            string        `yaml:"path"` // kept for backward compat, ignored
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

// AdminConfig holds admin-related settings.
type AdminConfig struct {
	JWTSecret string `yaml:"jwt_secret"`
	// InitToken is required to claim admin role during first-user registration.
	// If empty, the first registered user will be a normal user (no auto-admin).
	InitToken string `yaml:"init_token"`
}

// SMSTemplate holds a template ID and its variable key name.
type SMSTemplate struct {
	ID       string `yaml:"id"`
	ParamKey string `yaml:"param_key"` // e.g. "code" or "msg_code"
}

// SMSTemplates holds template configs for different SMS scenarios.
type SMSTemplates struct {
	Register      SMSTemplate `yaml:"register"`
	Login         SMSTemplate `yaml:"login"`
	ResetPassword SMSTemplate `yaml:"reset_password"`
	// Alert is the ops-alert template; its single variable receives the alert name.
	Alert   SMSTemplate `yaml:"alert"`
	Lottery SMSTemplate `yaml:"lottery"` // 抽奖中奖通知模板（可选）
}

// SMSConfig holds SMS provider configuration.
type SMSConfig struct {
	Provider   string       `yaml:"provider"`
	AccessKey  string       `yaml:"access_key"`
	SecretKey  string       `yaml:"secret_key"`
	SignName   string       `yaml:"sign_name"`
	SmsAccount string       `yaml:"sms_account"`
	Templates  SMSTemplates `yaml:"templates"`
}

// EmailTemplate holds a single email template configuration.
type EmailTemplate struct {
	Subject    string `yaml:"subject"`
	TemplateID string `yaml:"template_id"` // 阿里云模板ID
	ParamKey   string `yaml:"param_key"`   // 模板变量名，例如 "code"
}

// EmailTemplates holds template configs for different email scenarios.
type EmailTemplates struct {
	Register      EmailTemplate `yaml:"register"`
	ResetPassword EmailTemplate `yaml:"reset_password"`
}

// EmailConfig holds email provider configuration.
type EmailConfig struct {
	Provider  string         `yaml:"provider"`   // "aliyun"
	AccessKey string         `yaml:"access_key"` // 阿里云 AccessKey ID
	SecretKey string         `yaml:"secret_key"` // 阿里云 AccessKey Secret
	Region    string         `yaml:"region"`     // 区域，例如 "cn-hangzhou"
	FromEmail string         `yaml:"from_email"` // 发信地址，例如 "noreply@yourdomain.com"
	FromName  string         `yaml:"from_name"`  // 发信人名称，例如 "LLM Gateway"
	Templates EmailTemplates `yaml:"templates"`
}

// AlipayConfig holds Alipay payment configuration.
type AlipayConfig struct {
	AppID               string `yaml:"app_id"`
	PrivateKeyPath      string `yaml:"private_key_path"`
	AlipayPublicKeyPath string `yaml:"alipay_public_key_path"`
	NotifyURL           string `yaml:"notify_url"`
	ReturnURL           string `yaml:"return_url"`  // 支付完成后浏览器跳转地址
	QuitURL             string `yaml:"quit_url"`    // WAP 支付取消时跳转地址（为空则回退到 return_url）
	// IsProduction selects Alipay gateway: true = openapi.alipay.com, false = sandbox (openapi.alipaydev.com).
	IsProduction bool `yaml:"is_production"`
}

// PaymentConfig holds payment provider configuration.
type PaymentConfig struct {
	Alipay AlipayConfig `yaml:"alipay"`
}

// PromotionConfig holds growth/marketing promotion settings.
type PromotionConfig struct {
	// TrialCreditsCNY is the CNY amount credited to every new user upon registration.
	// Set to 0 to disable.
	TrialCreditsCNY float64 `yaml:"trial_credits_cny"`
	// FirstRechargeBonusCNY is the bonus ratio for the first recharge (e.g., 1.0 = 100%).
	// The bonus amount is calculated as: recharge_amount × first_recharge_bonus_cny.
	// Set to 0 to disable.
	FirstRechargeBonusCNY float64 `yaml:"first_recharge_bonus_cny"`
	// ReferralInviterBonusCNY is credited to the inviter when an invited user
	// completes their first recharge. Set to 0 to disable referral rewards.
	ReferralInviterBonusCNY float64 `yaml:"referral_inviter_bonus_cny"`
	// ReferralInviteeBonusCNY is credited to the invited user on their first recharge.
	ReferralInviteeBonusCNY float64 `yaml:"referral_invitee_bonus_cny"`
	// ReferralRewardCNY is the legacy field (deprecated). When set and the new
	// fields are 0, it auto-fills both inviter and invitee bonuses for backward compatibility.
	ReferralRewardCNY float64 `yaml:"referral_reward_cny"`
}

// RetentionConfig holds user-retention / engagement settings.
type RetentionConfig struct {
	// BalanceAlertThresholdCNY: when a user's balance drops below this after a
	// settlement, send a low-balance notification (once per day). 0 disables.
	BalanceAlertThresholdCNY float64 `yaml:"balance_alert_threshold_cny"`
	// WeeklyReportEnabled toggles the Monday usage-report notification.
	WeeklyReportEnabled bool `yaml:"weekly_report_enabled"`
	// WinbackEnabled toggles the silent-user winback notifications.
	WinbackEnabled bool `yaml:"winback_enabled"`
	// CheckinBaseRewardCNY is the day-1 daily check-in reward. Each consecutive
	// day scales up to a 7-day cap; 0 disables monetary reward (still tracks streak).
	CheckinBaseRewardCNY float64 `yaml:"checkin_base_reward_cny"`
}

// AlertingConfig holds ops alerting settings. Rules (thresholds/cooldowns)
// live in the alert_rules table; this only controls the loop itself.
type AlertingConfig struct {
	Enabled bool `yaml:"enabled"`
	// CheckInterval is how often counters are evaluated (default 1m).
	CheckInterval time.Duration `yaml:"check_interval"`
	// AdminPhones overrides SMS recipients; empty = all admin users' phones.
	AdminPhones []string `yaml:"admin_phones"`
}

// TOSConfig holds Volcengine TOS (object storage) configuration.
type TOSConfig struct {
	Endpoint  string `yaml:"endpoint"`
	Region    string `yaml:"region"`
	Bucket    string `yaml:"bucket"`
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	URLPrefix string `yaml:"url_prefix"`
}

// Config is the top-level configuration structure.
type Config struct {
	Server         ServerConfig         `yaml:"server"`
	Models         []ModelConfig        `yaml:"models"`
	TenantModels   []TenantModelConfig  `yaml:"-"` // DB-only tenant upstream overrides
	CircuitBreaker CircuitBreakerConfig `yaml:"circuit_breaker"`
	Database       DatabaseConfig       `yaml:"database"`
	Admin          AdminConfig          `yaml:"admin"`
	SMS            SMSConfig            `yaml:"sms"`
	Email          EmailConfig          `yaml:"email"`
	Payment        PaymentConfig        `yaml:"payment"`
	Promotion      PromotionConfig      `yaml:"promotion"`
	Retention      RetentionConfig      `yaml:"retention"`
	Alerting       AlertingConfig       `yaml:"alerting"`
	TOS            TOSConfig            `yaml:"tos"`
	Billing        BillingConfig        `yaml:"billing"`
}

// BillingConfig holds async billing worker pool settings.
type BillingConfig struct {
	Workers             int `yaml:"workers"`
	QueueSize           int `yaml:"queue_size"`
	OverflowConcurrency int `yaml:"overflow_concurrency"`
}

// LoadFromFile reads a YAML file at path and returns a validated Config.
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read file %q: %w", path, err)
	}
	return LoadFromBytes(data)
}

// LoadFromBytes unmarshals YAML bytes and validates the resulting Config.
// Environment variables in the form ${VAR} or $VAR are expanded before parsing.
func LoadFromBytes(data []byte) (*Config, error) {
	expanded := os.ExpandEnv(string(data))
	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("config: unmarshal yaml: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Validate checks that required fields are present and logs warnings for
// suspicious but non-fatal conditions (e.g. unequal upstream weights).
func (c *Config) Validate() error {
	// auth_tokens is now optional — warn if empty instead of erroring
	if len(c.Server.AuthTokens) == 0 {
		slog.Warn("config: server.auth_tokens is empty; proxy auth will rely on API keys")
	}

	// models may be empty when seeded from DB
	if len(c.Models) == 0 {
		slog.Warn("config: models list is empty; models will be loaded from database")
	}

	for _, m := range c.Models {
		if len(m.Upstreams) < 2 {
			continue
		}
		first := m.Upstreams[0].Weight
		allEqual := true
		for _, u := range m.Upstreams[1:] {
			if u.Weight != first {
				allEqual = false
				break
			}
		}
		if !allEqual {
			slog.Warn("config: upstreams have unequal weights",
				slog.String("model", m.Name))
		}
	}

	// Backward compatibility: migrate legacy referral_reward_cny to new fields
	if c.Promotion.ReferralRewardCNY > 0 {
		if c.Promotion.ReferralInviterBonusCNY == 0 {
			c.Promotion.ReferralInviterBonusCNY = c.Promotion.ReferralRewardCNY
			slog.Info("config: migrated referral_reward_cny to referral_inviter_bonus_cny",
				"amount", c.Promotion.ReferralRewardCNY)
		}
		if c.Promotion.ReferralInviteeBonusCNY == 0 {
			c.Promotion.ReferralInviteeBonusCNY = c.Promotion.ReferralRewardCNY
			slog.Info("config: migrated referral_reward_cny to referral_invitee_bonus_cny",
				"amount", c.Promotion.ReferralRewardCNY)
		}
	}

	// Validate and set defaults for retry configuration
	if c.CircuitBreaker.MaxRetries < 0 {
		return fmt.Errorf("circuit_breaker.max_retries must be >= 0")
	}
	if c.CircuitBreaker.MaxRetries > 10 {
		slog.Warn("circuit_breaker.max_retries is very high, may cause long delays",
			"max_retries", c.CircuitBreaker.MaxRetries)
	}
	if c.CircuitBreaker.RetryBaseDelay <= 0 {
		c.CircuitBreaker.RetryBaseDelay = 100 * time.Millisecond
	}
	if c.CircuitBreaker.RetryMaxDelay <= 0 {
		c.CircuitBreaker.RetryMaxDelay = 5 * time.Second
	}
	if len(c.CircuitBreaker.RetryableStatusCodes) == 0 {
		c.CircuitBreaker.RetryableStatusCodes = []int{429, 502, 503, 504}
	}

	return nil
}

// Holder provides atomic, lock-free access to the current *Config and
// supports hot reload via Swap.
type Holder struct {
	p atomic.Pointer[Config]
}

// NewHolder creates a Holder initialised with cfg.
func NewHolder(cfg *Config) *Holder {
	h := &Holder{}
	h.p.Store(cfg)
	return h
}

// Get returns the currently active Config. It is safe for concurrent use.
func (h *Holder) Get() *Config {
	return h.p.Load()
}

// Swap atomically replaces the active Config with cfg.
func (h *Holder) Swap(cfg *Config) {
	h.p.Store(cfg)
}
