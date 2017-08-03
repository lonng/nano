package nano

import "errors"

var (
	ErrSessionOnNotify = errors.New("current session working on notify mode")
)
