package consumer

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/example/adnova/internal/config"
	ingestionservice "github.com/example/adnova/internal/ingestion/service"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/plain"
	"go.uber.org/zap"
)

type Processor interface {
	ProcessKafkaMessage(context.Context, ingestionservice.KafkaMessage) error
}

type Consumer struct {
	client    *kgo.Client
	config    config.KafkaConfig
	processor Processor
	logger    *zap.Logger
}

type DLQMessage struct {
	SourceTopic     string `json:"source_topic"`
	SourcePartition int32  `json:"source_partition"`
	SourceOffset    int64  `json:"source_offset"`
	Key             string `json:"key,omitempty"`
	PayloadHash     string `json:"payload_hash"`
	PayloadBase64   string `json:"payload_base64"`
	ErrorCode       string `json:"error_code"`
	ErrorMessage    string `json:"error_message"`
	FailedAt        string `json:"failed_at"`
}

func New(cfg config.KafkaConfig, processor Processor, logger *zap.Logger) (*Consumer, error) {
	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...), kgo.ConsumerGroup(cfg.GroupID), kgo.ConsumeTopics(cfg.Topics...),
		kgo.DisableAutoCommit(), kgo.FetchMaxBytes(10 << 20),
	}
	if cfg.TLSEnabled {
		opts = append(opts, kgo.DialTLSConfig(&tls.Config{MinVersion: tls.VersionTLS12}))
	}
	if cfg.Username != "" {
		opts = append(opts, kgo.SASL(plain.Auth{User: cfg.Username, Pass: cfg.Password}.AsMechanism()))
	}
	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("create Kafka client: %w", err)
	}
	return &Consumer{client: client, config: cfg, processor: processor, logger: logger}, nil
}

func (c *Consumer) Close() { c.client.Close() }

func (c *Consumer) Ping(ctx context.Context) error { return c.client.Ping(ctx) }

func (c *Consumer) Run(ctx context.Context) error {
	for {
		fetches := c.client.PollRecords(ctx, 1)
		if ctx.Err() != nil {
			return nil
		}
		if errs := fetches.Errors(); len(errs) > 0 {
			return fmt.Errorf("poll Kafka: %w", errs[0].Err)
		}
		for _, record := range fetches.Records() {
			if err := c.processRecord(ctx, record); err != nil {
				return err
			}
			if err := c.client.CommitRecords(ctx, record); err != nil {
				return fmt.Errorf("commit Kafka offset: %w", err)
			}
		}
	}
}

func (c *Consumer) processRecord(ctx context.Context, record *kgo.Record) error {
	input := ingestionservice.KafkaMessage{
		ExpectedTenantID: c.config.TenantID, ExpectedProducer: c.config.ProducerSystem,
		Topic: record.Topic, Partition: record.Partition, Offset: record.Offset, Payload: record.Value,
		Debounce: c.config.Debounce, Lease: c.config.MessageLease,
	}
	var lastErr error
	for attempt := 0; attempt <= c.config.ProcessRetries; attempt++ {
		lastErr = c.processor.ProcessKafkaMessage(ctx, input)
		if lastErr == nil {
			return nil
		}
		var permanent *ingestionservice.PermanentError
		if errors.As(lastErr, &permanent) {
			if err := c.publishDLQ(ctx, record, permanent); err != nil {
				return err
			}
			c.logger.Warn("Kafka ingestion message moved to DLQ", zap.String("topic", record.Topic), zap.Int32("partition", record.Partition), zap.Int64("offset", record.Offset), zap.String("error_code", permanent.Code))
			return nil
		}
		if attempt < c.config.ProcessRetries {
			delay := time.Duration(1<<attempt) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}
	return fmt.Errorf("Kafka message processing retries exhausted: %w", lastErr)
}

func (c *Consumer) publishDLQ(ctx context.Context, source *kgo.Record, failure *ingestionservice.PermanentError) error {
	digest := sha256.Sum256(source.Value)
	payload, _ := json.Marshal(DLQMessage{
		SourceTopic: source.Topic, SourcePartition: source.Partition, SourceOffset: source.Offset,
		Key: string(source.Key), PayloadHash: hex.EncodeToString(digest[:]), ErrorCode: failure.Code,
		PayloadBase64: base64.StdEncoding.EncodeToString(source.Value), ErrorMessage: failure.Err.Error(), FailedAt: time.Now().UTC().Format(time.RFC3339Nano),
	})
	record := &kgo.Record{Topic: c.config.DLQTopic, Key: source.Key, Value: payload}
	if err := c.client.ProduceSync(ctx, record).FirstErr(); err != nil {
		return fmt.Errorf("publish Kafka DLQ: %w", err)
	}
	return nil
}
