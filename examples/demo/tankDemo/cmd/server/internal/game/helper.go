package game

import (
	"errors"
	"github.com/lonng/nano/session"
)

func playerWithSession(s *session.Session) (*Player, error) {
	p, ok := s.Value(kCurPlayer).(*Player)
	if !ok {
		return nil, errors.New("玩家不存在")
	}
	return p, nil
}
