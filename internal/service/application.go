package service

import (
	"context"

	nostrAdapter "github.com/dkeysil/cross-nostter/internal/adapters/nostr"
	sqlstore "github.com/dkeysil/cross-nostter/internal/adapters/sql_store"
	uplink "github.com/dkeysil/cross-nostter/internal/adapters/uplink_file_uploader"
	"github.com/dkeysil/cross-nostter/internal/config"
	telegramPort "github.com/dkeysil/cross-nostter/internal/ports/telegram"
	"github.com/dkeysil/cross-nostter/migrations"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

func RunApplication(cfg config.Config) {
	ctx := context.Background()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	upl, err := uplink.NewUplinkFileUploader(ctx, cfg.UplinkFileUploader)
	if err != nil {
		logger.Fatal("error while creating uplink client", zap.Error(err))
	}

	db, err := sqlx.Connect("sqlite3", "crossnostter.db")
	if err != nil {
		logger.Fatal("error while conntecting to the database", zap.Error(err))
	}

	err = migrations.Run(db.DB)
	if err != nil {
		logger.Fatal("error while running migrations", zap.Error(err))
	}
	sqlStore := sqlstore.New(db)

	bot, err := tgbotapi.NewBotAPI(cfg.TelegramBotToken)
	if err != nil {
		logger.Fatal("error while creating telegram bot", zap.Error(err))
	}

	nAdapter := nostrAdapter.NewNostrAdapter(ctx, logger, cfg.Relays)

	tgPort := telegramPort.NewTelegramPort(nAdapter, bot, sqlStore, upl, logger)

	me, err := bot.GetMe()
	if err != nil {
		logger.Fatal("error while getting bot info", zap.Error(err))
	}

	logger.Info("starting application", zap.String("bot_name", me.String()))
	tgPort.Listen(ctx)
	logger.Info("application stopped")
}
