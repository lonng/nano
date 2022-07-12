package game

import (
	"github.com/cute-angelia/go-utils/components/loggerV3"
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/examples/demo/tankDemo/cmd/server/pkg/constant"
	"github.com/lonng/nano/examples/demo/tankDemo/cmd/server/pkg/errno"
	"github.com/lonng/nano/examples/demo/tankDemo/pb"
	"github.com/lonng/nano/scheduler"
	"github.com/lonng/nano/session"
	"log"
	"time"
)

type (
	RoomManager struct {
		component.Base
		rooms map[uint64]*Room
	}
)

var defaultRoomManager = NewRoomManager()

func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms: map[uint64]*Room{},
	}
}

func (manager *RoomManager) AfterInit() {
	session.Lifetime.OnClosed(func(s *session.Session) {
		// Fixed: 玩家WIFI切换到4G网络不断开, 重连时，将UID设置为illegalSessionUid
		if s.UID() > 0 {
			if err := manager.onPlayerDisconnect(s); err != nil {
				loggerV3.GetLogger().Info().Msgf("玩家退出: UID=%d, Error=%s", s.UID, err.Error())
			}
		}
	})

	// 每5分钟清空一次已摧毁的房间信息
	scheduler.NewTimer(300*time.Second, func() {
		destroyDesk := map[uint64]*Room{}
		deadline := time.Now().Add(-24 * time.Hour).Unix()
		for no, d := range manager.rooms {
			// 清除创建超过24小时的房间
			if d.state == constant.RoomStatusDestory || d.createdAt < deadline {
				destroyDesk[no] = d
			}
		}
		for _, d := range destroyDesk {
			d.destroy()
		}

		// 打印信息
		manager.dumpRoomInfo()

		// 统计结果异步写入数据库
	})
}

// 打印信息
func (manager *RoomManager) dumpRoomInfo() {
	c := len(manager.rooms)
	if c < 1 {
		return
	}

	log.Printf("剩余房间数量: %d 在线人数: %d  当前时间: %s", c, defaultManager.playerCount(), time.Now().Format("2006-01-02 15:04:05"))
	for no, d := range manager.rooms {
		log.Printf("房号: %d, 创建时间: %s, 创建玩家: %d, 状态: %s",
			no, time.Unix(d.createdAt, 0).String(), d.owner.GetUid(), d.state.String())
	}
}

// 保存房间
func (manager *RoomManager) SetRoom(roomId uint64, room *Room) {
	if room == nil {
		delete(manager.rooms, roomId)
		log.Printf("清除房间: 剩余: %d", len(manager.rooms))
	} else {
		manager.rooms[roomId] = room
	}
}

func (manager *RoomManager) GetRoom(roomId uint64) *Room {
	return manager.rooms[roomId]
}

// 事件
func (manager *RoomManager) onPlayerDisconnect(s *session.Session) error {
	uid := s.UID()
	p, err := playerWithSession(s)
	if err != nil {
		return err
	}
	log.Println("DeskManager.onPlayerDisconnect: 玩家网络断开")

	// 移除session
	p.removeSession()

	if p.room == nil || p.room.isDestroy() {
		defaultManager.offline(uid)
		return nil
	}

	d := p.room
	d.onPlayerExit(s, true)
	return nil
}

// Create 创建房间
func (manager *RoomManager) CreateRoom(s *session.Session, data *pb.CreateRoom_Request) error {
	d, ok := manager.rooms[data.GetRoomId()]
	if ok {
		// 房间已存在
		return s.Response(pb.CreateRoom_Response{
			Error: &pb.ErrorInfo{
				Code: errno.RoomExist.Int32(),
				Msg:  errno.RoomExist.String(),
			},
		})
	}

	// 人数足够了
	if len(d.players) >= int(d.GetMaxPlayerCount()) {
		return s.Response(pb.CreateRoom_Response{
			Error: &pb.ErrorInfo{
				Code: errno.RoomPlayerNumEnough.Int32(),
				Msg:  errno.RoomPlayerNumEnough.String(),
			},
		})
	}

	// 创建房间
	p := s.Value(kCurPlayer).(*Player)
	newR := NewRoom(s, data.GetRoomId(), p, data.MaxPlayerCount)
	manager.SetRoom(data.GetRoomId(), newR)

	// 玩家加入房间
	if err := d.JoinRoom(s, false); err != nil {
		log.Printf("玩家加入房间失败，UID=%d, Error=%s", s.UID(), err.Error())

		return s.Response(pb.CreateRoom_Response{
			Error: &pb.ErrorInfo{
				Code: errno.RoomJoinFailed.Int32(),
				Msg:  errno.RoomJoinFailed.String() + err.Error(),
			},
		})
	}

	users := []*pb.UserInfo{}
	for _, i2 := range newR.players {
		users = append(users, &pb.UserInfo{
			Uid: i2.GetUid(),
		})
	}

	return s.Response(pb.CreateRoom_Response{
		Data: &pb.RoomInfo{
			RoomId:     data.GetRoomId(),
			RandomSeed: newR.randomSeed,
			Players:    users,
		},
	})
}

// Join 加入房间
func (manager *RoomManager) JoinRoom(s *session.Session, data *pb.JoinRoom_Request) error {
	room, ok := manager.rooms[data.GetRoomId()]
	if !ok {
		// 房间不存在
		return s.Response(pb.JoinRoom_Response{
			Error: &pb.ErrorInfo{
				Code: errno.RoomNotFound.Int32(),
				Msg:  errno.RoomNotFound.String(),
			},
		})
	}

	// 人数足够了
	if len(room.players) >= int(room.GetMaxPlayerCount()) {
		return s.Response(pb.JoinRoom_Response{
			Error: &pb.ErrorInfo{
				Code: errno.RoomPlayerNumEnough.Int32(),
				Msg:  errno.RoomPlayerNumEnough.String(),
			},
		})
	}

	// 玩家加入房间
	if err := room.JoinRoom(s, false); err != nil {
		log.Printf("玩家加入房间失败，UID=%d, Error=%s", s.UID(), err.Error())

		return s.Response(pb.JoinRoom_Response{
			Error: &pb.ErrorInfo{
				Code: errno.RoomJoinFailed.Int32(),
				Msg:  errno.RoomJoinFailed.String() + err.Error(),
			},
		})
	}

	users := []*pb.UserInfo{}
	for _, i2 := range room.players {
		users = append(users, &pb.UserInfo{
			Uid: i2.GetUid(),
		})
	}

	// 广播
	defer func() {
		room.group.Broadcast("JoinRoom", pb.JoinRoom_Response{
			Data: &pb.RoomInfo{
				RoomId:     data.GetRoomId(),
				RandomSeed: room.randomSeed,
				Players:    users,
			},
		})
	}()

	return s.Response(pb.JoinRoom_Response{
		Data: &pb.RoomInfo{
			RoomId:     data.GetRoomId(),
			RandomSeed: room.randomSeed,
			Players:    users,
		},
	})
}

// 离开房间
func (manager *RoomManager) LeaveRoom(s *session.Session, data *pb.LeaveRoom_Request) error {
	_, ok := manager.rooms[data.GetRoomId()]
	if !ok {
		// 房间不存在
		return s.Response(pb.LeaveRoom_Response{
			Error: &pb.ErrorInfo{
				Code: errno.RoomNotFound.Int32(),
				Msg:  errno.RoomNotFound.String(),
			},
		})
	}

	manager.SetRoom(data.RoomId, nil)

	return s.Response(pb.LeaveRoom_Response{
		Error: &pb.ErrorInfo{
			Code: errno.CodeSuccess,
			Msg:  "离开房间成功",
		},
	})
}

func (manager *RoomManager) Ready(s *session.Session, data *pb.Ready_Request) error {
	p := getPlayerBySession(s)
	room, ok := manager.rooms[p.getRoom().roomId]
	if !ok {
		// 房间不存在
		return s.Response(pb.Ready_Response{
			Error: &pb.ErrorInfo{
				Code: errno.RoomNotFound.Int32(),
				Msg:  errno.RoomNotFound.String(),
			},
		})
	}

	// 设置状态
	p.SetStatus(PlayerStatusReady)

	s.Response(pb.Ready_Response{
		Error: &pb.ErrorInfo{
			Code: errno.CodeSuccess,
			Msg:  "准备中",
		},
	})

	// 判断是否所有人都准备了， 准备了，开始游戏
	readyCount := 0
	if int(room.GetMaxPlayerCount()) == len(room.players) {
		for _, player := range room.players {
			if player.status == PlayerStatusReady {
				readyCount = readyCount + 1
			}
		}
	}

	if int(room.GetMaxPlayerCount()) == readyCount {
		// 通知开始游戏
		room.group.Broadcast("StartGame", &pb.Start_Notify{})
	}

	return nil
}
