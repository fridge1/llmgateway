-- Outbound webhooks feature removed: drop delivery log first (FK to endpoints).
DROP TABLE IF EXISTS webhook_deliveries;
DROP TABLE IF EXISTS webhook_endpoints;
