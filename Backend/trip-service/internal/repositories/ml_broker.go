package repositories

import "context"

// handler nil → ack, error → nak.
type MLBroker interface {
	PublishMLTask(ctx context.Context, msg MLTaskMessage) error
	SubscribeMLResults(ctx context.Context, handler MLResultHandler) error
	Close() error
}

type MLResultHandler func(ctx context.Context, msg MLResultMessage) error
