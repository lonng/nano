package main

import (
	"github.com/lonng/nano"
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/examples/demo/chatKcp/cmd/server/logic"
	"github.com/lonng/nano/pipeline"
	"github.com/lonng/nano/serialize/protobuf"
	"strings"
)

func main() {
	components := &component.Components{}
	components.Register(
		logic.NewRoomManager(),
		component.WithName("room"), // rewrite component and handler name
		component.WithNameFunc(strings.ToLower),
	)

	// traffic stats
	pip := pipeline.New()
	var stats = logic.NewStats()
	pip.Outbound().PushBack(stats.Outbound)
	pip.Inbound().PushBack(stats.Inbound)

	nano.Listen(":3251",
		nano.WithIsKcpSocket(true), // kcp 服务
		nano.WithPipeline(pip),
		nano.WithDebugMode(),
		nano.WithSerializer(protobuf.NewSerializer()), // override default serializer
		nano.WithComponents(components),
	)
}
