package sqlstore

import (
	"context"
	"time"

	"github.com/dkeysil/cross-nostter/internal/domain"
	"github.com/jmoiron/sqlx"
)

func (s *SqlStore) UpsertTelegramChannel(ctx context.Context, channel domain.TelegramChannel) error {
	channel.CreatedAt = time.Now()
	channel.UpdatedAt = time.Now()

	query := `
		INSERT INTO telegram_channels (
			channel_id,
			channel_name,
			channel_username,
			channel_description,
			channel_photo_url,
			is_bot_added,
			channel_created_at,
			channel_updated_at
		) VALUES (
			:channel_id,
			:channel_name,
			:channel_username,
			:channel_description,
			:channel_photo_url,
			:is_bot_added,
			:channel_created_at,
			:channel_updated_at
		) ON CONFLICT (channel_id) DO UPDATE SET
			channel_name = :channel_name,
			channel_username = :channel_username,
			channel_description = :channel_description,
			channel_photo_url = :channel_photo_url,
			is_bot_added = :is_bot_added,
			channel_updated_at = :channel_updated_at
	`

	query, args, err := sqlx.Named(query, &channel)
	if err != nil {
		return err
	}

	query = s.db.Rebind(query)

	_, err = s.db.ExecContext(
		ctx,
		query,
		args...,
	)

	return err
}

func (s *SqlStore) GetTelegramChannel(ctx context.Context, username string) (*domain.TelegramChannel, error) {
	var channel domain.TelegramChannel
	err := s.db.GetContext(
		ctx,
		&channel,
		"SELECT channel_id, channel_name, channel_username, channel_description, channel_photo_url, is_bot_added FROM telegram_channels WHERE channel_username = $1 AND is_bot_added = true",
		username,
	)

	return &channel, err
}
