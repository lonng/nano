package utils

import (
	"github.com/lonng/nano/examples/demo/tankDemo/cmd/server/internal/common"
	"github.com/lonng/nano/examples/demo/tankDemo/cmd/server/internal/game"
	"github.com/lonng/nano/session"
)

func GetPlayerBySession(s *session.Session) *game.Player {
	return s.Value(common.KeyCurPlayer).(*game.Player)
}
