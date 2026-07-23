package worker

import (
	"context"
	"log/slog"
	"strings"

	"pinz/backend/trip-service/internal/repositories"
)

// Перекладывает протухшие сообщения в ml.dlq.* и удаляет оригинал из WorkQueue.
func RunDLQRouter(ctx context.Context, broker *repositories.NATSBroker) error {
	if broker == nil {
		return nil
	}
	return broker.SubscribeMaxDeliveriesAdvisories(ctx, func(ctx context.Context, adv repositories.DLQAdvisory, payload []byte) {
		if adv.Subject == "" {
			slog.WarnContext(ctx, "dlq router: advisory without subject, skipping",
				"stream", adv.Stream, "seq", adv.StreamSeq)
			return
		}
		dlqSubject := "ml.dlq." + strings.TrimPrefix(adv.Subject, "ml.")
		if err := broker.PublishRaw(ctx, dlqSubject, payload); err != nil {
			slog.WarnContext(ctx, "dlq router: publish failed",
				"stream", adv.Stream, "consumer", adv.Consumer,
				"seq", adv.StreamSeq, "dlq_subject", dlqSubject, "error", err)
			return
		}
		if err := broker.DeleteStreamMsg(ctx, adv.Stream, adv.StreamSeq); err != nil {
			slog.WarnContext(ctx, "dlq router: delete original failed",
				"stream", adv.Stream, "seq", adv.StreamSeq, "error", err)
		}
		slog.InfoContext(ctx, "dlq router: message routed",
			"stream", adv.Stream, "consumer", adv.Consumer,
			"seq", adv.StreamSeq, "deliveries", adv.Deliveries,
			"dlq_subject", dlqSubject)
	})
}
