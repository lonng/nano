package cluster_test

import (
	"testing"

	"github.com/lonng/nano/cluster"
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/session"
	. "github.com/pingcap/check"
)

type nodeSuite struct{}

var _ = Suite(&nodeSuite{})

type (
	MasterComponent  struct{ component.Base }
	Member1Component struct{ component.Base }
	Member2Component struct{ component.Base }
)

func (c *MasterComponent) Test(session *session.Session, _ []byte) error  { return nil }
func (c *Member1Component) Test(session *session.Session, _ []byte) error { return nil }
func (c *Member2Component) Test(session *session.Session, _ []byte) error { return nil }

func TestNode(t *testing.T) {
	TestingT(t)
}

func (s *nodeSuite) TestNodeStartup(c *C) {
	masterComps := &component.Components{}
	masterComps.Register(&MasterComponent{})
	masterNode := &cluster.Node{
		IsMaster:      true,
		AdvertiseAddr: "127.0.0.1:4450",
		ServerAddr:    "127.0.0.1:4451",
		Components:    masterComps,
	}
	err := masterNode.Startup()
	c.Assert(err, IsNil)
	masterHandler := masterNode.Handler()
	c.Assert(masterHandler.LocalService(), DeepEquals, []string{"MasterComponent"})

	member1Comps := &component.Components{}
	member1Comps.Register(&Member1Component{})
	memberNode1 := &cluster.Node{
		AdvertiseAddr: "127.0.0.1:4450",
		MemberAddr:    "127.0.0.1:14451",
		ServerAddr:    "127.0.0.1:14452",
		Components:    member1Comps,
	}
	err = memberNode1.Startup()
	c.Assert(err, IsNil)
	member1Handler := memberNode1.Handler()
	c.Assert(masterHandler.LocalService(), DeepEquals, []string{"MasterComponent"})
	c.Assert(masterHandler.RemoteService(), DeepEquals, []string{"Member1Component"})
	c.Assert(member1Handler.LocalService(), DeepEquals, []string{"Member1Component"})
	c.Assert(member1Handler.RemoteService(), DeepEquals, []string{"MasterComponent"})

	member2Comps := &component.Components{}
	member2Comps.Register(&Member2Component{})
	memberNode2 := &cluster.Node{
		AdvertiseAddr: "127.0.0.1:4450",
		MemberAddr:    "127.0.0.1:24451",
		ServerAddr:    "127.0.0.1:24452",
		Components:    member2Comps,
	}
	err = memberNode2.Startup()
	c.Assert(err, IsNil)
	member2Handler := memberNode2.Handler()
	c.Assert(masterHandler.LocalService(), DeepEquals, []string{"MasterComponent"})
	c.Assert(masterHandler.RemoteService(), DeepEquals, []string{"Member1Component", "Member2Component"})
	c.Assert(member1Handler.LocalService(), DeepEquals, []string{"Member1Component"})
	c.Assert(member1Handler.RemoteService(), DeepEquals, []string{"MasterComponent", "Member2Component"})
	c.Assert(member2Handler.LocalService(), DeepEquals, []string{"Member2Component"})
	c.Assert(member2Handler.RemoteService(), DeepEquals, []string{"MasterComponent", "Member1Component"})
}
