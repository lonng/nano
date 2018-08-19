// Copyright (c) nano Author. All Rights Reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package nano

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/websocket"
	pb "github.com/jmesyan/nano/protos"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"io"
	"strings"
	"sync/atomic"
	"time"
)

var running int32

func connect(addr string, opts ...Option) {
	reconnect.addr = addr
	reconnect.opts = opts

	// mark application running
	if atomic.AddInt32(&running, 1) != 1 {
		logger.Println("Nano has running")
		return
	}

	for _, opt := range opts {
		opt(handler.options)
	}

	// cache heartbeat data
	hbdEncode()

	// initial all components
	startupComponents()

	// create global ticker instance, timer precision could be customized
	// by SetTimerPrecision
	globalTicker = time.NewTicker(timerPrecision)
	// startup logic dispatcher
	go handler.dispatch()

	go func() {
		connectAndServe(addr)
	}()

	logger.Println(fmt.Sprintf("starting application %s, connect to %s", app.name, reconnect.addr))
	sg := make(chan os.Signal)
	signal.Notify(sg, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL)

	// stop server
	select {
	case <-env.die:
		logger.Println("The app will shutdown in a few seconds")
	case s := <-sg:
		logger.Println("got signal", s)
	}

	logger.Println("server is stopping...")

	// shutdown all components registered by application, that
	// call by reverse order against register
	shutdownComponents()
	atomic.StoreInt32(&running, 0)
}

func connectAndServe(addr string) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		logger.Fatal("fail to dial: %v", err)
	}
	defer conn.Close()
	client := pb.NewGrpcServiceClient(conn)
	rpcService(client)
}

func rpcService(client pb.GrpcServiceClient) {
	md := metadata.Pairs("gid", "1001")
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	stream, err := client.MService(ctx)
	if err != nil {
		logger.Fatal("%v.MService(_) = _, %v", client, err)
	}
	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				close(env.die)
				logger.Println("Read message error: %v, application will be closed immediately", err)
				return
			}
			if err != nil {
				close(env.die)
				logger.Println("Failed to receive a rpcmessage : %v", err)
				return
			}
			go handler.handleC(stream, in)
			// userMessage := &pb.UserMessage{}
			// err = proto.Unmarshal(in.Data, userMessage)
			// if err != nil {
			// 	logger.Fatal("unmarshaling error: ", err)
			// }
			// logger.Println(in, userMessage)
		}
	}()

	sg := make(chan os.Signal)
	signal.Notify(sg, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL)
	select {
	case <-env.die:
		logger.Println("The app will shutdown in a few seconds2")
	case s := <-sg:
		logger.Println("got signal2", s)
	}
	stream.CloseSend()
}

func listen(addr string, isWs bool, opts ...Option) {
	// mark application running
	if atomic.AddInt32(&running, 1) != 1 {
		logger.Println("Nano has running")
		return
	}

	for _, opt := range opts {
		opt(handler.options)
	}

	// cache heartbeat data
	hbdEncode()

	// initial all components
	startupComponents()

	// create global ticker instance, timer precision could be customized
	// by SetTimerPrecision
	globalTicker = time.NewTicker(timerPrecision)

	// startup logic dispatcher
	go handler.dispatch()

	go func() {
		if isWs {
			listenAndServeWS(addr)
		} else {
			listenAndServe(addr)
		}
	}()

	logger.Println(fmt.Sprintf("starting application %s, listen at %s", app.name, addr))
	sg := make(chan os.Signal)
	signal.Notify(sg, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL)

	// stop server
	select {
	case <-env.die:
		logger.Println("The app will shutdown in a few seconds")
	case s := <-sg:
		logger.Println("got signal", s)
	}

	logger.Println("server is stopping...")

	// shutdown all components registered by application, that
	// call by reverse order against register
	shutdownComponents()
	atomic.StoreInt32(&running, 0)
}

// Enable current server accept connection
func listenAndServe(addr string) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Fatal(err.Error())
	}

	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Println(err.Error())
			continue
		}

		go handler.handle(conn)
	}
}

func listenAndServeWS(addr string) {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     env.checkOrigin,
	}

	http.HandleFunc("/"+strings.TrimPrefix(env.wsPath, "/"), func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Println(fmt.Sprintf("Upgrade failure, URI=%s, Error=%s", r.RequestURI, err.Error()))
			return
		}

		handler.handleWS(conn)
	})

	if err := http.ListenAndServe(addr, nil); err != nil {
		logger.Fatal(err.Error())
	}
}
