# 如何构建你的第一个`nano`应用

在这个教程中，我们将构建一个基于浏览器和`WebSocket`的聊天应用。

由于游戏在场景管理、客户端动画等方面有一定的复杂性，并不适合作为`nano`的入门应用。对于大多数开发者
而言，普通聊天室是一个更加适合入门`nano`的应用。

`nano`是一个轻量级的服务器框架，它最适合的应用领域是网页游戏、社交游戏、移动游戏的服务端。当然还不
仅仅是游戏，用`nano`开发高实时web应用也非常合适。

## 前言

- 本教程适用于对`nano`零基础的用户，如果你已经有过一定的`nano`开发基础，请跳过这个教程，你可以阅读
开发指南，那里会对一些话题作较为详细的探讨。

- 由于`nano`是基于Go开发的，因此希望你在阅读本教程前对Go语言有一些了解。

- 本教程的示例源码放在github上[完整代码](https://github.com/lonnng/nano/tree/master/examples/demo/chat)

- 本教程将以一个实时聊天应用为例子，通过对这个应用进行不同的修改来展示`nano`框架的一些功能特性，让用
户能大致了解`nano`，熟悉并能够使用`nano`进行应用程序的开发。

- 本教程假定你使用的开发环境是类Unix系统，如果你使用的Windows系统，希望你能够知道相关的对应方式，比
如一些.sh脚本，在Windows下会使用一个同名的bat文件，本教程中对于Windows系统，不做特殊说明。

## 术语解释

`nano`有一些自己的术语，这里先对术语做一些简单的解释，给读者一个直观的概念，不至于看到相应术语时产生
迷惑。

### 组件(Component)

`nano`应用是由一些松散耦合的`Component`组成的，每个`Component`完成一些功能。整个应用可以看作是一
个`Component`容器，完成`Component`的加载以及生命周期管理。每个`Component`往往有`Init`，`AfterInit`，
`BeforeShutdown`，`Shutdown`等方法，用来完成生命周期管理。
```go
type DemoComponent struct{}

func (c *DemoComponent) Init()           {}
func (c *DemoComponent) AfterInit()      {}
func (c *DemoComponent) BeforeShutdown() {}
func (c *DemoComponent) Shutdown()       {}
```

### Handler

`Handler`用来处理业务逻辑，`Handler`可以有如下形式的签名：
```go
// 以下的Handler会自动将消息反序列化，在调用时当做参数传进来
func (c *DemoComponent) DemoHandler(s *session.Session, payload *pb.DemoPayload) error {
    // 业务逻辑开始
    // ...
    // 业务逻辑结束

    return nil
}

// 以下的Handler不会自动将消息反序列化，会将客户端发送过来的消息直接当作参数传进来
func (c *DemoComponent) DemoHandler(s *session.Session, raw []byte) error {
    // 业务逻辑开始
    // ...
    // 业务逻辑结束

    return nil
}
```

### 路由(Route)

route用来标识一个具体服务或者客户端接受服务端推送消息的位置，对服务端来说，其形式一般是..,例如
"Room.Message", 在我们的示例中, `Room`是一个包含相关`Handler`的组件, `Message`是一个定义在
`Room`中的`Handler`, `Room`中所有符合`Handler`签名的方法都会在`nano`应用启动时自动注册.

对客户端来说，其路由一般形式为onXXX(比如我们示例中的onMessage)，当服务端推送消息时，客户端会
有相应的回调。

### 会话(Session)

`Session`对应于一个客户端会话, 当客户端连接服务器后, 会建立一个会话, 会话在玩家保持连接期间可以
用于保存一些上下文信息, 这些信息会在连接断开后释放.

### 组(Group)

`Group`可以看作是一个`Session`的容器，主要用于需要广播推送消息的场景。可以把某个玩家的`Session`加
入到一个`Group`中，当对这个`Group`推送消息的时候，所有加入到这个`Group`的玩家都会收到推送过来的消
息。一个玩家的`Session`可能会被加入到多个`Group`中，这样玩家就会收到其加入的`Group`推送过来的消息。

### 请求(Request), 响应(Response), 通知(Notify), 推送(Push)

`nano`中有四种消息类型的消息，分别是请求(Request), 响应(Response), 通知(Notify)和推送(Push)，客
户端发起`Request`到服务器端，服务器端处理后会给其返回响应`Response`; `Notify`是客户端发给服务端的
通知，也就是不需要服务端给予回复的请求; `Push`是服务端主动给客户端推送消息的类型。在后面的叙述中，将
会使用这些术语而不再作解释。

## 示例开始

### Server
```go
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/lonnng/nano"
	"github.com/lonnng/nano/component"
	"github.com/lonnng/nano/serialize/json"
	"github.com/lonnng/nano/session"
)

type (
	// define component
	Room struct {
		component.Base
		group *nano.Group
	}

	// protocol messages
	UserMessage struct {
		Name    string `json:"name"`
		Content string `json:"content"`
	}

	NewUser struct {
		Content string `json:"content"`
	}

	AllMembers struct {
		Members []int64 `json:"members"`
	}

	JoinResponse struct {
		Code   int    `json:"code"`
		Result string `json:"result"`
	}
)

func NewRoom() *Room {
	return &Room{
		group: nano.NewGroup("room"),
	}
}

func (r *Room) AfterInit() {
	nano.OnSessionClosed(func(s *session.Session) {
		r.group.Leave(s)
	})
}

// Join room
func (r *Room) Join(s *session.Session, msg []byte) error {
	s.Bind(s.ID()) // binding session uid
	s.Push("onMembers", &AllMembers{Members: r.group.Members()})
	// notify others
	r.group.Broadcast("onNewUser", &NewUser{Content: fmt.Sprintf("New user: %d", s.ID())})
	// new user join group
	r.group.Add(s) // add session to group
	return s.Response(&JoinResponse{Result: "sucess"})
}

// Send message
func (r *Room) Message(s *session.Session, msg *UserMessage) error {
	return r.group.Broadcast("onMessage", msg)
}

func main() {
	nano.Register(NewRoom())
	nano.SetSerializer(json.NewSerializer())
	nano.EnableDebug()
	log.SetFlags(log.LstdFlags | log.Llongfile)

	http.Handle("/web/", http.StripPrefix("/web/", http.FileServer(http.Dir("web"))))

	nano.SetCheckOriginFunc(func(_ *http.Request) bool { return true })
	nano.Listen(":3250", nano.WithIsWebsocket(true))
}
```

1. 首先, 导入这个代码片段需要应用的包
2. 定义`Room`组件
3. 定义所有全后端交互可能用到的协议结构体(实际项目中可能使用Protobuf)
4. 定义所有的Handler, 这里包含`Join`和`Message`
5. 启动我们的应用
   - 注册组件
   - 设置序列化反序列器
   - 开启调试信息
   - 设置log输出信息
   - Set WebSocket check origin function
   - 开始监听`WebSocket`地址":3250"

### Client

参考各个客户端SDK文档

## Summary

这部分, 我们构建了一个简单的聊天应用, 并对代码做了简单的介绍, 通过这个教程, 相信读者对`nano`的工作
流程和工作机制有了一个初步的了解.
