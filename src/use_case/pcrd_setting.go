package use_case

import (
	"github.com/SpeedxPz/pcrd-version-updater/src/entity/credential"
	"github.com/SpeedxPz/pcrd-version-updater/src/entity/setting"
)

type PCRDSetting struct {
	Setting           setting.Setting
	Credential        credential.Credential
	GuessStartVersion string
}
