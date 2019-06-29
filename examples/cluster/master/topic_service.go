package master

import (
	"fmt"

	"github.com/lonng/nano"
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
}

type TopicService struct {
	component.Base
	nextUid int64
	users   map[int64]*User
	group   *nano.Group
}

func newTopicService() *TopicService {
	return &TopicService{
		users: map[int64]*User{},
		group: nano.NewGroup("all-users"),
	}
}

func (ts *TopicService) NewUser(s *session.Session, msg *protocol.NewUserRequest) error {
	// exists users

	ts.nextUid++
	uid := ts.nextUid
	user := &User{
		session:  s,
		nickname: msg.Nickname,
		gateId:   msg.GateUid,
		masterId: uid,
	}
	ts.users[uid] = user

	broadcast := &protocol.NewUserBroadcast{
		Content: fmt.Sprintf("User user join: %v", msg.Nickname),
	}
	if err := ts.group.Broadcast("onNewUser", broadcast); err != nil {
		return err
	}
	return ts.group.Add(s)
}

type OpenTopicRequest struct {
	Name string `json:"name"`
}

func (ts *TopicService) OpenTopic(s *session.Session, msg *OpenTopicRequest) error {
	return errors.Errorf("not implemented: %v", msg)
}
