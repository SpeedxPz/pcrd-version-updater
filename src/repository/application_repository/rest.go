package application_repository

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/SpeedxPz/pcrd-version-updater/src/entity/application"
	"github.com/SpeedxPz/pcrd-version-updater/src/entity/logger"
	"github.com/SpeedxPz/pcrd-version-updater/src/use_case"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"time"
)

type rest struct {
	client  *http.Client
	baseURL string
}

type restApplicationResp struct {
	Results []restApplication `json:"results"`
	Total   int64             `json:"total"`
}

type restApplication struct {
	AppID          string `json:"app_id"`
	BundleID       string `json:"bundle_id"`
	Name           string `json:"name"`
	Version        string `json:"version"`
	Author         string `json:"author"`
	Icon           string `json:"icon"`
	Platform       string `json:"platform"`
	UpdateDateTime string `json:"update_datetime"`
}

func (r restApplicationResp) ToEntities() ([]application.Application, error) {

	results := make([]application.Application, len(r.Results))
	for i, _ := range r.Results {
		result, err := r.Results[i].ToEntity()
		if err != nil {
			return nil, err
		}
		results[i] = result
	}

	return results, nil
}

func (r restApplication) ToEntity() (application.Application, error) {

	return application.Application{
		AppID:    r.AppID,
		BundleID: r.BundleID,
		Name:     r.Name,
		Version:  r.Version,
		Author:   r.Author,
		Icon:     r.Icon,
	}, nil
}

func (r rest) GetAndroidAppByID(ctx context.Context, appID string) (application.Application, error) {
	ctx, span := tracer.Start(ctx, "application_repository.GetAndroidAppByID")
	defer span.End()

	if len(appID) <= 0 {
		zap.L().Error("missing app id", logger.WithTraceId(ctx), zap.Any("error", use_case.ErrMissingAppID))
		span.SetStatus(codes.Error, fmt.Sprintf("missing app id: %s", use_case.ErrMissingAppID))
		return application.Application{}, fmt.Errorf("%w", use_case.ErrMissingAppID)
	}

	endpoint := fmt.Sprintf("%s/app?platform=android&bundle_id=%s", r.baseURL, appID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		zap.L().Error("create request failed", logger.WithTraceId(ctx), zap.Any("error", err))
		span.SetStatus(codes.Error, fmt.Sprintf("create request failed: %s", err))
		return application.Application{}, fmt.Errorf("%w", use_case.ErrRetrivingApplication)
	}

	res, err := r.client.Do(req)
	if err != nil {
		zap.L().Error("execute request failed", logger.WithTraceId(ctx), zap.Any("error", err))
		span.SetStatus(codes.Error, fmt.Sprintf("execute request failed: %s", err))
		return application.Application{}, fmt.Errorf("%w", use_case.ErrRetrivingApplication)
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		zap.L().Error("io read failed", logger.WithTraceId(ctx), zap.Any("error", err))
		span.SetStatus(codes.Error, fmt.Sprintf("io read failed: %s", err))
		return application.Application{}, fmt.Errorf("%w", use_case.ErrRetrivingApplication)
	}
	defer res.Body.Close()

	var o restApplicationResp
	err = json.Unmarshal([]byte(data), &o)
	if err != nil {
		zap.L().Error("unmarshal failed", logger.WithTraceId(ctx), zap.Any("error", err))
		span.SetStatus(codes.Error, fmt.Sprintf("unmarshal failed: %s", err))
		return application.Application{}, fmt.Errorf("%w", use_case.ErrRetrivingApplication)
	}

	result, err := o.ToEntities()
	if err != nil {
		zap.L().Error("convert to entities failed", logger.WithTraceId(ctx), zap.Any("error", err))
		span.SetStatus(codes.Error, fmt.Sprintf("convert to entities failed: %s", err))
		return application.Application{}, fmt.Errorf("%w", use_case.ErrRetrivingApplication)
	}

	if len(result) <= 0 {
		zap.L().Error("app not found", logger.WithTraceId(ctx), zap.Any("AppID", appID), zap.Any("error", use_case.ErrApplicationNotFound))
		span.SetStatus(codes.Error, fmt.Sprintf("app %s: %s", appID, use_case.ErrApplicationNotFound))
		return application.Application{}, fmt.Errorf("%s: %w", appID, use_case.ErrApplicationNotFound)
	}

	return result[0], nil
}

func (r rest) HealthCheck(ctx context.Context) error {
	return nil
}

func NewRest(baseURL string) use_case.ApplicationRepository {
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
	}
	return r
}
