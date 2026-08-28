-- IP Blocking table for DDoS protection
CREATE TABLE IF NOT EXISTS blocked_ips (
    ip_address VARCHAR(45) PRIMARY KEY,  -- IPv4 (15 chars) or IPv6 (45 chars)
    reason TEXT NOT NULL,                 -- Why this IP was blocked
    blocked_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,                 -- NULL = permanent block
    blocked_by VARCHAR(255),              -- Admin user who blocked (optional)
    notes TEXT                            -- Additional notes
);

-- Index for efficient expiry cleanup
CREATE INDEX IF NOT EXISTS idx_blocked_ips_expires ON blocked_ips(expires_at) WHERE expires_at IS NOT NULL;

-- Index for audit/search by blocked_at
CREATE INDEX IF NOT EXISTS idx_blocked_ips_blocked_at ON blocked_ips(blocked_at DESC);
