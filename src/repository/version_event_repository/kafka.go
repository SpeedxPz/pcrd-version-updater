package version_event_repository

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/SpeedxPz/pcrd-version-updater/src/entity/logger"
	"github.com/SpeedxPz/pcrd-version-updater/src/use_case"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
	"net"
	"time"
)

type kafkaMQVersionEvent struct {
	ID             string    `json:"id"`
	ServerCode     string    `json:"serverCode"`
	AppVersion     string    `json:"appVersion"`
	ResVersion     string    `json:"resVersion"`
	UpdateDateTime time.Time `json:"updatedAt"`
}

type kafkaMQ struct {
	client *kafka.Writer
}

func (k kafkaMQ) PublishVersion(ctx context.Context, version use_case.GameVersion) error {
	ctx, span := tracer.Start(ctx, "version_event_repository.PublishVersion")
	defer span.End()

	ver := kafkaMQVersionEvent{
		ID:             version.Setting.ID,
		ServerCode:     string(version.Setting.ServerCode),
		AppVersion:     version.AppVersion,
		ResVersion:     version.ResVersion,
		UpdateDateTime: time.Now(),
	}

	messageBytes, err := json.Marshal(ver)
	if err != nil {
		zap.L().Error("error while saving data", logger.WithTraceId(ctx), zap.Any("error", err), zap.Any("Ver", ver))
		span.SetStatus(codes.Error, fmt.Sprintf("error while saving data %+v: %s", ver, err))
		return fmt.Errorf("error while saving data: %w", use_case.ErrVersionPublish)
	}

	message := kafka.Message{
		Key:   []byte(fmt.Sprintf("%s", ver.ID)),
		Value: messageBytes,
	}

	err = k.client.WriteMessages(ctx, message)
	if err != nil {
		zap.L().Error("error while writing message", logger.WithTraceId(ctx), zap.Any("error", err))
		span.SetStatus(codes.Error, fmt.Sprintf("error while saving data: %s", err))
		return fmt.Errorf("error while saving data: %w", use_case.ErrVersionPublish)
	}

	return nil
}

func NewKafkaMQ(boostrapServer string, topic string) use_case.VersionEventRepository {
	var w kafka.Writer

	w = kafka.Writer{
		Addr:     kafka.TCP(boostrapServer),
		Topic:    topic,
		Balancer: &kafka.Hash{},
		Transport: &kafka.Transport{
			Dial: (&net.Dialer{
				Timeout: 10 * time.Second,
			}).DialContext,
		},
	}

	k := kafkaMQ{client: &w}

	return k
}
