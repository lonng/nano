package game

import (
	"github.com/lonng/nano/examples/demo/tankDemo/cmd/server/internal/game/tank"
	"github.com/lonng/nano/session"
	log "github.com/sirupsen/logrus"
)

type Loser struct {
	uid   uint64
	score int
}

type Player struct {
	uid int64 // 用户ID

	// 玩家数据
	session *session.Session

	// 游戏相关字段
	ctx *tank.Context

	room *Room //当前房间
}

func newPlayer(s *session.Session, uid int64) *Player {
	p := &Player{
		uid: uid,
		ctx: &tank.Context{Uid: uid},
	}

	p.ctx.Reset()
	p.bindSession(s)

	// 同步金币
	// p.syncCoinFromDB()

	return p
}

func (p *Player) setRoom(r *Room) {
	if r == nil {
		log.Println("房间不存在")
		return
	}

	p.ctx.RoomId = r.roomID
	p.ctx.Uid = p.uid
}

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
