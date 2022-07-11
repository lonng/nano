package game

import (
	"github.com/lonng/nano/scheduler"
	"github.com/lonng/nanoserver/protocol"

	"time"

	"github.com/lonng/nano"
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/session"
	log "github.com/sirupsen/logrus"
)

const kickResetBacklog = 8

var defaultManager = NewManager()

type (
	Manager struct {
		component.Base
		group      *nano.Group       // 广播channel
		players    map[int64]*Player // 所有的玩家
		chKick     chan int64        // 退出队列
		chReset    chan int64        // 重置队列
		chRecharge chan RechargeInfo // 充值信息
	}

	RechargeInfo struct {
		Uid  int64 // 用户ID
		Coin int64 // 房卡数量
	}
)

func NewManager() *Manager {
	return &Manager{
		group:      nano.NewGroup("_SYSTEM_MESSAGE_BROADCAST"),
		players:    map[int64]*Player{},
		chKick:     make(chan int64, kickResetBacklog),
		chReset:    make(chan int64, kickResetBacklog),
		chRecharge: make(chan RechargeInfo, 32),
	}
}

func (m *Manager) AfterInit() {
	session.Lifetime.OnClosed(func(s *session.Session) {
		m.group.Leave(s)
	})

	// 处理踢出玩家和重置玩家消息(来自http)
	scheduler.NewTimer(time.Second, func() {
	ctrl:
		for {
			select {
			case uid := <-m.chKick:
				p, ok := defaultManager.player(uid)
				if !ok || p.session == nil {
					logger.Errorf("玩家%d不在线", uid)
				}
				p.session.Close()
				logger.Infof("踢出玩家, UID=%d", uid)

			case uid := <-m.chReset:
				p, ok := defaultManager.player(uid)
				if !ok {
					return
				}
				if p.session != nil {
					logger.Errorf("玩家正在游戏中，不能重置: %d", uid)
					return
				}
				p.desk = nil
				logger.Infof("重置玩家, UID=%d", uid)

			case ri := <-m.chRecharge:
				player, ok := m.player(ri.Uid)
				// 如果玩家在线
				if s := player.session; ok && s != nil {
					s.Push("onCoinChange", &protocol.CoinChangeInformation{Coin: ri.Coin})
				}

			default:
				break ctrl
			}
		}
	})
}

func (m *Manager) Login(s *session.Session, req *protocol.LoginToGameServerRequest) error {
	uid := req.Uid
	s.Bind(uid)

	log.Infof("玩家: %d登录: %+v", uid, req)
	if p, ok := m.player(uid); !ok {
		log.Infof("玩家: %d不在线，创建新的玩家", uid)
		p = newPlayer(s, uid, req.Name, req.HeadUrl, req.IP, req.Sex)
		m.setPlayer(uid, p)
	} else {
		log.Infof("玩家: %d已经在线", uid)
		// 重置之前的session
		if prevSession := p.session; prevSession != nil && prevSession != s {
			// 移除广播频道
			m.group.Leave(prevSession)

			// 如果之前房间存在，则退出来
			if p, err := playerWithSession(prevSession); err == nil && p != nil && p.desk != nil && p.desk.group != nil {
				p.desk.group.Leave(prevSession)
			}

			prevSession.Clear()
			prevSession.Close()
		}

		// 绑定新session
		p.bindSession(s)
	}

	// 添加到广播频道
	m.group.Add(s)

	res := &protocol.LoginToGameServerResponse{
		Uid:      s.UID(),
		Nickname: req.Name,
		Sex:      req.Sex,
		HeadUrl:  req.HeadUrl,
		FangKa:   req.FangKa,
	}

	return s.Response(res)
}

func (m *Manager) player(uid int64) (*Player, bool) {
	p, ok := m.players[uid]

	return p, ok
}

func (m *Manager) setPlayer(uid int64, p *Player) {
	if _, ok := m.players[uid]; ok {
		log.Warnf("玩家已经存在，正在覆盖玩家， UID=%d", uid)
	}
	m.players[uid] = p
}

func (m *Manager) CheckOrder(s *session.Session, msg *protocol.CheckOrderReqeust) error {
	log.Infof("%+v", msg)

	return s.Response(&protocol.CheckOrderResponse{
		FangKa: 20,
	})
}

func (m *Manager) sessionCount() int {
	return len(m.players)
}

func (m *Manager) offline(uid int64) {
	delete(m.players, uid)
	log.Infof("玩家: %d从在线列表中删除, 剩余：%d", uid, len(m.players))
}
