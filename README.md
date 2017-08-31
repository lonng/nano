# Nano [![Build Status][1]][2] [![GoDoc][3]][4] [![Go Report Card][5]][6] [![MIT licensed][7]][8] 

[1]: https://travis-ci.org/lonnng/nano.svg?branch=master
[2]: https://travis-ci.org/lonnng/nano
[3]: https://godoc.org/github.com/lonnng/nano?status.svg
[4]: https://godoc.org/github.com/lonnng/nano
[5]: https://goreportcard.com/badge/github.com/lonnng/nano
[6]: https://goreportcard.com/report/github.com/lonnng/nano
[7]: https://img.shields.io/badge/license-MIT-blue.svg
[8]: LICENSE

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

- English
    + [How to build your first nano application](./docs/get_started.md)
    + [Route compression](./docs/route_compression.md)
    + [Communication protocol](./docs/communication_protocol.md)
    + [Design patterns](./docs/design_patterns.md)
    + [API Reference(Server)](https://godoc.org/github.com/lonnng/nano)
    + [How to integrate `Lua` into `Nano` component(incomplete)](.)
    
- 简体中文
    + [如何构建你的第一个nano应用](./docs/get_started_zh_CN.md)
    + [路由压缩](./docs/route_compression_zh_CN.md)
    + [通信协议](./docs/communication_protocol_zh_CN.md)
    + [API参考(服务器)](https://godoc.org/github.com/lonnng/nano)
    + [如何将`lua`脚本集成到`nano`组件中(未完成)](.)

## Client SDK

- Javascript
  + [nano-websocket-client](https://github.com/lonnng/nano-websocket-client)

## Community

- QQGroup: [289680347](https://jq.qq.com/?_wv=1027&k=4EMMaha)
- Reddit: [nanolabs](https://www.reddit.com/r/nanolabs/)

## Demo

- [Implement a chat room in 100 lines with nano and WebSocket](./examples/demo/chat)
- [Tadpole demo](./examples/demo/tadpole)

## Benchmark

```shell
# Case:   PingPong
# OS:     Windows 10
# Device: i5-6500 3.2GHz 4 Core/1000-Concurrent   => IOPS 11W(Average)
# Other:  ...

cd $GOPATH/src/github.com/lonnng/nano/benchmark/io
go test -v -tags "benchmark"
```

## License

[MIT License](./LICENSE)
