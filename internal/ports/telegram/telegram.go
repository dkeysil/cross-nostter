package ports

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	nostrAdapter "github.com/dkeysil/cross-nostter/internal/adapters/nostr"
	sqlstore "github.com/dkeysil/cross-nostter/internal/adapters/sql_store"
	"github.com/dkeysil/cross-nostter/internal/domain"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"go.uber.org/zap"
)

type FileUploader interface {
	UploadFile(ctx context.Context, fileName string, imageURL io.Reader) (url string, err error)
}

type TelegramPort struct {
	store    *sqlstore.SqlStore
	nostr    *nostrAdapter.NostrAdapter
	uploader FileUploader
	logger   *zap.Logger

	bot     *tgbotapi.BotAPI
	updates tgbotapi.UpdatesChannel
}

func NewTelegramPort(
	n *nostrAdapter.NostrAdapter,
	bot *tgbotapi.BotAPI,
	store *sqlstore.SqlStore,
	uploader FileUploader,
	logger *zap.Logger,
) *TelegramPort {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	return &TelegramPort{
		nostr:    n,
		store:    store,
		uploader: uploader,
		logger:   logger,

		bot:     bot,
		updates: updates,
	}
}

func (p *TelegramPort) Listen(ctx context.Context) {
	var mediaGroupID string
	var post *Post
	for {
		select {
		case <-ctx.Done():
			return
		case update := <-p.updates:
			if update.ChannelPost != nil {
				// If ChannelPost contains photo - Photo slice will contain different sizes of the same photo
				// If ChannelPost contains more than one photo or video - MediaGroupID field will be filled and other photos or videos will arrive in the next updates
				// If ChannelPost contains video - Video field will be filled
				if mediaGroupID != update.ChannelPost.MediaGroupID {
					if post != nil {
						go p.PublishMessage(ctx, post)
						post = nil
					}

					mediaGroupID = update.ChannelPost.MediaGroupID
				}

				if post == nil {
					post = &Post{
						ChannelID: update.ChannelPost.Chat.ID,
					}
				}

				if len(update.ChannelPost.Photo) > 0 {
					photoUrl, err := p.bot.GetFileDirectURL(
						update.ChannelPost.Photo[len(update.ChannelPost.Photo)-1].FileID,
					)
					if err != nil {
						continue
					}

					resp, err := http.Get(photoUrl)
					if err != nil {
						continue
					}

					splittedFileName := strings.Split(photoUrl, "/")
					if len(splittedFileName) == 0 {
						continue
					}
					fileName := splittedFileName[len(splittedFileName)-1]

					post.Files = append(post.Files, File{
						Name:   fileName,
						Reader: resp.Body,
					})
				}

				if update.ChannelPost.Video != nil {
					videoUrl, err := p.bot.GetFileDirectURL(update.ChannelPost.Video.FileID)
					if err != nil {
						continue
					}

					splittedFileName := strings.Split(videoUrl, "/")
					if len(splittedFileName) == 0 {
						continue
					}
					fileName := splittedFileName[len(splittedFileName)-1]

					resp, err := http.Get(videoUrl)
					if err != nil {
						continue
					}

					post.Files = append(post.Files, File{
						Name:   fileName,
						Reader: resp.Body,
					})
				}

				if update.ChannelPost.Text != "" {
					post.Text = update.ChannelPost.Text
				}

				if update.ChannelPost.Caption != "" {
					post.Text = update.ChannelPost.Caption
				}

				if mediaGroupID == "" {
					go p.PublishMessage(ctx, post)
					post = nil
				}
			}
			if update.Message != nil {
				command := update.Message.Command()
				arguments := update.Message.CommandArguments()
				if command == "" || arguments == "" {
					continue
				}

				args := strings.Split(arguments, " ")

				if command == "set_nsec" && len(args) == 2 {
					go p.HandleSetNsecCommand(ctx, &Command{
						UserID:  update.Message.From.ID,
						Command: command,
						Args:    args,
					})
				}
			}
			if update.MyChatMember != nil && update.MyChatMember.Chat.Type == "channel" {
				channelUpdate := &ChannelUpdate{
					Username:   strings.ToLower(update.MyChatMember.Chat.UserName),
					Title:      update.MyChatMember.Chat.Title,
					ID:         update.MyChatMember.Chat.ID,
					IsBotAdded: true,
				}

				if update.MyChatMember.NewChatMember.Status == "left" {
					channelUpdate.IsBotAdded = false
				}

				go p.HandleChannelUpdate(ctx, channelUpdate)
				continue
			}
		}
	}
}

func (p *TelegramPort) PublishMessage(ctx context.Context, post *Post) {
	nsec, err := p.store.GetNsecByChannelID(ctx, post.ChannelID)
	if err != nil {
		p.logger.Error(
			"failed to get nsec by channel id",
			zap.Error(err),
			zap.Int64("channel_id", post.ChannelID),
		)
		return
	}

	for _, f := range post.Files {
		url, err := p.uploader.UploadFile(ctx, f.Name, f.Reader)
		if err != nil {
			p.logger.Error("failed to upload file", zap.Error(err))
			continue
		}

		post.Text += "\n" + url
	}

	err = p.nostr.Publish(ctx, nsec, post.Text)
	if err != nil {
		p.logger.Error("failed to publish post", zap.Error(err), zap.Int64("channel_id", post.ChannelID))
		return
	}
}

func (p *TelegramPort) HandleChannelUpdate(ctx context.Context, channel *ChannelUpdate) {
	err := p.store.UpsertTelegramChannel(ctx, domain.TelegramChannel{
		TelegramID: channel.ID,
		Name:       channel.Title,
		Username:   channel.Username,
		IsBotAdded: channel.IsBotAdded,
	})

	if err != nil {
		p.logger.Error("failed to upsert telegram channel", zap.Error(err), zap.String("username", channel.Username))
		return
	}
}

func (p *TelegramPort) HandleSetNsecCommand(ctx context.Context, command *Command) {
	if command == nil {
		return
	}

	var err error
	defer func() {
		if err != nil {
			p.logger.Error("failed to handle set_nsec command", zap.Error(err), zap.Any("command", command))
		}
	}()

	if len(command.Args) != 2 {
		err = errors.New("invalid args")
		return
	}

	channelUsername := command.Args[0]

	channel, err := p.store.GetTelegramChannel(ctx, strings.ToLower(channelUsername))
	if err != nil || channel == nil {
		return
	}

	isOwner, err := p.IsUserChannelOwner(command.UserID, channel.TelegramID)
	if err != nil {
		return
	}

	if !isOwner {
		err = fmt.Errorf("user=%d is not channel owner", command.UserID)
		return
	}

	nsec := command.Args[1]

	_, sk, err := nip19.Decode(nsec)
	if err != nil {
		return
	}

	npub, err := nostr.GetPublicKey(sk.(string))
	if err != nil {
		return
	}

	p.store.SetNostrAccount(ctx, domain.NostrAccount{
		Npub:                npub,
		Nsec:                nsec,
		TelegramChannelID:   channel.TelegramID,
		CrossPostingEnabled: true,
	})

	p.logger.Info("nsec set", zap.String("username", channelUsername))

}

func (p *TelegramPort) IsUserChannelOwner(userID, channelID int64) (bool, error) {
	member, err := p.bot.GetChatMember(tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
			ChatID: channelID,
			UserID: userID,
		},
	})
	if err != nil {
		return false, err
	}

	if member.Status == "creator" {
		return true, nil
	}

	return false, nil
}
