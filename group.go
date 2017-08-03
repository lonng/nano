// Copyright (c) nano Author. All Rights Reserved.
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

package nano

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/lonnng/nano/session"
)

const (
	groupStatusWorking = 0
	groupStatusClosed  = 1
)

type SessionFilter func(*session.Session) bool

var (
	ErrCloseClosedGroup = errors.New("close closed group")
	ErrClosedGroup      = errors.New("group closed")
	ErrMemberNotFound   = errors.New("member not found in the group")
)

// Group represents a session group which used to manage a number of
// sessions, data send to the group will send to all session in it.
type Group struct {
	sync.RWMutex
	status  int32
	name    string                     // channel name
	uids    map[int64]*session.Session // uid map to session pointer
	members []int64                    // all user ids
}

func NewGroup(n string) *Group {
	return &Group{
		status: groupStatusWorking,
		name:   n,
		uids:   make(map[int64]*session.Session),
	}
}

func (c *Group) Member(uid int64) *session.Session {
	c.RLock()
	defer c.RUnlock()

	return c.uids[uid]
}

func (c *Group) Members() []int64 {
	c.RLock()
	defer c.RUnlock()

	return c.members
}

// Push message to partial client, which filter return true
func (c *Group) Multicast(route string, v interface{}, filter SessionFilter) error {
	if c.isClosed() {
		return ErrClosedGroup
	}

	data, err := serializeOrRaw(v)
	if err != nil {
		return err
	}

	if env.debug {
		log.Println(fmt.Sprintf("Type=Multicast Route=%s, Data=%+v", route, v))
	}

	c.RLock()
	defer c.RUnlock()

	for _, s := range c.uids {
		if !filter(s) {
			continue
		}
		err = s.Push(route, data)
		if err != nil {
			log.Println(err.Error())
		}
	}

	return nil
}

// Push message to all client
func (c *Group) Broadcast(route string, v interface{}) error {
	if c.isClosed() {
		return ErrClosedGroup
	}

	data, err := serializeOrRaw(v)
	if err != nil {
		return err
	}

	if env.debug {
		log.Println(fmt.Sprintf("Type=Broadcast Route=%s, Data=%+v", route, v))
	}

	c.RLock()
	defer c.RUnlock()

	for _, s := range c.uids {
		err = s.Push(route, data)
		if err != nil {
			log.Println(err.Error())
		}
	}

	return err
}

func (c *Group) IsContain(uid int64) bool {
	c.RLock()
	defer c.RUnlock()

	if _, ok := c.uids[uid]; ok {
		return true
	}

	return false
}

func (c *Group) Add(session *session.Session) error {
	if c.isClosed() {
		return ErrClosedGroup
	}

	c.Lock()
	defer c.Unlock()

	c.uids[session.Uid] = session
	c.members = append(c.members, session.Uid)

	return nil
}

func (c *Group) Leave(uid int64) error {
	if c.isClosed() {
		return ErrClosedGroup
	}

	if !c.IsContain(uid) {
		return ErrMemberNotFound
	}

	c.Lock()
	defer c.Unlock()

	var temp []int64
	for i, u := range c.members {
		if u == uid {
			temp = append(temp, c.members[:i]...)
			c.members = append(temp, c.members[(i+1):]...)
			break
		}
	}
	delete(c.uids, uid)

	return nil
}

func (c *Group) LeaveAll() error {
	if atomic.LoadInt32(&c.status) == groupStatusClosed {
		return ErrClosedGroup
	}

	c.Lock()
	defer c.Unlock()

	c.uids = make(map[int64]*session.Session)
	c.members = make([]int64, 0)

	return nil
}

// Count get current member amount in the group
func (c *Group) Count() int {
	c.RLock()
	defer c.RUnlock()

	return len(c.uids)
}

func (c *Group) isClosed() bool {
	if atomic.LoadInt32(&c.status) == groupStatusClosed {
		return true
	}
	return false
}

// Close destroy group, which will release all resource in the group
func (c *Group) Close() error {
	if c.isClosed() {
		return ErrCloseClosedGroup
	}

	atomic.StoreInt32(&c.status, groupStatusClosed)

	// release all reference
	c.uids = make(map[int64]*session.Session)
	c.members = []int64{}

	return nil
}
