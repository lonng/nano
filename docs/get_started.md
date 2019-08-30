# How to build your first nano application

In this tutorial, we will build a chat application which based web browser and WebSocket.

Because of the complexity of the game in scene management, client animation, they are
not suitable entry level application for the nano. The chat application is more suitable
as a developer to contact nano's first application and therefore more suitable for the
tutorial.

Nano is really a game server framework, but it is essentially a high real-time, application
framework. In addition to some special parts of the game library in the library section,
the rest of the framework can be used for development of real-time web application.

## Preface

- This tutorial is suitable for beginners, if you have some development experience in nano,
please skip this tutorial. You can read the developer guide, there will be some topics
discussed in detail.

- Since nano is based on Go, so we hope you have some familiarity with Go before reading this
tutorial.

- The tutorial examples' source code is on github, [complete code](https://github.com/lonnng/nano/tree/master/examples/demo/chat)

- This tutorial uses a real-time chat application as an example, and we make some modifications
of the example to show different features of nano, allowing users to have a general understanding
of nano, and be familiar with it and be able to use it for application development.

- This tutorial assumes that your development environment is Unix-like system, if you use
Windows, we hope you know the corresponding manner, such as some .sh script, and uses a bat
file with the same name. This tutorial would not make any special instructions for Windows system.

## Terminologies

Nano has it's own terminology which some may find confusing without a brief explanation. Here
we will try and give readers an overview of some common terms you may come across in this tutorial.

### Component

The nano framework is composed of a number of loosely coupled components and the nano framework
can be regarded as a container of component. Each component defines callbacks: `Init`, `AfterInit`,
`BeforeShutdown`, `Shutdown`.
```go
type DemoComponent struct{}

func (c *DemoComponent) Init()           {}
func (c *DemoComponent) AfterInit()      {}
func (c *DemoComponent) BeforeShutdown() {}
func (c *DemoComponent) Shutdown()       {}
```

### Handler

Handler is used to do business logic, which signature is declared as follows:
```go
// handler that receives unmarshalled data
func (c *DemoComponent) DemoHandler(s *session.Session, payload *pb.DemoPayload) error {
    // business logic begin
    // ...
    // business logic end

    return nil
}

// handler that receives raw data from client
func (c *DemoComponent) DemoHandler(s *session.Session, raw []byte) error {
    // business logic begin
    // ...
    // business logic end

    return nil
}
```

### Route

A "route" is a unique identifier to a specific service endpoint where clients push messages to
your servers, or where clients handle data received from servers. For servers, routes are usually
reached with the following route naming convention: .., such as "Room.Message". In our example,
`Room` is the component that contains a bundle of handler,  `Message` is the handler defined in
`Room` component, all handler methods that defined in component will be registered by nano
automatically.

For the client, its general form will be on[ExpectedEventName] (for our example, onMessage). When
servers push messages, the client will assign a function to handle the incoming data from the
server for display or processing (commonly referred to as a callback).

### Session

Session is used to save the player's context information, which related data will be released
when the player connection was broken.

### Group

Group can be seen as a container of players, it is used in the cases in which broadcasting is
very frequent. When broadcasting to a channel, all the users in the channel will receive the
broadcasting message. A player can be contained by multiple group.

### Request, Response, Notify, Push

There are four types of messages in Nano: request, response, notify and push. Client initiates
request to server, and then server returns a response after handling the request. Notify message
is also sent to server by client, but it does not need a response. Pushing message is sent by
server to client actively.

## Get started

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

1. First of all, we import packages that required in this code snippet.
2. Define room component
3. Define all protocol structure, we use JSON in this tutorial.
4. Define handlers, `Join` and `Message` in this tutorial.
5. Startup our application
   - Register component
   - Set serializer
   - Enable debug information
   - Set log flags
   - Set WebSocket check origin function
   - Listen with ":3250" use WebSocket

### Client

Reference Client SDK documents.

## Summary

In this section, we obtain a simple chat application and make it run, and briefly analyze its
source code.