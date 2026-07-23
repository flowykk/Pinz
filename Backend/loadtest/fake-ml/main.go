// Fake ML worker для e2e-проверки NATS pipeline.
package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type mlMedia struct {
	MediaID string `json:"media_id"`
	IsNew   bool   `json:"is_new"`
}

type mlPin struct {
	PinID string    `json:"pin_id"`
	IsNew bool      `json:"is_new"`
	Media []mlMedia `json:"media"`
}

type mlTask struct {
	Flow      string  `json:"flow"`
	TripID    string  `json:"trip_id"`
	SessionID string  `json:"session_id,omitempty"`
	Pins      []mlPin `json:"pins"`
}

type mlSuggestion struct {
	PinID    string   `json:"pin_id"`
	Category string   `json:"category"`
	Tags     []string `json:"tags,omitempty"`
}

type mlResult struct {
	Flow           string         `json:"flow"`
	TripID         string         `json:"trip_id"`
	SessionID      string         `json:"session_id,omitempty"`
	SimilarGroups  [][]string     `json:"similar_groups,omitempty"`
	NSFWIDs        []string       `json:"nsfw_ids,omitempty"`
	PinSuggestions []mlSuggestion `json:"pin_suggestions,omitempty"`
}

func main() {
	url := flag.String("url", "nats://localhost:4223", "NATS URL")
	token := flag.String("token", "", "auth token, опционально")
	insecure := flag.Bool("insecure", false, "skip TLS certificate verification")
	failCount := flag.Int("fail-count", 0, "NAK первые N сообщений (для retry/DLQ)")
	category := flag.String("category", "food", "category для is_new пинов")
	tags := flag.String("tag", "tasty,test", "теги через запятую")
	flag.Parse()

	opts := []nats.Option{nats.MaxReconnects(-1)}
	if *token != "" {
		opts = append(opts, nats.Token(*token))
	}
	if *insecure {
		opts = append(opts, nats.Secure(&tls.Config{InsecureSkipVerify: true}))
	}
	nc, err := nats.Connect(*url, opts...)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer nc.Drain()
	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatalf("jetstream: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cons, err := js.CreateOrUpdateConsumer(ctx, "ML_TASKS", jetstream.ConsumerConfig{
		Durable:    "ml-workers",
		AckPolicy:  jetstream.AckExplicitPolicy,
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	})
	if err != nil {
		log.Fatalf("consumer: %v", err)
	}
	fmt.Println("fake-ml: subscribed ML_TASKS / ml-workers")
	if *failCount > 0 {
		fmt.Printf("fake-ml: will NAK first %d messages\n", *failCount)
	}

	var processed atomic.Int64
	_, err = cons.Consume(func(m jetstream.Msg) {
		seq := processed.Add(1)
		meta, _ := m.Metadata()
		fmt.Printf("fake-ml: got msg seq=%d subject=%s delivered=%d\n",
			seq, m.Subject(), meta.NumDelivered)

		if int64(*failCount) >= seq {
			fmt.Printf("fake-ml: NAK seq=%d (simulated failure)\n", seq)
			_ = m.NakWithDelay(2 * time.Second)
			return
		}

		var task mlTask
		if err := json.Unmarshal(m.Data(), &task); err != nil {
			fmt.Printf("fake-ml: malformed task, ack-drop: %v\n", err)
			_ = m.Ack()
			return
		}

		result := mlResult{
			Flow:      task.Flow,
			TripID:    task.TripID,
			SessionID: task.SessionID,
		}
		for _, p := range task.Pins {
			if !p.IsNew {
				continue
			}
			result.PinSuggestions = append(result.PinSuggestions, mlSuggestion{
				PinID:    p.PinID,
				Category: *category,
				Tags:     strings.Split(*tags, ","),
			})
		}

		respSubject := "ml.results." + task.Flow
		payload, _ := json.Marshal(result)
		if _, err := js.Publish(ctx, respSubject, payload); err != nil {
			fmt.Printf("fake-ml: publish result failed: %v\n", err)
			_ = m.NakWithDelay(2 * time.Second)
			return
		}
		_ = m.Ack()
		fmt.Printf("fake-ml: published result to %s (trip=%s, suggestions=%d)\n",
			respSubject, task.TripID, len(result.PinSuggestions))
	})
	if err != nil {
		log.Fatalf("consume: %v", err)
	}

	<-ctx.Done()
	fmt.Println("fake-ml: shutdown")
}
