package cluster

import (
	"context"
	"net"

	"github.com/lonng/nano/cluster/clusterpb"
	"github.com/lonng/nano/internal/message"
	"github.com/lonng/nano/mock"
	"github.com/lonng/nano/session"
)

type acceptor struct {
	sid        int64
	gateClient clusterpb.MemberClient
	session    *session.Session
	lastMid    uint64
}

// Push implements the session.NetworkEntity interface
func (a *acceptor) Push(route string, v interface{}) error {
	// TODO: buffer
	data, err := message.Serialize(v)
	if err != nil {
		return err
	}
	request := &clusterpb.PushMessage{
		SessionId: a.sid,
		Route:     route,
		Data:      data,
	}
	_, err = a.gateClient.HandlePush(context.Background(), request)
	return err
}

// MID implements the session.NetworkEntity interface
func (a *acceptor) MID() uint64 {
	return a.lastMid
}

// Response implements the session.NetworkEntity interface
func (a *acceptor) Response(v interface{}) error {
	return a.ResponseMID(a.lastMid, v)
}

// ResponseMID implements the session.NetworkEntity interface
func (a *acceptor) ResponseMID(mid uint64, v interface{}) error {
	// TODO: buffer
	data, err := message.Serialize(v)
	if err != nil {
		return err
	}
	request := &clusterpb.ResponseMessage{
		SessionId: a.sid,
		Id:        mid,
		Data:      data,
	}
	_, err = a.gateClient.HandleResponse(context.Background(), request)
	return err
}

// Close implements the session.NetworkEntity interface
func (a *acceptor) Close() error {
	// TODO: buffer
	request := &clusterpb.CloseSessionRequest{
		SessionId: a.sid,
	}
	_, err := a.gateClient.CloseSession(context.Background(), request)
	return err
}

// RemoteAddr implements the session.NetworkEntity interface
func (*acceptor) RemoteAddr() net.Addr {
	return mock.NetAddr{}
}
