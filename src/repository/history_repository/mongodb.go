package history_repository

import (
	"context"
	"fmt"
	"github.com/SpeedxPz/pcrd-version-updater/src/entity/logger"
	"github.com/SpeedxPz/pcrd-version-updater/src/entity/setting"
	"github.com/SpeedxPz/pcrd-version-updater/src/use_case"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
	"time"
)

type mongoDB struct {
	col *mongo.Collection
}

type mongoDBVersion struct {
	ID             string    `bson:"id"`
	ServerCode     string    `bson:"serverCode"`
	AppVersion     string    `bson:"appVersion"`
	ResVersion     string    `bson:"resVersion"`
	CreateDateTime time.Time `bson:"createdAt"`
	UpdateDateTime time.Time `bson:"updatedAt"`
}

func newMongoDBVersion(version use_case.GameVersion) mongoDBVersion {
	return mongoDBVersion{
		ID:         version.Setting.ID,
		ServerCode: string(version.Setting.ServerCode),
		AppVersion: version.AppVersion,
		ResVersion: version.ResVersion,
	}
}

func (m mongoDBVersion) ToUseCaseGameVersion() (use_case.GameVersion, error) {

	serverCode, err := setting.ParseServerCode(m.ServerCode)
	if err != nil {
		return use_case.GameVersion{}, err
	}

	return use_case.GameVersion{
		Setting: setting.Setting{
			ID:         m.ID,
			ServerCode: serverCode,
		},
		AppVersion: m.AppVersion,
		ResVersion: m.ResVersion,
	}, nil
}

func (m mongoDB) Create(ctx context.Context, version use_case.GameVersion) error {
	ctx, span := tracer.Start(ctx, "version_repository.Create")
	defer span.End()

	doc := newMongoDBVersion(version)
	doc.CreateDateTime = time.Now()
	doc.UpdateDateTime = time.Now()
	_, err := m.col.InsertOne(ctx, doc)

	if err != nil {
		zap.L().Error("error while reading retrieving", logger.WithTraceId(ctx), zap.Any("version", version), zap.Any("error", err))
		span.SetStatus(codes.Error, fmt.Sprintf("%s", err))
		return fmt.Errorf("%w", use_case.ErrSavingVersion)
	}

	return nil
}

func (m mongoDB) HealthCheck(ctx context.Context) error {
	return m.col.Database().Client().Ping(ctx, readpref.Primary())
}

func NewMongoDb(db *mongo.Database) use_case.HistoryRepository {
	m := &mongoDB{col: db.Collection("histories")}

	return m
}
