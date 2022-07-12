package game

import (
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/examples/demo/tankDemo/pb"
	"github.com/lonng/nano/session"
	"log"
)

var defaultNewTest = newTest()

type (
	Test struct {
		component.Base
	}
)

func newTest() *Test {
	return &Test{}
}

func (t *Test) AfterInit() {
	log.Println("good")
}

func (t *Test) CreateRoom(s *session.Session, data *pb.CreateRoom_Request) error {
	log.Println(s)
	return nil
}
