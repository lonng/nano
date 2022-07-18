package game

import (
	"github.com/lonng/nano/examples/demo/tankDemo/cmd/server/internal/common"
	"github.com/lonng/nano/examples/demo/tankDemo/cmd/server/internal/tank"
	"github.com/lonng/nano/session"
	log "github.com/sirupsen/logrus"
)

// 房间玩家状态
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

func NewPlayer(s *session.Session, uid int64) *Player {
	p := &Player{
		uid:    uid,
		ctx:    &tank.Context{Uid: uid},
		status: PlayerStatusDefault,
	}

	// reset
	p.ctx.Reset()

	// session
	p.BindSession(s)
	return p
}

// SetRoom 加入房间后
func (p *Player) SetRoom(r *Room) {
	if r == nil {
		log.Println("房间不存在")
		return
	}

	p.room = r

	p.ctx.RoomId = r.roomId
	p.ctx.Uid = p.uid
}

func (p *Player) GetRoom() *Room {
	return p.room
}

func (p *Player) GetStatus() playStatus {
	return p.status
}

func (p *Player) SetStatus(status playStatus) {
	p.status = status
}

// BindSession 登陆成功后
func (p *Player) BindSession(s *session.Session) {
	p.session = s
	p.session.Set(common.KeyCurPlayer, p)
}

func (p *Player) RemoveSession() {
	p.session.Remove(common.KeyCurPlayer)
	p.session = nil
}

func (p *Player) GetUid() int64 {
	return p.uid
}

func (p *Player) Reset() {
	// 重置channel
	p.ctx.Reset()
}
