package game

import (
	"github.com/lonng/nano/examples/demo/tankDemo/cmd/server/internal/game/tank"
	"github.com/lonng/nano/session"
	log "github.com/sirupsen/logrus"
)

type playStatus int32

const (
	PlayerStatusDefault playStatus = iota
	PlayerStatusReady
	PlayerStatusGame
)

type Player struct {
	uid int64 // 用户ID

	// 玩家数据
	session *session.Session

	// 游戏相关字段
	ctx *tank.Context

	room *Room //当前房间

	// 状态
	status playStatus
}

func (p *Player) GetStatus() playStatus {
	return p.status
}

func (p *Player) SetStatus(status playStatus) {
	p.status = status
}

func newPlayer(s *session.Session, uid int64) *Player {
	p := &Player{
		uid:    uid,
		ctx:    &tank.Context{Uid: uid},
		status: PlayerStatusDefault,
	}

	// reset
	p.ctx.Reset()

	// session
	p.bindSession(s)
	return p
}

func getPlayerBySession(s *session.Session) *Player {
	return s.Value(kCurPlayer).(*Player)
}

// 加入房间后，setRoom
func (p *Player) setRoom(r *Room) {
	if r == nil {
		log.Println("房间不存在")
		return
	}

	p.ctx.RoomId = r.roomId
	p.ctx.Uid = p.uid
}

func (p *Player) getRoom() *Room {
	return p.room
}

// 登陆成功后
func (p *Player) bindSession(s *session.Session) {
	p.session = s
	p.session.Set(kCurPlayer, p)
}

func (p *Player) removeSession() {
	p.session.Remove(kCurPlayer)
	p.session = nil
}

func (p *Player) GetUid() int64 {
	return p.uid
}

func (p *Player) reset() {
	// 重置channel
	p.ctx.Reset()
}
