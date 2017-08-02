package nano

import (
	"math/rand"
	"testing"

	"github.com/lonnng/nano/session"
)

func TestChannel_Add(t *testing.T) {
	c := NewGroup("test_add")

	var paraCount = 100
	w := make(chan bool, paraCount)
	for i := 0; i < paraCount; i++ {
		go func(id int) {
			s := &session.Session{}
			s.Bind(int64(id + 1))
			c.Add(s)
			w <- true
		}(i)
	}

	for i := 0; i < paraCount; i++ {
		<-w
	}

	if c.Count() != paraCount {
		t.Fail()
	}

	n := rand.Int63n(int64(paraCount) + 1)
	if !c.IsContain(n) {
		t.Fail()
	}

	// leave
	for i := 0; i < paraCount; i++ {
		go func(id int) {
			c.Leave(int64(id) + 1)
			w <- true
		}(i)
	}

	for i := 0; i < paraCount; i++ {
		<-w
	}

	if c.Count() != 0 {
		t.Fail()
	}
}
