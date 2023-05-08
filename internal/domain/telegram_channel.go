package domain

import "time"

type TelegramChannel struct {
	TelegramID  int64     `db:"channel_id"`
	Name        string    `db:"channel_name"`
	Username    string    `db:"channel_username"`
	Description string    `db:"channel_description"`
	PhotoURL    string    `db:"channel_photo_url"`
	IsBotAdded  bool      `db:"is_bot_added"`
	CreatedAt   time.Time `db:"channel_created_at"`
	UpdatedAt   time.Time `db:"channel_updated_at"`
}
