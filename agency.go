package nano

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	pb "github.com/jmesyan/nano/protos"
	"sync"
)

var (
	AgentManager       = new(AgentPool)
	cmdACK       int32 = 0x8000000
)

type Agency struct {
	Conn  pb.GrpcService_MServiceClient
	Cid   int32
	Cmd   int32
	T     int32
	N     int32
	Route string
}

func (a *Agency) SendMsg(cmd int32, data interface{}) error {
	msg := new(pb.GrpcMessage)
	msg.Cid = a.Cid
	msg.Cmd = cmd | cmdACK
	msg.T = a.T
	msg.N = a.N
	switch v := data.(type) {
	case proto.Message:
		send, err := serializer.Marshal(data)
		if err != nil {
			fmt.Print(err)
			return err
		}
		msg.Data = send
	case []byte:
		msg.Data = v
	default:
		fmt.Println("the type is:", v)
	}
	return a.Conn.Send(msg)
}

type AgentPool struct {
	pool sync.Pool
}

func (p *AgentPool) agentGet(stream pb.GrpcService_MServiceClient, message *pb.GrpcMessage) *Agency {
	var agent *Agency
	a := p.pool.Get()
	if a == nil {
		agent = new(Agency)
	} else {
		agent = a.(*Agency)
	}
	agent.Conn = stream
	agent.Cid = message.Cid
	agent.Cmd = message.Cmd
	agent.T = message.T
	agent.N = message.N
	agent.Route = message.Route
	return agent
}

func (p *AgentPool) agentPut(agent *Agency) {
	p.pool.Put(agent)
}
