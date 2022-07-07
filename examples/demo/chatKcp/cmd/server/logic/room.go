package logic

import (
	"fmt"
	"github.com/lonng/nano"
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/examples/demo/chatKcp/pb"
	"github.com/lonng/nano/scheduler"
	"github.com/lonng/nano/session"
	"google.golang.org/protobuf/proto"
	"log"
	"time"
)

type Room struct {
	group *nano.Group
}

const (
	roomIDKey = "ROOM_ID"
)

// RoomManager represents a component that contains a bundle of room
type RoomManager struct {
	component.Base
	timer *scheduler.Timer
	rooms map[uint64]*Room
}

func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms: map[uint64]*Room{},
	}
}

// AfterInit component lifetime callback
func (mgr *RoomManager) AfterInit() {
	session.Lifetime.OnClosed(func(s *session.Session) {
		if !s.HasKey(roomIDKey) {
			return
		}
		room := s.Value(roomIDKey).(*Room)
		room.group.Leave(s)
	})
	mgr.timer = scheduler.NewTimer(time.Minute, func() {
		for roomId, room := range mgr.rooms {
			println(fmt.Sprintf("UserCount: RoomID=%d, Time=%s, Count=%d",
				roomId, time.Now().String(), room.group.Count()))
		}
	})
}

// Join room
func (mgr *RoomManager) Join(s *session.Session, msg *pb.C2S_JoinRoomMsg) error {

	log.Println(msg)

	// s.HasKey()
	room, found := mgr.rooms[msg.GetRoomId()]
	if !found {
		room = &Room{
			group: nano.NewGroup(fmt.Sprintf("room-%d", msg.GetRoomId())),
		}
		mgr.rooms[msg.GetRoomId()] = room
	}

	fakeUID := s.ID() //just use s.ID as uid !!!
	s.Bind(fakeUID)   // binding session uids.Set(roomIDKey, room)
	s.Set(roomIDKey, room)

	// push
	// s.Push("onMembers", &AllMembers{Members: room.group.Members()})

	// notify others
	// room.group.Broadcast("onNewUser", &NewUser{Content: fmt.Sprintf("New user: %d", s.ID())})

	// new user join group
	room.group.Add(s) // add session to group

	return s.Response(&pb.S2C_JoinRoomMsg{
		RoomSeatId: proto.Int32(3),
	})
}

// Message sync last message to all members
//func (mgr *RoomManager) Message(s *session.Session, msg *UserMessage) error {
//	if !s.HasKey(roomIDKey) {
//		return fmt.Errorf("not join room yet")
//	}
//	room := s.Value(roomIDKey).(*Room)
//	return room.group.Broadcast("onMessage", msg)
//}
