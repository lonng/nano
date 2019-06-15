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

package mock_test

import (
	"testing"

	"github.com/lonng/nano/mock"
	. "github.com/pingcap/check"
)

type networkEntitySuite struct{}

func TestNetworkEntity(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&networkEntitySuite{})

func (s *networkEntitySuite) TestNetworkEntity(c *C) {
	entity := mock.NewNetworkEntity()

	c.Assert(entity.LastResponse(), IsNil)
	c.Assert(entity.LastMid(), Equals, uint64(1))
	c.Assert(entity.Response("hello"), IsNil)
	c.Assert(entity.LastResponse().(string), Equals, "hello")

	c.Assert(entity.FindResponseByMID(1), IsNil)
	c.Assert(entity.ResponseMid(1, "test"), IsNil)
	c.Assert(entity.FindResponseByMID(1).(string), Equals, "test")

	c.Assert(entity.FindResponseByRoute("t.tt"), IsNil)
	c.Assert(entity.Push("t.tt", "test"), IsNil)
	c.Assert(entity.FindResponseByRoute("t.tt").(string), Equals, "test")

	c.Assert(entity.RemoteAddr().String(), Equals, "mock-addr")
	c.Assert(entity.Close(), IsNil)
}
