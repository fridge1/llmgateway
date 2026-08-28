-- Support tickets: user-submitted issues with a message thread.
CREATE TABLE IF NOT EXISTS tickets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category VARCHAR(32) NOT NULL DEFAULT 'other',
    subject VARCHAR(200) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'open',   -- open / pending / resolved / closed
    related_order_no VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_tickets_user ON tickets (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets (status, created_at DESC);

CREATE TABLE IF NOT EXISTS ticket_messages (
    id BIGSERIAL PRIMARY KEY,
    ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    sender_role VARCHAR(16) NOT NULL,             -- user / admin
    sender_id UUID,
    content TEXT NOT NULL,
    attachments JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_ticket_messages_ticket ON ticket_messages (ticket_id, created_at);
