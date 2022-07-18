package game

import (
	"github.com/google/uuid"
	"github.com/lonng/nano"
	"github.com/lonng/nano/examples/demo/tankDemo/cmd/server/internal/common"
	"github.com/lonng/nano/examples/demo/tankDemo/cmd/server/pkg/constant"
	"github.com/lonng/nano/examples/demo/tankDemo/cmd/server/pkg/errno"
	"github.com/lonng/nano/scheduler"
	"github.com/lonng/nano/session"
	log "github.com/sirupsen/logrus"
	"time"
)

type Room struct {
	group *nano.Group         // 组播通道
	state constant.RoomStatus // 状态

	die chan struct{}

	roomId         uint64
	roomMaster     *Player // 房主
	players        []*Player
	maxPlayerCount uint32 // 最多人数

	prepare *prepareContext // 准备相关状态

	randomSeed int64 // 随机种子
	createdAt  int64 // 创建时间

	// 绑定游戏
	game common.GameInterface
}

// NewRoom 创建房间，设置房主，最大人数
func NewRoom(roomId uint64, owner *Player, maxPlayerCount uint32, game common.GameInterface) *Room {
	d := &Room{
		state:          constant.RoomStatusCreate,
		roomId:         roomId,
		players:        []*Player{},
		group:          nano.NewGroup(uuid.New().String()),
		die:            make(chan struct{}),
		randomSeed:     time.Now().Unix(),
		createdAt:      time.Now().Unix(),
		roomMaster:     owner,
		game:           game,
		prepare:        newPrepareContext(),
		maxPlayerCount: maxPlayerCount,
	}
	// d.players = append(d.players, owner)

	return d
}

func (d *Room) GetRoomMaster() *Player {
	return d.roomMaster
}

func (d *Room) SetRoomMaster(roomMaster *Player) {
	d.roomMaster = roomMaster
}

func (d *Room) GetMaxPlayerCount() uint32 {
	return d.maxPlayerCount
}

func (d *Room) SetMaxPlayerCount(maxPlayerCount uint32) {
	d.maxPlayerCount = maxPlayerCount
}

func (d *Room) SetState(state constant.RoomStatus) {
	d.state = state
}

func (d *Room) isDestroy() bool {
	return d.state == constant.RoomStatusDestory
}

// 摧毁
func (d *Room) destroy() {
	if d.state == constant.RoomStatusDestory {
		log.Println("桌子已经解散")
		return
	}

	close(d.die)

	// 标记为销毁
	d.SetState(constant.RoomStatusDestory)

	log.Println("销毁房间")
	for i := range d.players {
		p := d.players[i]
		log.Printf("销毁房间，清除玩家%d数据", p.GetUid())
		p.Reset()
		p.room = nil
		d.players[i] = nil
	}

	// 释放desk资源
	d.group.Close()

	//删除桌子
	scheduler.PushTask(func() {
		defaultRoomManager.SetRoom(d.roomId, nil)
	})
}

// 玩家退出
func (d *Room) onPlayerExit(s *session.Session, isDisconnect bool) {
	uid := s.UID()
	d.group.Leave(s)
	// 掉线
	if isDisconnect {
		// 统计
		// d.dissolve.updateOnlineStatus(uid, false)
	} else {
		// 退出房间
		restPlayers := []*Player{}
		for _, p := range d.players {
			if p.GetUid() != uid {
				restPlayers = append(restPlayers, p)
			} else {
				p.Reset()
				p.room = nil
			}
		}
		d.players = restPlayers
	}

	//如果桌上已无玩家, destroy it
	if d.GetRoomMaster().GetUid() == uid && !isDisconnect {
		//if d.dissolve.offlineCount() == len(d.players) || (d.creator == uid && !isDisconnect) {
		log.Println("所有玩家下线或房主主动解散房间")
		//if d.dissolve.isDissolving() {
		//	d.dissolve.stop()
		//}
		d.destroy()

		// 数据库异步更新
		//async.Run(func() {
		//	desk := &model.Desk{
		//		Id:    d.deskID,
		//		Round: 0,
		//	}
		//	if err := db.UpdateDesk(desk); err != nil {
		//		log.Error(err)
		//	}
		//})
	}
}

// 获取玩家
func (d *Room) getPlayerWithId(uid int64) (*Player, error) {
	for _, p := range d.players {
		if p.GetUid() == uid {
			return p, nil
		}
	}
	return nil, errno.RoomPlayerNotFound.Error()
}

// JoinRoom 玩家进入房间 如果是重新进入 isReJoin: true
func (d *Room) JoinRoom(s *session.Session, isReJoin bool) error {
	uid := s.UID()
	var (
		p   *Player
		err error
	)

	if isReJoin {
		p, err = d.getPlayerWithId(uid)
		if err != nil {
			log.Printf("玩家: %d重新加入房间, 但是没有找到玩家在房间中的数据", uid)
			return err
		}

		// 加入分组
		d.group.Add(s)
	} else {
		exists := false
		for _, p := range d.players {
			if p.GetUid() == uid {
				exists = true
				log.Println("玩家已经在房间中")
				break
			}
		}
		if !exists {
			d.group.Add(s)

			p = s.Value(common.KeyCurPlayer).(*Player)
			d.players = append(d.players, p)
			for _, p := range d.players {
				p.SetRoom(d)
			}
		}
	}

	// 游戏类处理
	d.game.OnJoinGame(s)
	return nil
}

// start 开始游戏
func (d *Room) start() {
	log.Println("游戏开始")

	// 游戏类处理
	d.game.OnGameStart()
}
