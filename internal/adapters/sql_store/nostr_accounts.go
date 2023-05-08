package sqlstore

import (
	"context"

	"github.com/dkeysil/cross-nostter/internal/domain"
)

func (s *SqlStore) SetNostrAccount(ctx context.Context, nostrAccount domain.NostrAccount) error {
	_, err := s.db.ExecContext(
		ctx,
		"INSERT INTO nostr_accounts (npub, nsec, telegram_channel_id, cross_posting_enabled) VALUES ($1, $2, $3, $4) ON CONFLICT (telegram_channel_id) DO UPDATE SET npub = $1, nsec = $2, cross_posting_enabled = $4",
		nostrAccount.Npub,
		nostrAccount.Nsec,
		nostrAccount.TelegramChannelID,
		nostrAccount.CrossPostingEnabled,
	)
	return err
}

func (s *SqlStore) GetNsecByChannelID(ctx context.Context, channelID int64) (string, error) {
	var nsec string
	err := s.db.GetContext(
		ctx,
		&nsec,
		"SELECT nsec FROM nostr_accounts WHERE telegram_channel_id = $1",
		channelID,
	)
	return nsec, err
}
