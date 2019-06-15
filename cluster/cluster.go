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

	"github.com/lonng/nano/cluster/clusterpb"
)

// cluster represents a nano cluster, which contains a bunch of nano nodes
// and each of them provide a group of different services. All services requests
// from client will send to gate firstly and be forwarded to appropriate node.
type cluster struct {
	// If cluster is not large enough, use slice is OK
	currentNode *Node
	members     []*Member
	rpcClient   *rpcClient
}

func newCluster(currentNode *Node) *cluster {
	return &cluster{currentNode: currentNode}
}

// Register implements the MasterServer gRPC service
func (c *cluster) Register(_ context.Context, req *clusterpb.RegisterRequest) (*clusterpb.RegisterResponse, error) {
	if req.MemberInfo == nil {
		return nil, ErrInvalidRegisterReq
	}

	resp := &clusterpb.RegisterResponse{}
	for _, m := range c.members {
		if m.memberInfo.ServiceAddr == req.MemberInfo.ServiceAddr {
			return nil, fmt.Errorf("address %s has registered", req.MemberInfo.ServiceAddr)
		}
	}

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

	// Register services to current node
	c.currentNode.handler.addRemoteService(req.MemberInfo)
	c.members = append(c.members, &Member{isMaster: false, memberInfo: req.MemberInfo})
	return resp, nil
}

func (c *cluster) setRpcClient(client *rpcClient) {
	c.rpcClient = client
}
