package pcrd_jp_repository

import (
	"context"
	"fmt"
	"github.com/SpeedxPz/pcrd-version-updater/src/entity/logger"
	"github.com/SpeedxPz/pcrd-version-updater/src/use_case"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"time"
)

type rest struct {
	client  *http.Client
	baseURL string
	locale  string
}

func (r rest) GetResourceVersion(ctx context.Context, startVersion string) (string, error) {
	ctx, span := tracer.Start(ctx, "pcrd_jp_repository.GetResourceVersion")
	defer span.End()

	version, err := strconv.ParseInt(startVersion, 10, 64)
	if err != nil {
		zap.L().Error("invalid start version", logger.WithTraceId(ctx), zap.Any("error", err), zap.Any("startVersion", startVersion))
		span.SetStatus(codes.Error, fmt.Sprintf("invalid start version %s: %s", startVersion, err))
		return "", fmt.Errorf("invalid start version %s: %w", startVersion, use_case.ErrRetrieveData)
	}

	var newVersion string
	for i := 1; i < 20; i++ {
		guessNumber := version + (int64(i) * int64(10))
		zap.L().Debug("guesing", logger.WithTraceId(ctx), zap.Any("guessNumber", guessNumber))
		result, err := r.call(ctx, guessNumber)
		if err != nil {
			zap.L().Error("guessing failed", logger.WithTraceId(ctx), zap.Any("error", err), zap.Any("version", guessNumber))
			continue
		}
		if result {
			newVersion = fmt.Sprintf("%d", guessNumber)
			zap.L().Debug("version accept!", logger.WithTraceId(ctx), zap.Any("guessNumber", guessNumber))
		}
		time.Sleep(1 * time.Second)
	}

	if len(newVersion) <= 0 {
		newVersion = startVersion
	}

	return newVersion, nil
}

func (r rest) call(ctx context.Context, version int64) (bool, error) {
	ctx, span := tracer.Start(ctx, fmt.Sprintf("pcrd_jp_repository.call(%d)", version))
	defer span.End()

	endpoint := fmt.Sprintf("%s/dl/Resources/%d/%s/AssetBundles/Android/manifest/manifest_assetmanifest", r.baseURL, version, r.locale)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		zap.L().Error("create request failed", logger.WithTraceId(ctx), zap.Any("error", err))
		span.SetStatus(codes.Error, fmt.Sprintf("create request failed: %s", err))
		return false, fmt.Errorf("create request failed: %w", use_case.ErrRetrieveData)
	}

	res, err := r.client.Do(req)
	if err != nil {
		zap.L().Error("request failed", logger.WithTraceId(ctx), zap.Any("error", err))
		span.SetStatus(codes.Error, fmt.Sprintf("request failed: %s", err))
		return false, fmt.Errorf("request failed: %w", use_case.ErrRetrieveData)
	}

	if res.StatusCode != http.StatusOK {
		return false, nil
	}

	return true, nil
}

func (r rest) HealthCheck(ctx context.Context) error {
	return nil
}

func NewRest(baseURL string) use_case.PcrdJPRepository {
	c := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			IdleConnTimeout:     30 * time.Second,
			DisableCompression:  true,
			MaxIdleConnsPerHost: 10,
		},
	}

	r := &rest{
		client:  c,
		baseURL: baseURL,
		locale:  "Jpn",
	}
	return r
}
