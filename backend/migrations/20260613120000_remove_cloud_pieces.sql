-- +goose Up
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_webhook_records_provider_event_id;
DROP TABLE IF EXISTS webhook_records;

DROP INDEX IF EXISTS idx_subscription_events_subscription_id;
ALTER TABLE subscription_events DROP CONSTRAINT IF EXISTS fk_subscription_events_subscription_id;
DROP TABLE IF EXISTS subscription_events;

DROP INDEX IF EXISTS idx_invoices_provider_invoice_id;
DROP INDEX IF EXISTS idx_invoices_subscription_id;
ALTER TABLE invoices DROP CONSTRAINT IF EXISTS fk_invoices_subscription_id;
DROP TABLE IF EXISTS invoices;

DROP INDEX IF EXISTS idx_subscriptions_provider_sub_id;
DROP INDEX IF EXISTS idx_subscriptions_status;
DROP INDEX IF EXISTS idx_subscriptions_database_id;
DROP TABLE IF EXISTS subscriptions;

ALTER TABLE storages
    DROP COLUMN IF EXISTS is_system;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE storages
    ADD COLUMN is_system BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE subscriptions (
    id                                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    database_id                        UUID NOT NULL,
    status                             TEXT NOT NULL,
    storage_gb                         INT NOT NULL,
    pending_storage_gb                 INT,
    current_period_start               TIMESTAMPTZ NOT NULL,
    current_period_end                 TIMESTAMPTZ NOT NULL,
    canceled_at                        TIMESTAMPTZ,
    data_retention_grace_period_until  TIMESTAMPTZ,
    provider_name                      TEXT,
    provider_sub_id                    TEXT,
    provider_customer_id               TEXT,
    created_at                         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_subscriptions_database_id ON subscriptions (database_id);
CREATE INDEX idx_subscriptions_status ON subscriptions (status);
CREATE INDEX idx_subscriptions_provider_sub_id ON subscriptions (provider_sub_id);

CREATE TABLE invoices (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id     UUID NOT NULL,
    provider_invoice_id TEXT NOT NULL,
    amount_cents        BIGINT NOT NULL,
    storage_gb          INT NOT NULL,
    period_start        TIMESTAMPTZ NOT NULL,
    period_end          TIMESTAMPTZ NOT NULL,
    status              TEXT NOT NULL,
    paid_at             TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE invoices
    ADD CONSTRAINT fk_invoices_subscription_id
    FOREIGN KEY (subscription_id)
    REFERENCES subscriptions (id)
    ON DELETE CASCADE;

CREATE INDEX idx_invoices_subscription_id ON invoices (subscription_id);
CREATE INDEX idx_invoices_provider_invoice_id ON invoices (provider_invoice_id);

CREATE TABLE subscription_events (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id   UUID NOT NULL,
    provider_event_id TEXT,
    type              TEXT NOT NULL,
    old_storage_gb    INT,
    new_storage_gb    INT,
    old_status        TEXT,
    new_status        TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE subscription_events
    ADD CONSTRAINT fk_subscription_events_subscription_id
    FOREIGN KEY (subscription_id)
    REFERENCES subscriptions (id)
    ON DELETE CASCADE;

CREATE INDEX idx_subscription_events_subscription_id ON subscription_events (subscription_id);

CREATE TABLE webhook_records (
    request_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_name     TEXT NOT NULL,
    event_type        TEXT NOT NULL,
    provider_event_id TEXT NOT NULL,
    raw_payload       TEXT NOT NULL,
    processed_at      TIMESTAMPTZ,
    error             TEXT,
    is_skipped        BOOLEAN NOT NULL DEFAULT FALSE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_records_provider_event_id ON webhook_records (provider_event_id);

-- +goose StatementEnd
