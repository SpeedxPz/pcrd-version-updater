package setting_repository

import (
	"context"
	"fmt"
	"github.com/SpeedxPz/pcrd-version-updater/src/entity/credential"
	"github.com/SpeedxPz/pcrd-version-updater/src/entity/logger"
	"github.com/SpeedxPz/pcrd-version-updater/src/entity/setting"
	"github.com/SpeedxPz/pcrd-version-updater/src/use_case"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

type mongoDB struct {
	col *mongo.Collection
}

type mongoDBSetting struct {
	ID          string             `bson:"id"`
	ServerCode  string             `bson:"serverCode"`
	Credential  mongoDBCredential  `bson:"credential"`
	GuessConfig mongoDBGuessConfig `bson:"guess"`
}

type mongoDBCredential struct {
	UDID      string `bson:"udid"`
	ShortUDID int32  `bson:"shortUdid"`
	ViewerID  int32  `bson:"viewerId"`
}

type mongoDBGuessConfig struct {
	StartVersion string `bson:"startVersion"`
}

func (m mongoDBSetting) toUsecasePCRDSetting() (use_case.PCRDSetting, error) {

	serverCode, err := setting.ParseServerCode(m.ServerCode)
	if err != nil {
		return use_case.PCRDSetting{}, err
	}

	return use_case.PCRDSetting{
		Setting: setting.Setting{
			ID:         m.ID,
			ServerCode: serverCode,
		},
		Credential: credential.Credential{
			Udid:      m.Credential.UDID,
			ShortUdid: m.Credential.ShortUDID,
			ViewerID:  m.Credential.ViewerID,
			SessionID: "",
		},
		GuessStartVersion: m.GuessConfig.StartVersion,
	}, nil
}

func (m mongoDB) GetSettingByID(ctx context.Context, ID string) (use_case.PCRDSetting, error) {
	ctx, span := tracer.Start(ctx, "setting_repository.GetSettingByID")
	defer span.End()

	var dbEntities []mongoDBSetting

	filter := bson.M{}
	filter["id"] = ID

	cur, err := m.col.Find(ctx, filter)
	if err != nil {
		zap.L().Error("error while reading retrieving", logger.WithTraceId(ctx), zap.Any("error", err))
		span.SetStatus(codes.Error, fmt.Sprintf("error while retrieving: %s", err))
		return use_case.PCRDSetting{}, fmt.Errorf("error while retrieving: %w", use_case.ErrRetrivingSetting)
	}
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var o mongoDBSetting
		err := cur.Decode(&o)
		if err != nil {
			zap.L().Error("error while reading retrieving", logger.WithTraceId(ctx), zap.Any("error", err))
			span.SetStatus(codes.Error, fmt.Sprintf("error while retrieving: %s", err))
			return use_case.PCRDSetting{}, fmt.Errorf("error while retrieving: %w", use_case.ErrRetrivingSetting)
		}
		dbEntities = append(dbEntities, o)
	}

	if err := cur.Err(); err != nil {
		zap.L().Error("error while reading retrieving", logger.WithTraceId(ctx), zap.Any("error", err))
		span.SetStatus(codes.Error, fmt.Sprintf("error while retrieving: %s", err))
		return use_case.PCRDSetting{}, fmt.Errorf("error while retrieving: %w", use_case.ErrRetrivingSetting)
	}

	if len(dbEntities) <= 0 {
		zap.L().Error("not exists", logger.WithTraceId(ctx), zap.Any("ID", ID), zap.Any("error", use_case.ErrSettingNotExists))
		span.SetStatus(codes.Error, fmt.Sprintf("%s: %s", ID, use_case.ErrSettingNotExists))
		return use_case.PCRDSetting{}, fmt.Errorf("%s: %w", ID, use_case.ErrSettingNotExists)
	}

	result, err := dbEntities[0].toUsecasePCRDSetting()
	if err != nil {
		zap.L().Error("error while reading retrieving", logger.WithTraceId(ctx), zap.Any("ID", ID), zap.Any("error", err))
		span.SetStatus(codes.Error, fmt.Sprintf("%s", err))
		return use_case.PCRDSetting{}, fmt.Errorf("%w", use_case.ErrRetrivingSetting)
	}

	return result, nil
}

func (m mongoDB) HealthCheck(ctx context.Context) error {
	return m.col.Database().Client().Ping(ctx, readpref.Primary())
}

func NewMongoDb(db *mongo.Database) use_case.SettingRepository {
	m := &mongoDB{col: db.Collection("settings")}

	return m
}
