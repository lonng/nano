package chat

import (
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/session"
	"github.com/pingcap/errors"
)

type RoomService struct {
	component.Base
}

func newRoomService() *RoomService {
	return &RoomService{}
}

func (cs *RoomService) JoinTopic(s *session.Session, msg []byte) error {
	return errors.Errorf("not implement")
}
