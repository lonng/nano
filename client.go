package nano

import (
	"context"
	"fmt"
	pb "github.com/jmesyan/nano/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"github.com/jmesyan/nano/internal/message"
	"sync"
	"time"
)

func NewNanoClient(gsid []string, addr string, opts ...Option) *NanoClient {
	return &NanoClient{gsid: gsid, addr: addr, opts: opts}
}

type NanoClient struct {
	sync.Mutex
	gsid   []string
	conn   *grpc.ClientConn
	client pb.GrpcServiceClient
	stream pb.GrpcService_MServiceClient
	ctx    context.Context
	cancel context.CancelFunc
	token  string
	addr   string
	opts   []Option
}

func (this *NanoClient) Cancel() {
	this.cancel()
}

func (this *NanoClient) Done() <-chan struct{} {
	shutdownComponents()
	return this.ctx.Done()
}

func (this *NanoClient) Connect() error {
	this.Lock()
	defer this.Unlock()
	if this.conn != nil {
		this.conn.Close()
	}

	if len(handler.handlers) == 0 {
		for _, opt := range this.opts {
			opt(handler.options)
		}

		startupComponents()

		globalTicker = time.NewTicker(timerPrecision)

		// startup logic dispatcher
		go handler.dispatch()
	}

	ctx, cancel := context.WithCancel(context.Background())
	this.ctx = ctx
	this.cancel = cancel

	conn, err := grpc.DialContext(ctx, this.addr, grpc.WithInsecure())
	if err != nil {
		return err
	}

	client := pb.NewGrpcServiceClient(conn)
	this.conn = conn
	this.client = client
	this.stream = nil
	return nil
}

func (this *NanoClient) GetStream(ReconSingle chan int) pb.GrpcService_MServiceClient {
	this.Lock()
	defer this.Unlock()
	if this.stream != nil {
		return this.stream
	}

	md := metadata.Pairs("gsid", this.gsid[0], "gsid", this.gsid[1], "gsid", this.gsid[2])
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	for {
		stream, err := this.client.MService(ctx)
		if err != nil {
			logger.Println(fmt.Printf("get stream failed. %s", err.Error()))
			time.Sleep(1 * time.Second)
		} else {
			this.stream = stream
			//重连成功，发送桌子注册信息
			ReconSingle <- 1
			return this.stream
		}
	}
	return nil
}

func (this *NanoClient) Start(ReconSingle chan int) {
	this.Connect()
	go func() {
		var (
			reply *pb.GrpcMessage
			err   error
		)
		for {
			reply, err = this.GetStream(ReconSingle).Recv()
			if err != nil && grpc.Code(err) == codes.Unavailable {
				logger.Println("与服务器的连接被断开, 进行重试")
				err = this.Connect()
				if err == nil {
					logger.Println("正在尝试重连")
				} else {
					time.Sleep(2 * time.Second)
				}
				continue
			}
			if err != nil {
				// close(env.die)
				logger.Println("Failed to receive a rpcmessage : %v", err)
				return
			}
			go handler.handleC(this.GetStream(ReconSingle), reply)
		}
	}()

	<-this.Done()
}

func (this *NanoClient) SendMsg(route string, cmd int32, v interface{}){
	if this.stream == nil {
		logger.Println("the client has lose ,the client identity is: %s , route is %s, cmd is %d, the msg is:%v", this.gsid, route, cmd, v)
		return
	}
	payload, err := serializeOrRaw(v)
	if err != nil {
		logger.Println(err.Error())
		return
	}

	if cmd > 0 {
		cmd = cmd | cmdAck
	}
	msg := &pb.GrpcMessage{
		Cid:   0,
		Cmd:   cmd,
		Type:  message.Push,
		Mid:   0,
		Route: route,
		Data:  payload,
	}

	if err := this.stream.SendMsg(msg); err != nil {
		logger.Println(err.Error())
	}
}
