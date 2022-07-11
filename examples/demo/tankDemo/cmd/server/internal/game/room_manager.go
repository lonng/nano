package game

import (
	"fmt"
	"github.com/cute-angelia/go-utils/components/loggerV3"
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/examples/demo/tankDemo/cmd/server/pkg/constant"
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

func (manager *RoomManager) dumpRoomInfo() {
	c := len(manager.rooms)
	if c < 1 {
		return
	}

	log.Printf("剩余房间数量: %d 在线人数: %d  当前时间: %s", c, defaultManager.sessionCount(), time.Now().Format("2006-01-02 15:04:05"))
	for no, d := range manager.rooms {
		log.Printf("房号: %d, 创建时间: %s, 创建玩家: %d, 状态: %s",
			no, time.Unix(d.createdAt, 0).String(), d.owner.GetUid(), d.state.String())
	}
}

func (manager *RoomManager) SetRoom(roomId uint64, room *Room) {
	if room == nil {
		delete(manager.rooms, roomId)
		log.Printf("清除房间: 剩余: %d", len(manager.rooms))
	} else {
		manager.rooms[roomId] = room
	}
}

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
func (manager *RoomManager) Create(s *session.Session, data *pb.C2S_CreateRoomMsg) error {
	d, ok := manager.rooms[data.GetRoomId()]
	if ok {
		// 房间已存在
		return s.Response(deskNotFoundResponse)
	}

	if len(d.players) >= d.totalPlayerCount() {
		return s.Response(deskPlayerNumEnough)
	}

	// 如果是俱乐部房间，则判断玩家是否是俱乐部玩家
	// 否则直接加入房间
	if d.clubId > 0 {
		if db.IsClubMember(d.clubId, s.UID()) == false {
			return s.Response(&protocol.JoinDeskResponse{
				Code:  errorCode,
				Error: fmt.Sprintf("当前房间是俱乐部[%d]专属房间，俱乐部成员才可加入", d.clubId),
			})
		}
	}

	if err := d.playerJoin(s, false); err != nil {
		d.logger.Errorf("玩家加入房间失败，UID=%d, Error=%s", s.UID(), err.Error())
	}

	return s.Response(&protocol.JoinDeskResponse{
		TableInfo: protocol.TableInfo{
			DeskNo:    d.roomNo.String(),
			CreatedAt: d.createdAt,
			Creator:   d.creator,
			Title:     d.title(),
			Desc:      d.desc(true),
			Status:    d.status(),
			Round:     d.round,
			Mode:      d.opts.Mode,
		},
	})
}

// Join 加入房间
func (manager *RoomManager) Join(s *session.Session, data *pb.C2S_JoinRoomMsg) error {
	d, ok := manager.rooms[data.GetRoomId()]
	if !ok {
		return s.Response(deskNotFoundResponse)
	}

	if len(d.players) >= d.totalPlayerCount() {
		return s.Response(deskPlayerNumEnough)
	}

	// 如果是俱乐部房间，则判断玩家是否是俱乐部玩家
	// 否则直接加入房间
	if d.clubId > 0 {
		if db.IsClubMember(d.clubId, s.UID()) == false {
			return s.Response(&protocol.JoinDeskResponse{
				Code:  errorCode,
				Error: fmt.Sprintf("当前房间是俱乐部[%d]专属房间，俱乐部成员才可加入", d.clubId),
			})
		}
	}

	if err := d.playerJoin(s, false); err != nil {
		d.logger.Errorf("玩家加入房间失败，UID=%d, Error=%s", s.UID(), err.Error())
	}

	return s.Response(&protocol.JoinDeskResponse{
		TableInfo: protocol.TableInfo{
			DeskNo:    d.roomNo.String(),
			CreatedAt: d.createdAt,
			Creator:   d.creator,
			Title:     d.title(),
			Desc:      d.desc(true),
			Status:    d.status(),
			Round:     d.round,
			Mode:      d.opts.Mode,
		},
	})
}
