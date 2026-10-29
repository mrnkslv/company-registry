package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"

	"company-registry/internal/infrastructure/outbox"
)

type Publisher struct {
	writer *kafka.Writer
}

func NewPublisher(brokers []string, topic string) *Publisher {
	return &Publisher{
		writer: &kafka.Writer{ //nolint:exhaustruct_v5 // defaults are fine for the rest
			Addr:                   kafka.TCP(brokers...),
			Topic:                  topic,
			Balancer:               &kafka.Hash{Hasher: nil}, // default hasher; partition by key = company id
			RequiredAcks:           kafka.RequireAll,
			AllowAutoTopicCreation: true,
			BatchTimeout:           10 * time.Millisecond,
		},
	}
}

func (p *Publisher) Publish(ctx context.Context, messages []outbox.Message) error {
	kafkaMessages := make([]kafka.Message, 0, len(messages))

	for _, m := range messages {
		headers := make([]kafka.Header, 0, len(m.Headers))
		for k, v := range m.Headers {
			headers = append(headers, kafka.Header{Key: k, Value: []byte(v)})
		}

		kafkaMessages = append(kafkaMessages, kafka.Message{ //nolint:exhaustruct_v5 // only the fields we set
			Key:     m.Key,
			Value:   m.Value,
			Headers: headers,
		})
	}

	if err := p.writer.WriteMessages(ctx, kafkaMessages...); err != nil {
		return fmt.Errorf("writer.WriteMessages: %w", err)
	}

	return nil
}

func (p *Publisher) Close() error {
	if err := p.writer.Close(); err != nil {
		return fmt.Errorf("writer.Close: %w", err)
	}

	return nil
}
