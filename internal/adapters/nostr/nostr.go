package nostr

import (
	"context"
	"sync"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"go.uber.org/zap"
)

type NostrAdapter struct {
	// Lock is used to prevent concurrent map writes when reconnecting to relays
	Lock   *sync.RWMutex
	Relays map[string]*nostr.Relay
	Logger *zap.Logger
}

func NewNostrAdapter(ctx context.Context, logger *zap.Logger, relays []string) *NostrAdapter {
	a := &NostrAdapter{
		Lock:   &sync.RWMutex{},
		Logger: logger,
		Relays: make(map[string]*nostr.Relay, len(relays)),
	}

	for _, relay := range relays {
		connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		relayConnection, err := nostr.RelayConnect(connectCtx, relay)
		if err != nil {
			logger.Error("failed to connect to relay", zap.Error(err), zap.String("relay", relay))
			continue
		}

		a.Logger.Info("connected to relay", zap.String("relay", relay))

		a.Relays[relay] = relayConnection
	}

	return a
}

func (a *NostrAdapter) Publish(ctx context.Context, nsec string, text string) error {
	event := nostr.Event{
		Kind:      1,
		Content:   text,
		CreatedAt: nostr.Now(),
		Tags:      []nostr.Tag{},
	}

	_, sk, err := nip19.Decode(nsec)
	if err != nil {
		return err
	}

	err = event.Sign(sk.(string))
	if err != nil {
		return err
	}

	a.Lock.RLock()
	defer a.Lock.RUnlock()

	publishedTo := make([]string, 0, len(a.Relays))
	for _, relay := range a.Relays {
		publishCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		status, err := relay.Publish(publishCtx, event)
		if err != nil {
			a.Logger.Error("failed to publish event", zap.Error(err), zap.String("relay", relay.URL))
			go a.reconnect(ctx, relay.URL)
			continue
		}
		if status == nostr.PublishStatusSucceeded {
			publishedTo = append(publishedTo, relay.URL)
		}
	}
	a.Logger.Info("published event", zap.Strings("relay", publishedTo), zap.String("event_pubkey", event.PubKey))

	return nil
}

func (a *NostrAdapter) reconnect(ctx context.Context, relay string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			r := a.Relays[relay]
			if r != nil {
				r.Close()
			}

			relayConnection, err := nostr.RelayConnect(connectCtx, relay)
			if err != nil {
				a.Logger.Error("failed to reconnect to relay", zap.Error(err), zap.String("relay", relay))
				time.Sleep(10 * time.Second)
				continue
			}

			a.Logger.Info("reconnected to relay", zap.String("relay", relay))
			a.Lock.Lock()
			a.Relays[relay] = relayConnection
			a.Lock.Unlock()
			return
		}
	}
}
