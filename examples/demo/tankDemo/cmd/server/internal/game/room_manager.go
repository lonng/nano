package game

import (
	"github.com/cute-angelia/go-utils/components/loggerV3"
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/examples/demo/tankDemo/cmd/server/internal/common"
	"github.com/lonng/nano/examples/demo/tankDemo/cmd/server/internal/tank"
	"github.com/lonng/nano/examples/demo/tankDemo/cmd/server/pkg/constant"
	"github.com/lonng/nano/examples/demo/tankDemo/cmd/server/pkg/errno"
	"github.com/lonng/nano/examples/demo/tankDemo/pb"
	"github.com/lonng/nano/scheduler"
	"github.com/lonng/nano/session"
	"google.golang.org/protobuf/proto"
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
		log.Printf("房号: %d, 创建时间: %s, 创建玩家: %d, 当前状态: %s",
			no, time.Unix(d.createdAt, 0).String(), d.GetRoomMaster().GetUid(), d.state.String())
	}
}

// GetPlayerBySession 获取用户
func (manager *RoomManager) GetPlayerBySession(s *session.Session) *Player {
	return s.Value(common.KeyCurPlayer).(*Player)
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

func (manager *RoomManager) GetRoomBySession(s *session.Session) *Room {
	p := manager.GetPlayerBySession(s)
	room, ok := manager.rooms[p.GetRoom().roomId]
	if !ok {
		return nil
	}
	return room
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
	p.RemoveSession()

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
	_, ok := manager.rooms[data.GetRoomId()]
	if ok {
		// 房间已存在
		loggerV3.GetLogger().Err(errno.RoomExist.Error()).Send()
		return s.Response(&pb.CreateRoom_Response{
			Error: &pb.ErrorInfo{
				Code: errno.RoomExist.Int32(),
				Msg:  errno.RoomExist.String(),
			},
		})
	}

	player := s.Value(common.KeyCurPlayer).(*Player)

	// 创建房间
	// todo 指定为坦克游戏
	newR := NewRoom(data.GetRoomId(), player, data.MaxPlayerCount, tank.NewTank())
	manager.SetRoom(data.GetRoomId(), newR)

	// 玩家绑定房间
	player.SetRoom(newR)

	// 玩家加入房间
	if err := newR.JoinRoom(s, false); err != nil {
		log.Printf("玩家加入房间失败，UID=%d, Error=%s", s.UID(), err.Error())
		return s.Response(&pb.CreateRoom_Response{
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

	return s.Response(&pb.CreateRoom_Response{
		Data: &pb.RoomInfo{
			RoomId:     data.GetRoomId(),
			RoomMaster: newR.GetRoomMaster().GetUid(),
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

	// 玩家绑定房间
	player := manager.GetPlayerBySession(s)
	player.SetRoom(room)

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
	p := manager.GetPlayerBySession(s)

	log.Println("ready")

	room, ok := manager.rooms[p.GetRoom().roomId]
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
	room.prepare.ready(s.UID())
	p.SetStatus(PlayerStatusReady)

	s.Response(&pb.Ready_Response{
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

		// 房间开始游戏
		room.start()

	}

	return nil
}

func (manager *RoomManager) OnInput(s *session.Session, data *pb.Input_Notify) error {
	if room := manager.GetRoomBySession(s); room != nil {
		data.Uid = proto.Int64(s.UID())
		room.game.OnGameLockStepPush(data)

		log.Println("OnInput", data)
	}

	return nil
}
