-- +goose Up
CREATE TABLE telegram_channels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id BIGINT UNIQUE,
    channel_name VARCHAR(255),
    channel_username VARCHAR(255) NOT NULL,
    channel_description TEXT,
    channel_photo_url TEXT,
    is_bot_added BOOLEAN NOT NULL DEFAULT FALSE,
    channel_created_at TIMESTAMP NOT NULL,
    channel_updated_at TIMESTAMP NOT NULL
);

CREATE TABLE nostr_accounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    npub TEXT NOT NULL,
    nsec TEXT NOT NULL,
    telegram_channel_id INTEGER NOT NULL UNIQUE,
    cross_posting_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    FOREIGN KEY(telegram_channel_id) REFERENCES telegram_channels(id)
);

-- +goose Down

ALTER TABLE nostr_accounts
DROP TABLE nostr_accounts;
DROP TABLE telegram_channels;


