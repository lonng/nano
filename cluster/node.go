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

package cluster

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/lonng/nano/cluster/clusterpb"
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/internal/env"
	"github.com/lonng/nano/internal/log"
	"github.com/lonng/nano/pipeline"
	"google.golang.org/grpc"
)

// Node represents a node in nano cluster, which will contains a group of services.
// All services will register to cluster and messages will be forwarded to the node
// which provides respective service
type Node struct {
	IsMaster       bool
	AdvertiseAddr  string
	MemberAddr     string
	ServerAddr     string
	Components     *component.Components
	IsWebsocket    bool
	TSLCertificate string
	TSLKey         string
	Pipeline       pipeline.Pipeline

	cluster    *cluster
	grpcServer *grpc.Server
	handler    *handler
}

func (n *Node) Startup() error {
	n.cluster = newCluster()
	n.handler = newHandler(n.Pipeline)

	components := n.Components.List()
	for _, c := range components {
		err := n.handler.register(c.Comp, c.Opts)
		if err != nil {
			return err
		}
	}

	cache()

	// Bootstrap cluster if either current is master or advertise address is not empty
	// - Current node is master
	// - Current node is cluster member
	if n.IsMaster {
		if n.AdvertiseAddr == "" {
			return errors.New("advertise address cannot be empty in master node")
		}
		listener, err := net.Listen("tcp", n.AdvertiseAddr)
		if err != nil {
			return err
		}
		n.grpcServer = grpc.NewServer()
		clusterpb.RegisterClusterServer(n.grpcServer, n.cluster)
		go func() {
			err := n.grpcServer.Serve(listener)
			if err != nil {
				log.Println("Start master node failed: " + err.Error())
			}
		}()
	} else if n.AdvertiseAddr != "" {
		// TODO: connect to master node
	}

	// Initialize all components
	for _, c := range components {
		c.Comp.Init()
	}
	for _, c := range components {
		c.Comp.AfterInit()
	}

	go func() {
		if n.IsWebsocket {
			if len(n.TSLCertificate) != 0 {
				n.listenAndServeWSTLS()
			} else {
				n.listenAndServeWS()
			}
		} else {
			n.listenAndServe()
		}
	}()
	return nil
}

// Shutdowns all components registered by application, that
// call by reverse order against register
func (n *Node) Shutdown() {
	// reverse call `BeforeShutdown` hooks
	components := n.Components.List()
	length := len(components)
	for i := length - 1; i >= 0; i-- {
		components[i].Comp.BeforeShutdown()
	}

	// reverse call `Shutdown` hooks
	for i := length - 1; i >= 0; i-- {
		components[i].Comp.Shutdown()
	}

	if n.grpcServer != nil {
		n.grpcServer.GracefulStop()
	}
}

// Enable current server accept connection
func (n *Node) listenAndServe() {
	listener, err := net.Listen("tcp", n.ServerAddr)
	if err != nil {
		log.Fatal(err.Error())
	}

	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err.Error())
			continue
		}

		go n.handler.handle(conn)
	}
}

func (n *Node) listenAndServeWS() {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     env.CheckOrigin,
	}

	http.HandleFunc("/"+strings.TrimPrefix(env.WSPath, "/"), func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(fmt.Sprintf("Upgrade failure, URI=%s, Error=%s", r.RequestURI, err.Error()))
			return
		}

		n.handler.handleWS(conn)
	})

	if err := http.ListenAndServe(n.ServerAddr, nil); err != nil {
		log.Fatal(err.Error())
	}
}

func (n *Node) listenAndServeWSTLS() {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     env.CheckOrigin,
	}

	http.HandleFunc("/"+strings.TrimPrefix(env.WSPath, "/"), func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(fmt.Sprintf("Upgrade failure, URI=%s, Error=%s", r.RequestURI, err.Error()))
			return
		}

		n.handler.handleWS(conn)
	})

	if err := http.ListenAndServeTLS(n.ServerAddr, n.TSLCertificate, n.TSLKey, nil); err != nil {
		log.Fatal(err.Error())
	}
}
