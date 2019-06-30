# Nano [![Build Status][1]][2] [![GoDoc][3]][4] [![Go Report Card][5]][6] [![MIT licensed][7]][8] 

[1]: https://travis-ci.org/lonng/nano.svg?branch=master
[2]: https://travis-ci.org/lonng/nano
[3]: https://godoc.org/github.com/lonng/nano?status.svg
[4]: https://godoc.org/github.com/lonng/nano
[5]: https://goreportcard.com/badge/github.com/lonng/nano
[6]: https://goreportcard.com/report/github.com/lonng/nano
[7]: https://img.shields.io/badge/license-MIT-blue.svg
[8]: LICENSE

Nano is an easy to use, fast, lightweight game server networking library for Go.
It provides a core network architecture and a series of tools and libraries that
can help developers eliminate boring duplicate work for common underlying logic.
The goal of nano is to improve development efficiency by eliminating the need to
spend time on repetitious network related programming.

Nano was designed for server-side applications like real-time games, social games,
mobile games, etc of all sizes.

## How to build a system with `Nano`

#### What does a `Nano` application look like?

The simplest "nano" application as shown in the following figure, you can make powerful applications by combining different components.

![Application](media/application.png)

In fact, the `nano` application is a collection of  [Component ](./docs/get_started.md#component) , and a component is a bundle of  [Handler](./docs/get_started.md#handler), once you register a component to nano, nano will register all methods that can be converted to `Handler` to nano service container. Service was accessed by `Component.Handler`, and the handler will be called while client request. The handler will receive two parameters while handling a message:
  - `*session.Session`: corresponding a client that apply this request or notify.
  - `*protocol.FooBar`: the payload of the request.

While you had processed your logic, you can response or push message to the client by `session.Response(payload)` and `session.Push('eventName', payload)`, or returns error when some unexpected data received.

#### How to build distributed system with `Nano`

Nano contains built-in distributed system solution, and make you creating a distributed game server easily.

See: [The distributed chat demo](https://github.com/lonng/nano/tree/master/examples/cluster)

The Nano will remain simple, but you can perform any operations in the component and get the desired goals. You can startup a group of `Nano` application as agent to dispatch message to backend servers.

#### How to execute the asynchronous task

```go
func (manager *PlayerManager) Login(s *session.Session, msg *ReqPlayerLogin) error {
    var onDBResult = func(player *Player) {
        manager.players = append(manager.players, player)
        s.Push("PlayerSystem.LoginSuccess", &ResPlayerLogin)
    }
    
    // run slow task in new gorontine
    go func() {
        player, err := db.QueryPlayer(msg.PlayerId) // ignore error in demo
        // handle result in main logical gorontine
        nano.Invoke(func(){ onDBResult(player) })
    }
    return nil
}
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

## Resources

- Javascript
  + [nano-websocket-client](https://github.com/lonnng/nano-websocket-client)
  + [nano-egret-client](https://github.com/lonnng/nano-egret-client)

- Demo
  + [Implement a chat room in 100 lines with nano and WebSocket](./examples/demo/chat)
  + [Tadpole demo](./examples/demo/tadpole)
  + [四川麻将, 血战到底(三人模式/四人模式)完整项目实例](https://github.com/lonnng/nanoserver)

## Community

- QQGroup: [289680347](https://jq.qq.com/?_wv=1027&k=4EMMaha)
- Reddit: [nanolabs](https://www.reddit.com/r/nanolabs/)

## Successful cases

- [空来血战](https://fir.im/tios)

## Go version

`> go1.8`

## Installation

```shell
go get github.com/lonng/nano

# dependencies
go get -u github.com/golang/protobuf
go get -u github.com/gorilla/websocket
```

## Benchmark

```shell
# Case:   PingPong
# OS:     Windows 10
# Device: i5-6500 3.2GHz 4 Core/1000-Concurrent   => IOPS 11W(Average)
# Other:  ...

cd $GOPATH/src/github.com/lonng/nano/benchmark/io
go test -v -tags "benchmark"
```

## License

[MIT License](./LICENSE)
