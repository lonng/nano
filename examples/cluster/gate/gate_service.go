package gate

import (
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/examples/cluster/protocol"
	"github.com/lonng/nano/session"
	"github.com/pingcap/errors"
)

type BindService struct {
	component.Base
	nextGateUid int64
}

func newBindService() *BindService {
	return &BindService{}
}

type LoginRequest struct {
	Nickname string `json:"nickname"`
}

func (bs *BindService) Login(s *session.Session, msg *LoginRequest) error {
	bs.nextGateUid++
	uid := bs.nextGateUid
	return s.RPC("TopicService.NewUser", &protocol.NewUserRequest{
		Nickname: msg.Nickname,
		GateUid:  uid,
	})
}

func (bs *BindService) BindChatServer(s *session.Session, msg []byte) error {
	return errors.Errorf("not implement")
}
