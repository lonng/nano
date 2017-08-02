package main

import (
	"net/http"
	"log"

	"github.com/lonnng/nano"
	"github.com/lonnng/nano/component"
	"github.com/lonnng/nano/serialize/json"
	"github.com/lonnng/nano/session"
)

type Room struct {
	component.Base
	group *nano.Group
}

type UserMessage struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type JoinResponse struct {
	Code   int    `json:"code"`
	Result string `json:"result"`
}

func NewRoom() *Room {
	return &Room{
		group: nano.NewGroup("room"),
	}
}

func (r *Room) Join(s *session.Session, msg []byte) error {
	s.Bind(s.ID)   // binding session uid
	r.group.Add(s) // add session to group
	return s.Response(JoinResponse{Result: "sucess"})
}

func (r *Room) Message(s *session.Session, msg *UserMessage) error {
	return r.group.Broadcast("onMessage", msg)
}

func main() {
	nano.Register(NewRoom())
	nano.SetSerializer(json.NewSerializer())

	log.SetFlags(log.LstdFlags | log.Llongfile)

	nano.SetCheckOriginFunc(func(_ *http.Request) bool { return true })
	nano.ListenWithOptions(":3250", true)
}
