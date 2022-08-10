package setting

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidServerCode = errors.New("unknow server code")
)

type ServerCode string

const (
	ServerCodeNone ServerCode = ""
	ServerCodeTH   ServerCode = "th"
	ServerCodeJP   ServerCode = "jp"
)

func ParseServerCode(s string) (d ServerCode, e error) {
	dataTypes := map[ServerCode]struct{}{
		ServerCodeNone: {},
		ServerCodeTH:   {},
		ServerCodeJP:   {},
	}

	dat := ServerCode(s)
	_, ok := dataTypes[dat]
	if !ok {
		return d, fmt.Errorf("cannot parse:[%s] as servercode: %w", s, ErrInvalidServerCode)
	}
	return dat, nil
}

type Setting struct {
	ID         string
	ServerCode ServerCode
}
