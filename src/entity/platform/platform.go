package platform

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidPlatform = errors.New("unknow platform")
)

type PlatformType string

const (
	PlatformTypeNone    PlatformType = ""
	PlatformTypeAndroid PlatformType = "android"
	PlatformTypeIOS     PlatformType = "ios"
)

func ParsePlatformType(s string) (d PlatformType, e error) {
	dataTypes := map[PlatformType]struct{}{
		PlatformTypeNone:    {},
		PlatformTypeAndroid: {},
		PlatformTypeIOS:     {},
	}

	dat := PlatformType(s)
	_, ok := dataTypes[dat]
	if !ok {
		return d, fmt.Errorf("cannot parse:[%s] as platform: %w", s, ErrInvalidPlatform)
	}
	return dat, nil
}
