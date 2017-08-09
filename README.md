# Nano [![Build Status](https://travis-ci.org/lonnng/nano.svg?branch=master)](https://travis-ci.org/lonnng/nano) [![GoDoc](https://godoc.org/github.com/lonnng/nano?status.svg)](https://godoc.org/github.com/lonnng/nano) [![Go Report Card](https://goreportcard.com/badge/github.com/lonnng/nano)](https://goreportcard.com/report/github.com/lonnng/nano)

Nano is an easy to use, fast, lightweight game server networking library for Go.
It provides a core network architecture and a series of tools and libraries that
can help developers eliminate boring duplicate work for common underlying logic.
The goal of nano is to improve development efficiency by eliminating the need to
spend time on repetitious network related programming.

Nano was designed for server-side applications like real-time games, social games,
mobile games, etc of all sizes.

## Installation

```shell
go get github.com/lonnng/nano

# dependencies
go get -u github.com/golang/protobuf
go get -u github.com/gorilla/websocket
```

## Documents

- [How to build your first nano application](./docs/get_started.md)
- [Route compression](./docs/route_compression.md)
- [Communication protocol](./docs/communication_protocol.md)
- [Design patterns](./docs/design_patterns.md)
- [How to integrate Lua into Nano component(incomplete)](.)

## Demo

- [Implement a chat room in 100 lines with nano and WebSocket](./examples/demo/chat)
- [Tadpole demo](./examples/demo/tadpole)

## Benchmark

```shell
# Case:   PingPong
# OS:     Windows 10
# Device: i5-6500 32.GHz 4 Core/1000-Concurrent   => IOPS 11W(Average)
# Other:  ...

cd $GOPATH/src/github.com/lonnng/nano/benchmark/io
go test -v -tags "benchmark"
```

## License

[MIT License](./LICENSE)
