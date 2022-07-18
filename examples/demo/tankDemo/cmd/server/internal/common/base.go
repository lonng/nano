package common

import (
	"github.com/lonng/nano/examples/demo/tankDemo/pb"
	"github.com/lonng/nano/session"
)

type GameStatus int

const (
	GameStatusReady GameStatus = iota
	GameStatusGameIng
	GameStatusOver
	GameStatusStop
)

type GameInterface interface {
	OnJoinGame(s *session.Session)
	OnGameStart()
	OnLeaveGame()
	OnGameOver()
	OnGameLockStepPush(notify *pb.Input_Notify)
}
