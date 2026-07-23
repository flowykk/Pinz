package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Пустой NATS_URL → (nil, nil): сервис стартует в stub-режиме без публикации ML.
func InitNATSBroker() (*NATSBroker, error) {
	url := os.Getenv("NATS_URL")
	if url == "" {
		return nil, nil
	}
	cfg := DefaultNATSBrokerConfig()
	cfg.URL = url
	cfg.CredsPath = os.Getenv("NATS_CREDS_PATH")
	cfg.Token = os.Getenv("NATS_TOKEN")
	return NewNATSBroker(cfg)
}

type NATSBrokerConfig struct {
	URL        string
	CredsPath  string // JWT-creds, опционально
	Token      string // bearer-token, опционально (используется если CredsPath пустой)
	AckWait    time.Duration
	MaxDeliver int
}

func DefaultNATSBrokerConfig() NATSBrokerConfig {
	return NATSBrokerConfig{
		URL:        "nats://nats:4222",
		AckWait:    10 * time.Minute,
		MaxDeliver: 5,
	}
}

type NATSBroker struct {
	cfg NATSBrokerConfig
	nc  *nats.Conn
	js  jetstream.JetStream
}

func NewNATSBroker(cfg NATSBrokerConfig) (*NATSBroker, error) {
	if cfg.URL == "" {
		return nil, errors.New("nats broker: URL is required")
	}
	if cfg.AckWait == 0 {
		cfg.AckWait = 10 * time.Minute
	}
	if cfg.MaxDeliver == 0 {
		cfg.MaxDeliver = 5
	}

	opts := []nats.Option{
		nats.Name("trip-service"),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(-1),
	}
	if cfg.CredsPath != "" {
		opts = append(opts, nats.UserCredentials(cfg.CredsPath))
	} else if cfg.Token != "" {
		opts = append(opts, nats.Token(cfg.Token))
	}

	nc, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("nats broker: connect: %w", err)
	}
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("nats broker: jetstream: %w", err)
	}
	b := &NATSBroker{cfg: cfg, nc: nc, js: js}
	if err := b.ensureStreams(context.Background()); err != nil {
		nc.Close()
		return nil, fmt.Errorf("nats broker: ensure streams: %w", err)
	}
	return b, nil
}

// trip-service владеет конфигурацией стримов сам — Helm-job не нужен.
func (b *NATSBroker) ensureStreams(ctx context.Context) error {
	streams := []jetstream.StreamConfig{
		{
			Name:       MLStreamTasks,
			Subjects:   []string{"ml.tasks.>"},
			Retention:  jetstream.WorkQueuePolicy,
			Storage:    jetstream.FileStorage,
			MaxAge:     24 * time.Hour,
			Discard:    jetstream.DiscardOld,
			Duplicates: 2 * time.Minute,
		},
		{
			// Limits (не WorkQueue): результаты читает несколько потребителей
			// (trip-service + e2e/debug), workqueue допускает лишь одного.
			Name:       MLStreamResults,
			Subjects:   []string{"ml.results.>"},
			Retention:  jetstream.LimitsPolicy,
			Storage:    jetstream.FileStorage,
			MaxAge:     24 * time.Hour,
			Discard:    jetstream.DiscardOld,
			Duplicates: 2 * time.Minute,
		},
		{
			Name:      MLStreamTasksDLQ,
			Subjects:  []string{"ml.dlq.>"},
			Retention: jetstream.LimitsPolicy,
			Storage:   jetstream.FileStorage,
			MaxAge:    7 * 24 * time.Hour,
			Discard:   jetstream.DiscardOld,
		},
	}
	for _, cfg := range streams {
		if _, err := b.js.CreateOrUpdateStream(ctx, cfg); err != nil {
			return fmt.Errorf("ensure stream %q: %w", cfg.Name, err)
		}
	}
	return nil
}

func (b *NATSBroker) PublishMLTask(ctx context.Context, msg MLTaskMessage) error {
	subject := SubjectForFlow(msg.Flow)
	if subject == "" {
		return fmt.Errorf("nats broker: unknown flow %q", msg.Flow)
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("nats broker: marshal: %w", err)
	}
	if _, err := b.js.Publish(ctx, subject, data); err != nil {
		return fmt.Errorf("nats broker: publish %s: %w", subject, err)
	}
	return nil
}

func (b *NATSBroker) PublishMLTextTask(ctx context.Context, msg MLTextTaskMessage) error {
	if msg.Flow == "" {
		msg.Flow = MLFlowTextModeration
	}
	subject := SubjectForFlow(msg.Flow)
	if subject == "" {
		return fmt.Errorf("nats broker: unknown flow %q", msg.Flow)
	}
	if len(msg.Items) == 0 {
		return nil
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("nats broker: marshal: %w", err)
	}
	if _, err := b.js.Publish(ctx, subject, data); err != nil {
		return fmt.Errorf("nats broker: publish %s: %w", subject, err)
	}
	return nil
}

func (b *NATSBroker) SubscribeMLResults(ctx context.Context, handler MLResultHandler) error {
	if handler == nil {
		return errors.New("nats broker: handler is required")
	}

	cons, err := b.js.CreateOrUpdateConsumer(ctx, MLStreamResults, jetstream.ConsumerConfig{
		Durable:    MLConsumerTripResults,
		AckPolicy:  jetstream.AckExplicitPolicy,
		AckWait:    b.cfg.AckWait,
		MaxDeliver: b.cfg.MaxDeliver,
	})
	if err != nil {
		return fmt.Errorf("nats broker: create consumer: %w", err)
	}

	_, err = cons.Consume(func(m jetstream.Msg) {
		var msg MLResultMessage
		if err := json.Unmarshal(m.Data(), &msg); err != nil {
			slog.WarnContext(ctx, "nats broker: malformed ml result, ack-drop", "error", err, "subject", m.Subject())
			_ = m.Ack()
			return
		}
		if err := handler(ctx, msg); err != nil {
			slog.WarnContext(ctx, "nats broker: ml result handler failed, will retry", "error", err, "trip_id", msg.TripID)
			_ = m.NakWithDelay(10 * time.Second)
			return
		}
		_ = m.Ack()
	})
	if err != nil {
		return fmt.Errorf("nats broker: consume: %w", err)
	}
	return nil
}

func (b *NATSBroker) Close() error {
	if b.nc == nil {
		return nil
	}
	return b.nc.Drain()
}

type DLQAdvisory struct {
	Stream     string
	Consumer   string
	StreamSeq  uint64
	Deliveries int
	Subject    string
}

type DLQAdvisoryHandler func(ctx context.Context, adv DLQAdvisory, payload []byte)

func (b *NATSBroker) SubscribeMaxDeliveriesAdvisories(ctx context.Context, handler DLQAdvisoryHandler) error {
	if handler == nil {
		return errors.New("nats broker: handler is required")
	}
	sub, err := b.nc.Subscribe("$JS.EVENT.ADVISORY.CONSUMER.MAX_DELIVERIES.>", func(m *nats.Msg) {
		var raw struct {
			Stream     string `json:"stream"`
			Consumer   string `json:"consumer"`
			StreamSeq  uint64 `json:"stream_seq"`
			Deliveries int    `json:"deliveries"`
		}
		if err := json.Unmarshal(m.Data, &raw); err != nil {
			slog.WarnContext(ctx, "nats broker: malformed dlq advisory", "error", err)
			return
		}
		stream, err := b.js.Stream(ctx, raw.Stream)
		if err != nil {
			slog.WarnContext(ctx, "nats broker: dlq stream lookup failed", "stream", raw.Stream, "error", err)
			return
		}
		orig, err := stream.GetMsg(ctx, raw.StreamSeq)
		if err != nil {
			slog.WarnContext(ctx, "nats broker: dlq get msg failed", "stream", raw.Stream, "seq", raw.StreamSeq, "error", err)
			return
		}
		handler(ctx, DLQAdvisory{
			Stream:     raw.Stream,
			Consumer:   raw.Consumer,
			StreamSeq:  raw.StreamSeq,
			Deliveries: raw.Deliveries,
			Subject:    orig.Subject,
		}, orig.Data)
	})
	if err != nil {
		return fmt.Errorf("nats broker: subscribe advisories: %w", err)
	}
	go func() {
		<-ctx.Done()
		_ = sub.Unsubscribe()
	}()
	return nil
}

func (b *NATSBroker) PublishRaw(ctx context.Context, subject string, data []byte) error {
	if _, err := b.js.Publish(ctx, subject, data); err != nil {
		return fmt.Errorf("nats broker: publish %s: %w", subject, err)
	}
	return nil
}

// Нужен после переноса в DLQ, иначе оригинал не уйдёт из WorkQueue-стрима.
func (b *NATSBroker) DeleteStreamMsg(ctx context.Context, streamName string, seq uint64) error {
	stream, err := b.js.Stream(ctx, streamName)
	if err != nil {
		return fmt.Errorf("nats broker: stream lookup: %w", err)
	}
	if err := stream.DeleteMsg(ctx, seq); err != nil {
		return fmt.Errorf("nats broker: delete msg %d: %w", seq, err)
	}
	return nil
}
