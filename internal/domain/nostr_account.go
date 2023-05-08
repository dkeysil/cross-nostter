package domain

type NostrAccount struct {
	Npub                string
	Nsec                string
	TelegramChannelID   int64
	CrossPostingEnabled bool
}
