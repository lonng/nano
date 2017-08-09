package logic

import (
	"log"

	"github.com/lonnng/nano/component"
	"github.com/lonnng/nano/examples/demo/tadpole/logic/protocol"
	"github.com/lonnng/nano/session"
)

type Manager struct {
	component.Base
}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) Login(s *session.Session, msg *protocol.JoyLoginRequest) error {
	log.Println(msg)
	id := s.ID()
	s.Bind(id)
	return s.Response(protocol.LoginResponse{
		Status: protocol.LoginStatusSucc,
		ID:     id,
	})
}
