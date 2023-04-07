// Copyright (c) nano Authors. All Rights Reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package cluster

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lonng/nano/cluster/clusterpb"
	"github.com/lonng/nano/internal/env"
	"github.com/lonng/nano/internal/log"
)

// cluster represents a nano cluster, which contains a bunch of nano nodes
// and each of them provide a group of different services. All services requests
// from client will send to gate firstly and be forwarded to appropriate node.
type cluster struct {
	// If cluster is not large enough, use slice is OK
	currentNode *Node
	rpcClient   *rpcClient

	mu      sync.RWMutex
	members []*Member
}

func newCluster(currentNode *Node) *cluster {
	c := &cluster{currentNode: currentNode}
	if currentNode.IsMaster {
		c.checkMemberHeartbeat()
	}
	return c
}

// Register implements the MasterServer gRPC service
func (c *cluster) Register(_ context.Context, req *clusterpb.RegisterRequest) (*clusterpb.RegisterResponse, error) {
	if req.MemberInfo == nil {
		return nil, ErrInvalidRegisterReq
	}
	resp := &clusterpb.RegisterResponse{}
	c.mu.Lock()
	for k, m := range c.members {
		if m.memberInfo.ServiceAddr == req.MemberInfo.ServiceAddr {
			// 节点异常崩溃，不会执行unregister，此时再次启动该节点，由于已存在注册信息，将再也无法成功注册，这里做个修改，先移除后重新注册
			if k >= len(c.members)-1 {
				c.members = c.members[:k]
			} else {
				c.members = append(c.members[:k], c.members[k+1:]...)
			}
			break
			//return nil, fmt.Errorf("address %s has registered", req.MemberInfo.ServiceAddr)
		}
	}
	c.mu.Unlock()

	// Notify registered node to update remote services
	newMember := &clusterpb.NewMemberRequest{MemberInfo: req.MemberInfo}
	for _, m := range c.members {
		resp.Members = append(resp.Members, m.memberInfo)
		if m.isMaster {
			continue
		}
		pool, err := c.rpcClient.getConnPool(m.memberInfo.ServiceAddr)
		if err != nil {
			return nil, err
		}
		client := clusterpb.NewMemberClient(pool.Get())
		_, err = client.NewMember(context.Background(), newMember)
		if err != nil {
			return nil, err
		}
	}

	log.Println("New peer register to cluster", req.MemberInfo.ServiceAddr)

	// Register services to current node
	c.currentNode.handler.addRemoteService(req.MemberInfo)
	c.mu.Lock()
	c.members = append(c.members, &Member{isMaster: false, memberInfo: req.MemberInfo, lastHeartbeatAt: time.Now()})
	c.mu.Unlock()
	return resp, nil
}

// Unregister implements the MasterServer gRPC service
func (c *cluster) Unregister(_ context.Context, req *clusterpb.UnregisterRequest) (*clusterpb.UnregisterResponse, error) {
	if req.ServiceAddr == "" {
		return nil, ErrInvalidRegisterReq
	}

	var index = -1
	resp := &clusterpb.UnregisterResponse{}
	for i, m := range c.members {
		if m.memberInfo.ServiceAddr == req.ServiceAddr {
			index = i
			break
		}
	}
	if index < 0 {
		return nil, fmt.Errorf("address %s has not registered", req.ServiceAddr)
	}

	// Notify registered node to update remote services
	delMember := &clusterpb.DelMemberRequest{ServiceAddr: req.ServiceAddr}
	for i, m := range c.members {
		if i == index {
			// this node is down.
			continue
		}

		if m.MemberInfo().ServiceAddr == c.currentNode.ServiceAddr {
			continue
		}
		pool, err := c.rpcClient.getConnPool(m.memberInfo.ServiceAddr)
		if err != nil {
			return nil, err
		}
		client := clusterpb.NewMemberClient(pool.Get())
		_, err = client.DelMember(context.Background(), delMember)
		if err != nil {
			return nil, err
		}
	}

	log.Println("Exists peer unregister to cluster", req.ServiceAddr)

	if c.currentNode.UnregisterCallback != nil {
		c.currentNode.UnregisterCallback(*c.members[index])
	}

	// Register services to current node
	c.currentNode.handler.delMember(req.ServiceAddr)
	c.mu.Lock()
	if index >= len(c.members)-1 {
		c.members = c.members[:index]
	} else {
		c.members = append(c.members[:index], c.members[index+1:]...)
	}
	c.mu.Unlock()

	return resp, nil
}

func (c *cluster) Heartbeat(_ context.Context, req *clusterpb.HeartbeatRequest) (*clusterpb.HeartbeatResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	isHit := false
	for i, m := range c.members {
		if m.MemberInfo().GetServiceAddr() == req.GetMemberInfo().GetServiceAddr() {
			c.members[i].lastHeartbeatAt = time.Now()
			isHit = true
		}
	}
	if !isHit {
		// master local not binding this node, other members do not need to be notified, because this node registered.
		// maybe the master process reload
		m := &Member{
			isMaster:        false,
			memberInfo:      req.GetMemberInfo(),
			lastHeartbeatAt: time.Now(),
		}
		c.members = append(c.members, m)
		c.currentNode.handler.addRemoteService(req.MemberInfo)
		log.Println("Heartbeat peer register to cluster", req.MemberInfo.ServiceAddr)
	}
	return &clusterpb.HeartbeatResponse{}, nil
}

func (c *cluster) checkMemberHeartbeat() {
	check := func() {
		unregisterMembers := make([]*Member, 0)
		// check heartbeat time
		for _, m := range c.members {
			if time.Now().Sub(m.lastHeartbeatAt) > 4*env.Heartbeat && !m.isMaster {
				unregisterMembers = append(unregisterMembers, m)
			}
		}

		for _, m := range unregisterMembers {
			if _, err := c.Unregister(context.Background(), &clusterpb.UnregisterRequest{
				ServiceAddr: m.MemberInfo().ServiceAddr,
			}); err != nil {
				log.Println("Heartbeat unregister error", err)
			}
		}
	}
	go func() {
		ticker := time.NewTicker(env.Heartbeat)
		for {
			select {
			case <-ticker.C:
				if !c.currentNode.IsMaster {
					ticker.Stop()
					return
				}
				check()
			}
		}
	}()
}

func (c *cluster) setRpcClient(client *rpcClient) {
	c.rpcClient = client
}

func (c *cluster) remoteAddrs() []string {
	var addrs []string
	c.mu.RLock()
	for _, m := range c.members {
		addrs = append(addrs, m.memberInfo.ServiceAddr)
	}
	c.mu.RUnlock()
	return addrs
}

func (c *cluster) initMembers(members []*clusterpb.MemberInfo) {
	c.mu.Lock()
	for _, info := range members {
		c.members = append(c.members, &Member{
			memberInfo: info,
		})
	}
	c.mu.Unlock()
}

func (c *cluster) addMember(info *clusterpb.MemberInfo) {
	c.mu.Lock()
	var found bool
	for _, member := range c.members {
		if member.memberInfo.ServiceAddr == info.ServiceAddr {
			member.memberInfo = info
			found = true
			break
		}
	}
	if !found {
		c.members = append(c.members, &Member{
			memberInfo: info,
		})
	}
	c.mu.Unlock()
}

func (c *cluster) delMember(addr string) {
	c.mu.Lock()
	var index = -1
	for i, member := range c.members {
		if member.memberInfo.ServiceAddr == addr {
			index = i
			break
		}
	}
	if index != -1 {
		if index >= len(c.members)-1 {
			c.members = c.members[:index]
		} else {
			c.members = append(c.members[:index], c.members[index+1:]...)
		}
	}
	c.mu.Unlock()
}
