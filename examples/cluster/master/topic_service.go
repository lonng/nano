package master

import (
	"log"
	"strings"

	"github.com/lonng/nano/component"
	"github.com/lonng/nano/examples/cluster/protocol"
	"github.com/lonng/nano/session"
	"github.com/pingcap/errors"
)

type User struct {
	session  *session.Session
	nickname string
	gateId   int64
	masterId int64
	balance  int64
	message  int
}

type TopicService struct {
	component.Base
	nextUid int64
	users   map[int64]*User
}

func newTopicService() *TopicService {
	return &TopicService{
		users: map[int64]*User{},
	}
}

type ExistsMembersResponse struct {
	Members string `json:"members"`
}

func (ts *TopicService) NewUser(s *session.Session, msg *protocol.NewUserRequest) error {
	ts.nextUid++
	uid := ts.nextUid
	if err := s.Bind(uid); err != nil {
		return errors.Trace(err)
	}

	var members []string
	for _, u := range ts.users {
		members = append(members, u.nickname)
	}
	err := s.Push("onMembers", &ExistsMembersResponse{Members: strings.Join(members, ",")})
	if err != nil {
		return errors.Trace(err)
	}

	user := &User{
		session:  s,
		nickname: msg.Nickname,
		gateId:   msg.GateUid,
		masterId: uid,
		balance:  1000,
	}
	ts.users[uid] = user

	chat := &protocol.JoinRoomRequest{
		Nickname:  msg.Nickname,
		GateUid:   msg.GateUid,
		MasterUid: uid,
	}
	return s.RPC("RoomService.JoinRoom", chat)
}

type UserBalanceResponse struct {
	CurrentBalance int64 `json:"currentBalance"`
}

func (ts *TopicService) Stats(s *session.Session, msg *protocol.MasterStats) error {
	// It's OK to use map without lock because of this service running in main thread
	user, found := ts.users[msg.Uid]
	if !found {
		return errors.Errorf("User not found: %v", msg.Uid)
	}
	user.message++
	user.balance--
	return s.Push("onBalance", &UserBalanceResponse{user.balance})
}

func (ts *TopicService) userDisconnected(s *session.Session) {
	uid := s.UID()
	delete(ts.users, uid)
	log.Println("User session disconnected", s.UID())
}
