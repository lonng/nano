package nano

import (
	"fmt"
	// "github.com/jmesyan/nano/internal/message"
	pb "github.com/jmesyan/nano/protos"
	"github.com/jmesyan/nano/session"
	"sync/atomic"
)

type (
	PipelineFunc func(s *session.Session, msg pb.GrpcMessage) error

	Pipeline interface {
		Outbound() PipelineChannel
		Inbound() PipelineChannel
	}

	pipeline struct {
		outbound, inbound *pipelineChannel
	}

	PipelineChannel interface {
		PushFront(h PipelineFunc)
		PushBack(h PipelineFunc)
		Process(s *session.Session, msg pb.GrpcMessage) error
	}

	pipelineChannel struct {
		handlers []PipelineFunc
	}
)

func NewPipeline() Pipeline {
	return &pipeline{
		outbound: &pipelineChannel{},
		inbound:  &pipelineChannel{},
	}
}

func (p *pipeline) Outbound() PipelineChannel { return p.outbound }
func (p *pipeline) Inbound() PipelineChannel  { return p.inbound }

// PushFront should not be used after nano running
func (p *pipelineChannel) PushFront(h PipelineFunc) {
	if atomic.LoadInt32(&running) > 0 {
		logger.Println("PushFront should not be used after Nano running")
		return
	}
	handlers := make([]PipelineFunc, len(p.handlers)+1)
	handlers[0] = h
	copy(handlers[1:], p.handlers)
	p.handlers = handlers
}

// PushBack should not be used after nano running
func (p *pipelineChannel) PushBack(h PipelineFunc) {
	if atomic.LoadInt32(&running) > 0 {
		logger.Println("PushFront should not be used after Nano running")
		return
	}
	p.handlers = append(p.handlers, h)
}

func (p *pipelineChannel) Process(s *session.Session, msg pb.GrpcMessage) error {
	if len(p.handlers) < 1 {
		return nil
	}
	for _, h := range p.handlers {
		if err := h(s, msg); err != nil {
			logger.Println(fmt.Sprintf("nano/handler: broken pipeline: %s", err.Error()))
			return err
		}
	}
	return nil
}
