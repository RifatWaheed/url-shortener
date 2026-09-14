CREATE TABLE clicks (
    id         BIGSERIAL PRIMARY KEY,
    link_id    BIGINT NOT NULL REFERENCES links(id) ON DELETE CASCADE,
    clicked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    referrer   TEXT,
    user_agent TEXT,
    ip_address INET,
    country    TEXT
);

CREATE INDEX idx_clicks_link_id ON clicks (link_id);