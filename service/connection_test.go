package service

import (
	"testing"
)

const paraCount = 500000

func TestNewConnectionService(t *testing.T) {
	service := newConnectionService()
	w := make(chan bool, paraCount)
	for i := 0; i < paraCount; i++ {
		go func() {
			service.Increment()
			service.SessionID()
			w <- true
		}()
	}

	for i := 0; i < paraCount; i++ {
		<-w
	}

	if service.Count() != paraCount {
		t.Error("wrong connection count")
	}

	if service.SessionID() != paraCount+1 {
		t.Error("wrong session id")
	}
}
