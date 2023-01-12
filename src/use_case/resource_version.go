package use_case

import (
	"context"
	"errors"
	"fmt"
	"github.com/SpeedxPz/pcrd-version-updater/src/entity/logger"
	"github.com/SpeedxPz/pcrd-version-updater/src/entity/setting"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

func (u UseCase) UpdateResourceVersion(
	ctx context.Context,
	ID string,
) error {
	ctx, span := tracer.Start(ctx, fmt.Sprintf("use_case.UpdateResourceVersion(%s)", ID))
	defer span.End()
	zap.L().Info("use_case.UpdateResourceVersion",
		logger.WithTraceId(ctx),
		zap.Any("ID", ID),
	)

	appSetting, err := u.settingRepository.GetSettingByID(ctx, ID)
	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("%s", err))
		return err
	}

	application, err := u.applicationRepository.GetAndroidAppByID(ctx, appSetting.Setting.ID)
	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("%s", err))
		return err
	}

	currentVersion, err := u.versionRepository.GetByID(ctx, appSetting.Setting.ID)
	if err != nil {
		if errors.Is(err, ErrVersionNotFound) {
			gameVersion := GameVersion{
				Setting:    appSetting.Setting,
				AppVersion: application.Version,
				ResVersion: "",
			}
			err = u.versionRepository.Create(ctx, gameVersion)
			if err != nil {
				span.SetStatus(codes.Error, fmt.Sprintf("%s", err))
				return err
			}
			currentVersion = gameVersion
		} else {
			span.SetStatus(codes.Error, fmt.Sprintf("%s", err))
			return err
		}
	}

	var version string

	if appSetting.Setting.ServerCode == setting.ServerCodeTH {
		result, err := u.pcrdTHRepository.GetResourceVersion(ctx, appSetting.Credential, PcrdVersion{
			AppVersion: application.Version,
		})
		if err != nil {
			span.SetStatus(codes.Error, fmt.Sprintf("%s", err))
			return err
		}
		version = result
	} else {
		// Japan Logic
		guessVersion := currentVersion.ResVersion
		if len(guessVersion) <= 0 {
			guessVersion = appSetting.GuessStartVersion
		}

		result, err := u.pcrdJPRepository.GetResourceVersion(ctx, guessVersion)
		if err != nil {
			span.SetStatus(codes.Error, fmt.Sprintf("%s", err))
			return err
		}
		version = result
	}

	if currentVersion.ResVersion == version {
		zap.L().Info("use_case.UpdateResourceVersion",
			logger.WithTraceId(ctx),
			zap.Any("message", "nothing to update"),
		)
		// Nothing to update
		return nil
	}

	currentVersion.ResVersion = version
	currentVersion.AppVersion = application.Version

	err = u.versionRepository.Update(ctx, currentVersion)
	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("%s", err))
		return err
	}

	err = u.historyRepository.Create(ctx, currentVersion)
	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("%s", err))
		return err
	}

	u.versionEventRepository.PublishVersion(ctx, currentVersion)
	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("%s", err))
		return err
	}

	return nil
}
