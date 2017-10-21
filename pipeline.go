package nano

import "github.com/lonnng/nano/session"

var Pipeline = struct {
	Outbound, Inbound *pipelineChannel
}{&pipelineChannel{}, &pipelineChannel{}}

type (
	pipelineHandler func(s *session.Session, in []byte) (out []byte, err error)

	pipelineChannel struct {
		handlers []pipelineHandler
	}
)

// PushFront should not be used after nano running
func (p *pipelineChannel) PushFront(h pipelineHandler) {
	handlers := make([]pipelineHandler, len(p.handlers)+1)
	handlers[0] = h
	copy(handlers[1:], p.handlers)
	p.handlers = handlers
}

// PushBack should not be used after nano running
func (p *pipelineChannel) PushBack(h pipelineHandler) {
	p.handlers = append(p.handlers, h)
}
