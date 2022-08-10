package version_repository

import (
	"context"
	"fmt"
	"github.com/SpeedxPz/pcrd-version-updater/src/entity/logger"
	"github.com/SpeedxPz/pcrd-version-updater/src/entity/setting"
	"github.com/SpeedxPz/pcrd-version-updater/src/use_case"
	"go.mongodb.org/mongo-driver/bson"
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

func (m mongoDB) GetByID(ctx context.Context, appId string) (use_case.GameVersion, error) {
	ctx, span := tracer.Start(ctx, "version_repository.GetByID")
	defer span.End()

	if len(appId) <= 0 {
		zap.L().Error("missing app id", logger.WithTraceId(ctx), zap.Any("error", use_case.ErrMissingAppID))
		span.SetStatus(codes.Error, fmt.Sprintf("missing app id: %s", use_case.ErrMissingAppID))
		return use_case.GameVersion{}, fmt.Errorf("%w", use_case.ErrMissingAppID)
	}

	var dbEntities []mongoDBVersion

	filter := bson.M{}
	filter["id"] = appId

	cur, err := m.col.Find(ctx, filter)
	if err != nil {
		zap.L().Error("error while reading retrieving", logger.WithTraceId(ctx), zap.Any("error", err))
		span.SetStatus(codes.Error, fmt.Sprintf("error while retrieving: %s", err))
		return use_case.GameVersion{}, fmt.Errorf("error while retrieving: %w", use_case.ErrRetrivingVersion)
	}
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var o mongoDBVersion
		err := cur.Decode(&o)
		if err != nil {
			zap.L().Error("error while reading retrieving", logger.WithTraceId(ctx), zap.Any("error", err))
			span.SetStatus(codes.Error, fmt.Sprintf("error while retrieving: %s", err))
			return use_case.GameVersion{}, fmt.Errorf("error while retrieving: %w", use_case.ErrRetrivingVersion)
		}
		dbEntities = append(dbEntities, o)
	}

	if err := cur.Err(); err != nil {
		zap.L().Error("error while reading retrieving", logger.WithTraceId(ctx), zap.Any("error", err))
		span.SetStatus(codes.Error, fmt.Sprintf("error while retrieving: %s", err))
		return use_case.GameVersion{}, fmt.Errorf("error while retrieving: %w", use_case.ErrRetrivingVersion)
	}

	if len(dbEntities) <= 0 {
		zap.L().Error("not exists", logger.WithTraceId(ctx), zap.Any("ID", appId), zap.Any("error", use_case.ErrVersionNotFound))
		span.SetStatus(codes.Error, fmt.Sprintf("%s: %s", appId, use_case.ErrVersionNotFound))
		return use_case.GameVersion{}, fmt.Errorf("%s: %w", appId, use_case.ErrVersionNotFound)
	}

	result, err := dbEntities[0].ToUseCaseGameVersion()
	if err != nil {
		zap.L().Error("error while reading retrieving", logger.WithTraceId(ctx), zap.Any("ID", appId), zap.Any("error", err))
		span.SetStatus(codes.Error, fmt.Sprintf("%s", err))
		return use_case.GameVersion{}, fmt.Errorf("%w", use_case.ErrRetrivingVersion)
	}

	return result, nil
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

func (m mongoDB) Update(ctx context.Context, version use_case.GameVersion) error {
	ctx, span := tracer.Start(ctx, "version_repository.Create")
	defer span.End()

	res, err := m.col.UpdateOne(ctx, bson.M{
		"id": version.Setting.ID,
	}, bson.M{
		"$set": bson.M{
			"appVersion": version.AppVersion,
			"resVersion": version.ResVersion,
			"updatedAt":  time.Now(),
		},
	})

	if err != nil {
		zap.L().Error("error while saving retrieving", logger.WithTraceId(ctx), zap.Any("version", version), zap.Any("error", err))
		span.SetStatus(codes.Error, fmt.Sprintf("%s", err))
		return fmt.Errorf("%w", use_case.ErrSavingVersion)
	}

	if res.MatchedCount == 0 {
		zap.L().Error("cannot update", logger.WithTraceId(ctx), zap.Any("version", version), zap.Any("error", err))
		span.SetStatus(codes.Error, fmt.Sprintf("cannot update %s, %s", version, err))
		return fmt.Errorf("update version: %s not found: %w", version, use_case.ErrVersionNotFound)
	}

	return nil

}

func (m mongoDB) HealthCheck(ctx context.Context) error {
	return m.col.Database().Client().Ping(ctx, readpref.Primary())
}

func NewMongoDb(db *mongo.Database) use_case.VersionRepository {
	m := &mongoDB{col: db.Collection("versions")}

	return m
}
