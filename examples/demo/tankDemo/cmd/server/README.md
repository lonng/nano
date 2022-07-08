# kcp 版本


```go
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
		nano.WithHandshakeValidator(func(bytes []byte) error {
			log.Println(string(bytes))
			return nil
		}),
		nano.WithSerializer(protobuf.NewSerializer()), // override default serializer
		nano.WithComponents(components),
	)
```