package main

import (
	"context"
	"github.com/SpeedxPz/pcrd-version-updater/src/repository/application_repository"
	"github.com/SpeedxPz/pcrd-version-updater/src/repository/history_repository"
	"github.com/SpeedxPz/pcrd-version-updater/src/repository/pcrd_jp_repository"
	"github.com/SpeedxPz/pcrd-version-updater/src/repository/pcrd_th_repository"
	"github.com/SpeedxPz/pcrd-version-updater/src/repository/setting_repository"
	"github.com/SpeedxPz/pcrd-version-updater/src/repository/version_event_repository"
	"github.com/SpeedxPz/pcrd-version-updater/src/repository/version_repository"
	"github.com/SpeedxPz/pcrd-version-updater/src/use_case"
	"github.com/caarlos0/env/v6"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"time"
)

type config struct {
	AppName             string `env:"APP_NAME" envDefault:"pcrd-version-updater"`
	AppVersion          string `env:"APP_VERSION"`
	Environment         string `env:"ENVIRONMENT" envDefault:"development"`
	Port                uint   `env:"PORT" envDefault:"8081"`
	Debuglog            bool   `env:"DEBUG_LOG" envDefault:"true"`
	JaegerEndpoint      string `env:"JAEGER_ENDPOINT" envDefault:"http://localhost:14268/api/traces"`
	MongoDbUri          string `env:"MONGO_DB_URI" envDefault:"mongodb://localhost:27017"`
	MongoDbStoreVersion string `env:"MONGO_DB_PCRD_VERSION" envDefault:"develop-store-version"`
	TargetAppId         string `env:"TARGET_APPID"`
	Service             struct {
		Application string `env:"SERVICE_APPLICATION_BASEURL"`
	}
	PCRD struct {
		JPEndpoint string `env:"PCRD_JP_ENDPOINT" envDefault:"http://prd-priconne-redive.akamaized.net"`
		JPSalt     string `env:"PCRD_JP_SALT" envDefault:""`
		THEndpoint string `env:"PCRD_TH_ENDPOINT" envDefault:"https://pcc-game.i3play.com"`
		THSalt     string `env:"PCRD_TH_SALT" envDefault:""`
	}
	KafkaServer            string `env:"KAFKA_SERVER" envDefault:"localhost:9092"`
	KafkaTopicVersionEvent string `env:"KAFKA_TOPIC_VERSION_EVENT"`
}

func main() {
	cfg := initEnvironment()
	initLogger(cfg)
	initTracer(cfg)
	appRepo,
		settingRepo,
		pcrdTHRepo,
		pcrdJPRepo,
		versionRepo,
		historyRepo,
		versionEventRepo := initRepositories(cfg)
	useCase := use_case.New(appRepo, settingRepo, pcrdTHRepo, pcrdJPRepo, versionRepo, historyRepo, versionEventRepo)

	ctx := context.Background()
	err := useCase.UpdateResourceVersion(ctx, cfg.TargetAppId)
	tp := otel.GetTracerProvider()
	tp.(*trace.TracerProvider).ForceFlush(ctx)
	if err != nil {
		panic(err)
	}
}

func initEnvironment() config {
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))

	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file: %s\n", err)
	}

	var cfg config
	err = env.Parse(&cfg)
	if err != nil {
		log.Fatalf("Error parse env: %s\n", err)
	}

	return cfg
}
func initLogger(cfg config) {
	config := zap.NewProductionConfig()
	config.EncoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logLevel := zap.NewAtomicLevelAt(zap.InfoLevel)
	if cfg.Debuglog {
		logLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
	}
	config.Level = logLevel

	logger, err := config.Build()
	if err != nil {
		log.Fatalf("Error build logger: %s\n", err)
	}
	defer logger.Sync()

	zap.ReplaceGlobals(logger)
}

func initTracer(cfg config) {
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(cfg.JaegerEndpoint)))
	if err != nil {
		zap.S().Fatal("Error init Jaeger exporter: ", zap.Error(err))
	}

	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(cfg.AppName),
			semconv.ServiceVersionKey.String(cfg.AppVersion),
			attribute.String("environment", cfg.Environment),
		),
	)
	if err != nil {
		zap.S().Fatal("Error init Jaeger resource: ", zap.Error(err))
	}

	tp := trace.NewTracerProvider(
		// Always be sure to batch in production.
		trace.WithBatcher(exp),
		// Record information about this application in a Resource.
		trace.WithResource(r),
	)

	otel.SetTracerProvider(tp)
}

func initRepositories(cfg config) (
	use_case.ApplicationRepository,
	use_case.SettingRepository,
	use_case.PcrdTHRepository,
	use_case.PcrdJPRepository,
	use_case.VersionRepository,
	use_case.HistoryRepository,
	use_case.VersionEventRepository,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoDbUri))
	if err != nil {
		zap.L().Fatal("Error init mongo client: ", zap.Error(err))
	}

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		zap.L().Fatal("Error ping mongo client: ", zap.Error(err))
	}

	appRepo := application_repository.NewRest(cfg.Service.Application)
	settingRepo := setting_repository.NewMongoDb(client.Database(cfg.MongoDbStoreVersion))
	pcrdTHRepo := pcrd_th_repository.NewRest(cfg.PCRD.THEndpoint, cfg.PCRD.THSalt)
	pcrdJPRepo := pcrd_jp_repository.NewRest(cfg.PCRD.JPEndpoint)
	versionRepo := version_repository.NewMongoDb(client.Database(cfg.MongoDbStoreVersion))
	historyRepo := history_repository.NewMongoDb(client.Database(cfg.MongoDbStoreVersion))

	versionEventRepo := version_event_repository.NewKafkaMQ(cfg.KafkaServer, cfg.KafkaTopicVersionEvent)
	return appRepo, settingRepo, pcrdTHRepo, pcrdJPRepo, versionRepo, historyRepo, versionEventRepo
}
