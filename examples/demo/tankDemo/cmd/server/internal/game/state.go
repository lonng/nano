package game

import (
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/pipeline"
	"github.com/lonng/nano/scheduler"
	"github.com/lonng/nano/session"
	"time"
)

var defaultStats = NewStats()

type Stats struct {
	component.Base
	timer         *scheduler.Timer
	outboundBytes int
	inboundBytes  int
}

func NewStats() *Stats {
	return &Stats{}
}

func (stats *Stats) Outbound(s *session.Session, msg *pipeline.Message) error {
	stats.outboundBytes += len(msg.Data)
	return nil
}

func (stats *Stats) Inbound(s *session.Session, msg *pipeline.Message) error {
	stats.inboundBytes += len(msg.Data)
	return nil
}

func (stats *Stats) AfterInit() {
	stats.timer = scheduler.NewTimer(time.Minute, func() {
		println("OutboundBytes", stats.outboundBytes)
		println("InboundBytes", stats.outboundBytes)
	})
}
