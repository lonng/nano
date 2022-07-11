package game

import (
	"github.com/lonng/nano"
	"github.com/lonng/nano/examples/demo/tankDemo/cmd/server/pkg/constant"
	"github.com/lonng/nano/examples/demo/tankDemo/pb"
	"github.com/lonng/nano/scheduler"
	"github.com/lonng/nano/session"
	log "github.com/sirupsen/logrus"
)

type Room struct {
	roomID   uint64
	owner    *Player // 房主
	players  []*Player
	roomType pb.ROOMTYPE // 房间类型

	group *nano.Group // 组播通道
	die   chan struct{}

	createdAt int64               // 创建时间
	state     constant.RoomStatus // 状态
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
		p.reset()
		p.room = nil
		d.players[i] = nil
	}

	// 释放desk资源
	d.group.Close()

	//删除桌子
	scheduler.PushTask(func() {
		defaultDeskManager.setDesk(d.roomID, nil)
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
				p.reset()
				p.room = nil
			}
		}
		d.players = restPlayers
	}

	//如果桌上已无玩家, destroy it
	if d.owner.GetUid() == uid && !isDisconnect {
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

// totalPlayerCount 玩家总数
func (d *Room) totalPlayerCount() uint32 {
	if d.roomType == pb.ROOMTYPE_Room_Type_Two {
		return 2
	} else {
		return 2
	}
}
