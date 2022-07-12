package game

import (
	"github.com/lonng/nano/examples/demo/tankDemo/cmd/server/pkg/errno"
	"github.com/lonng/nano/examples/demo/tankDemo/pb"
	"github.com/lonng/nano/scheduler"
	"log"
	"time"

	"github.com/lonng/nano"
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/session"
)

const kickResetBacklog = 8

var defaultManager = NewManager()

type (
	Manager struct {
		component.Base
		group   *nano.Group       // 广播channel
		players map[int64]*Player // 所有的玩家
		chKick  chan int64        // 退出队列
		chReset chan int64        // 重置队列
	}
)

func NewManager() *Manager {
	return &Manager{
		group:   nano.NewGroup("_SYSTEM_MESSAGE_BROADCAST"),
		players: map[int64]*Player{},
		chKick:  make(chan int64, kickResetBacklog),
		chReset: make(chan int64, kickResetBacklog),
	}
}

func (m *Manager) AfterInit() {
	session.Lifetime.OnClosed(func(s *session.Session) {
		m.group.Leave(s)
		m.offline(s.UID())
	})

	// 处理踢出玩家和重置玩家消息(来自http)
	scheduler.NewTimer(time.Second, func() {
	ctrl:
		for {
			select {
			case uid := <-m.chKick:
				p, ok := defaultManager.player(uid)
				if !ok || p.session == nil {
					log.Printf("玩家%d不在线", uid)
				}
				p.session.Close()
				log.Printf("踢出玩家, UID=%d", uid)

			case uid := <-m.chReset:
				p, ok := defaultManager.player(uid)
				if !ok {
					return
				}
				if p.session != nil {
					log.Printf("玩家正在游戏中，不能重置: %d", uid)
					return
				}
				p.room = nil
				log.Printf("重置玩家, UID=%d", uid)

			default:
				break ctrl
			}
		}
	})
}

// Login 玩家登陆
func (m *Manager) Login(s *session.Session, req *pb.Login_Request) error {
	uid := req.Uid
	s.Bind(uid)

	log.Printf("玩家: %d登录: %+v", uid, req)
	if p, ok := m.player(uid); !ok {
		log.Printf("玩家: %d不在线，创建新的玩家", uid)
		p = newPlayer(s, uid)
		m.setPlayer(uid, p)
	} else {
		log.Printf("玩家: %d已经在线", uid)
		// 重置之前的session
		if prevSession := p.session; prevSession != nil && prevSession != s {
			// 移除广播频道
			m.group.Leave(prevSession)

			// 如果之前房间存在，则退出来
			if p, err := playerWithSession(prevSession); err == nil && p != nil && p.room != nil && p.room.group != nil {
				p.room.group.Leave(prevSession)
			}

			prevSession.Clear()
			prevSession.Close()
		}

		// 绑定新session
		p.bindSession(s)
	}

	// 添加到广播频道
	m.group.Add(s)

	res := &pb.Login_Response{
		Error: &pb.ErrorInfo{
			Code: errno.CodeSuccess,
			Msg:  "登陆成功",
		},
	}

	return s.Response(res)
}

func (m *Manager) player(uid int64) (*Player, bool) {
	p, ok := m.players[uid]
	return p, ok
}

func (m *Manager) setPlayer(uid int64, p *Player) {
	if _, ok := m.players[uid]; ok {
		log.Printf("玩家已经存在，正在覆盖玩家， UID=%d", uid)
	}
	m.players[uid] = p
}

func (m *Manager) playerCount() int {
	return len(m.players)
}

func (m *Manager) offline(uid int64) {
	delete(m.players, uid)
	log.Printf("玩家: %d从在线列表中删除, 剩余：%d", uid, len(m.players))
}
