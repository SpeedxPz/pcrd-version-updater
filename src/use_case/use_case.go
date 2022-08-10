package use_case

import (
	"context"
	"errors"
	"github.com/SpeedxPz/pcrd-version-updater/src/entity/application"
	"github.com/SpeedxPz/pcrd-version-updater/src/entity/credential"
	"github.com/SpeedxPz/pcrd-version-updater/src/entity/setting"
	"go.opentelemetry.io/otel"
)

var (
	ErrPermissionDenied     = errors.New("permission denied")
	ErrDataTransform        = errors.New("data transformation error")
	ErrInvalidRequestParam  = errors.New("invalid request parameter")
	ErrResVerNotAvailable   = errors.New("resource version not available from remote")
	ErrRetrieveData         = errors.New("data retrieve failed")
	ErrRetrivingSetting     = errors.New("failed to retrieving setting data")
	ErrSettingNotExists     = errors.New("setting not found")
	ErrMissingAppID         = errors.New("app id is required")
	ErrRetrivingApplication = errors.New("failed to retrieving application data")
	ErrApplicationNotFound  = errors.New("application not found")
	ErrRetrivingVersion     = errors.New("failed to retrieving version data")
	ErrVersionNotFound      = errors.New("version not found")
	ErrSavingVersion        = errors.New("failed to save version")
	ErrVersionPublish       = errors.New("cannot publish version")
	ErrSavingSetting        = errors.New("failed to save setting")
)

var tracer = otel.Tracer("use_case")

type UseCase struct {
	applicationRepository  ApplicationRepository
	settingRepository      SettingRepository
	pcrdTHRepository       PcrdTHRepository
	pcrdJPRepository       PcrdJPRepository
	versionRepository      VersionRepository
	historyRepository      HistoryRepository
	versionEventRepository VersionEventRepository
}

type ApplicationRepository interface {
	HealthCheck(ctx context.Context) error
	GetAndroidAppByID(ctx context.Context, appID string) (application.Application, error)
}

type SettingRepository interface {
	HealthCheck(ctx context.Context) error
	GetSettingByID(ctx context.Context, ID string) (PCRDSetting, error)
}

type VersionRepository interface {
	HealthCheck(ctx context.Context) error
	GetByID(ctx context.Context, appId string) (GameVersion, error)
	Create(ctx context.Context, version GameVersion) error
	Update(ctx context.Context, version GameVersion) error
}

type HistoryRepository interface {
	HealthCheck(ctx context.Context) error
	Create(ctx context.Context, version GameVersion) error
}

type PcrdTHRepository interface {
	GetResourceVersion(ctx context.Context, c credential.Credential, v PcrdVersion) (string, error)
	HealthCheck(ctx context.Context) error
}

type PcrdJPRepository interface {
	GetResourceVersion(ctx context.Context, startVersion string) (string, error)
	HealthCheck(ctx context.Context) error
}

type VersionEventRepository interface {
	PublishVersion(ctx context.Context, version GameVersion) error
}

type PcrdVersion struct {
	AppVersion string
	ResVersion string
}

type GameVersion struct {
	Setting    setting.Setting
	AppVersion string
	ResVersion string
}

func New(
	appRepo ApplicationRepository,
	settingRepo SettingRepository,
	pcrdTHRepo PcrdTHRepository,
	pcrdJPRepo PcrdJPRepository,
	versionRepo VersionRepository,
	historyRepo HistoryRepository,
	versionEventRepo VersionEventRepository,
) *UseCase {
	return &UseCase{
		settingRepository:      settingRepo,
		applicationRepository:  appRepo,
		pcrdTHRepository:       pcrdTHRepo,
		pcrdJPRepository:       pcrdJPRepo,
		versionRepository:      versionRepo,
		historyRepository:      historyRepo,
		versionEventRepository: versionEventRepo,
	}
}
