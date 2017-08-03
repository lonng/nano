package nano

import "errors"

var (
	ErrSessionOnNotify  = errors.New("current session working on notify mode")
	ErrCloseClosedGroup = errors.New("close closed group")
	ErrClosedGroup      = errors.New("group closed")
	ErrMemberNotFound   = errors.New("member not found in the group")
	ErrClosedSession    = errors.New("close closed session")
)
