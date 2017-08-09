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

// SessionFilter represents a filter which was used to filter session when Multicast,
// the session will receive the message while filter returns true.
type SessionFilter func(*session.Session) bool

// Group represents a session group which used to manage a number of
// sessions, data send to the group will send to all session in it.
type Group struct {
	sync.RWMutex
	status   int32                      // channel current status
	name     string                     // channel name
	sessions map[int64]*session.Session // session id map to session instance
}

// NewGroup returns a new group instance
func NewGroup(n string) *Group {
	return &Group{
		status:   groupStatusWorking,
		name:     n,
		sessions: make(map[int64]*session.Session),
	}
}

// Member returns specified UID's session
func (c *Group) Member(uid int64) (*session.Session, error) {
	c.RLock()
	defer c.RUnlock()

	for _, s := range c.sessions {
		if s.Uid() == uid {
			return s, nil
		}
	}

	return nil, ErrMemberNotFound
}

// Members returns all member's UID in current group
func (c *Group) Members() []int64 {
	c.RLock()
	defer c.RUnlock()

	members := []int64{}
	for _, s := range c.sessions {
		members = append(members, s.Uid())
	}

	return members
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

	for _, s := range c.sessions {
		if !filter(s) {
			continue
		}
		if err = s.Push(route, data); err != nil {
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

	for _, s := range c.sessions {
		if err = s.Push(route, data); err != nil {
			log.Println(fmt.Sprintf("Session push message error, ID=%d, Uid=%d, Error=%s", s.ID(), s.Uid(), err.Error()))
		}
	}

	return err
}

// IsContain decide whether a UID is contained in current group
func (c *Group) IsContain(uid int64) bool {
	_, err := c.Member(uid)
	return err == nil
}

// Add add session to group
func (c *Group) Add(session *session.Session) error {
	if c.isClosed() {
		return ErrClosedGroup
	}

	if env.debug {
		log.Println(fmt.Sprintf("Add session to group %s, Uid=%d", c.name, session.Uid()))
	}

	c.Lock()
	defer c.Unlock()

	id := session.ID()
	_, ok := c.sessions[session.ID()]
	if ok {
		return ErrSessionDuplication
	}

	c.sessions[id] = session
	return nil
}

// Leave remove specified UID related session from group
func (c *Group) Leave(s *session.Session) error {
	if c.isClosed() {
		return ErrClosedGroup
	}

	if env.debug {
		log.Println(fmt.Sprintf("Remove session from group %s, Uid=%d", c.name, s.Uid()))
	}

	c.Lock()
	defer c.Unlock()

	delete(c.sessions, s.ID())
	return nil
}

// LeaveAll clear all sessions in the group
func (c *Group) LeaveAll() error {
	if c.isClosed() {
		return ErrClosedGroup
	}

	c.Lock()
	defer c.Unlock()

	c.sessions = make(map[int64]*session.Session)
	return nil
}

// Count get current member amount in the group
func (c *Group) Count() int {
	c.RLock()
	defer c.RUnlock()

	return len(c.sessions)
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
	c.sessions = make(map[int64]*session.Session)
	return nil
}
