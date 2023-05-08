package config

import uplink "github.com/dkeysil/cross-nostter/internal/adapters/uplink_file_uploader"

type Config struct {
	TelegramBotToken string `envconfig:"TELEGRAM_BOT_TOKEN"`

	Relays []string `envconfig:"RELAYS" default:"wss://relay.nostr.band/"`

	UplinkFileUploader *uplink.Config `envconfig:"UPLINK"`
}
