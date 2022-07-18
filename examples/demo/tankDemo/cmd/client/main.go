package main

import (
	"github.com/cute-angelia/go-utils/components/loggerV3"
	"github.com/cute-angelia/go-utils/syntax/ijson"
	"github.com/cute-angelia/go-utils/utils/conf"
	"github.com/gogo/protobuf/proto"
	"github.com/lonng/nano/examples/demo/tankDemo/cmd/client/internal"
	"github.com/lonng/nano/examples/demo/tankDemo/pb"
	"github.com/lonng/nano/internal/codec"
	"github.com/lonng/nano/internal/packet"
	"github.com/lonng/nano/serialize/protobuf"
	"github.com/spf13/viper"
	"github.com/urfave/cli"
	"github.com/xtaci/kcp-go"
	"log"
	"math/rand"
	"net"
	"os"
	"sync"
	"time"
)

func serve(c *cli.Context) error {
	// 加载日志
	conf.MustLoadConfigFile(c.String("config"))

	serializer := protobuf.NewSerializer()

	// 地址
	var conn net.Conn
	var e error
	if c.Int("net") == 1 {
		conn, e = kcp.Dial(viper.GetString("common.server_addr"))
		if nil != e {
			panic(e)
		}
	} else if c.Int("net") == 3 {
		conn, e = net.Dial("tcp", viper.GetString("common.server_addr"))
		if e != nil {
			panic(e)
		}
	} else {
		panic("网络模式不支持， websocket")
	}

	defer conn.Close()

	log.Println("connecting...")

	connector := internal.NewConnector(conn)
	chReady := make(chan struct{})
	connector.OnConnected(func() {
		log.Println("connected")
		chReady <- struct{}{}
	})
	connector.Start()

	<-chReady
	// 心跳
	go func() {
		heartByte, _ := codec.Encode(packet.Heartbeat, nil)
		d := time.NewTicker(5 * time.Second)
		for {
			select {
			case <-d.C:
				// fmt.Println("The Current time is: ", tm)
				connector.Send(heartByte)
			}
		}
	}()

	// 登陆
	connector.Request("Manager.Login", &pb.Login_Request{
		Uid: c.Int64("uid"),
	}, func(data interface{}) {
		datapb := pb.Login_Response{}
		serializer.Unmarshal(data.([]byte), &datapb)
		log.Println(ijson.Pretty(datapb))
	})

	// 创建房间
	connector.Request("RoomManager.CreateRoom", &pb.CreateRoom_Request{
		RoomId:         c.Uint64("roomId"),
		MaxPlayerCount: uint32(c.Int64("max")),
	}, func(data interface{}) {
		datapb := pb.CreateRoom_Response{}
		serializer.Unmarshal(data.([]byte), &datapb)
		log.Println(ijson.Pretty(datapb))

		if datapb.Error != nil && datapb.Error.Code == 10002 {
			connector.Request("RoomManager.CreateRoom", &pb.JoinRoom_Request{
				RoomId: 1,
			}, func(data interface{}) {
				datapb := pb.JoinRoom_Response{}
				serializer.Unmarshal(data.([]byte), &datapb)
				log.Println(ijson.Pretty(datapb))
			})
		}
	})

	// 接受帧
	connector.On("OnFrameMsgNotify", func(data interface{}) {
		log.Println("OnFrameMsgNotify")
		datapb := pb.FrameMsg_Notify{}
		serializer.Unmarshal(data.([]byte), &datapb)
		log.Println(ijson.Pretty(datapb))
	})
	connector.On("StartGame", func(data interface{}) {
		for i := 10; i > 0; i-- {

			randsid := rand.Intn(1000)
			randx := rand.Intn(1000)
			randy := rand.Intn(1000)
			connector.Notify("RoomManager.OnInput", &pb.Input_Notify{
				Sid: proto.Uint32(uint32(randsid)),
				X:   proto.Float32(float32(randx)),
				Y:   proto.Float32(float32(randy)),
			})
			time.Sleep(time.Second)
		}
	})
	connector.On("OnTest", func(data interface{}) {
		log.Println(string(data.([]byte)))
	})

	//connector.Request("RoomManager.Ready",&pb.Ready_Request{})

	time.Sleep(time.Second * 2)
	connector.Request("RoomManager.Ready", &pb.Ready_Request{}, func(data interface{}) {
		datapb := pb.Ready_Response{}
		serializer.Unmarshal(data.([]byte), &datapb)
		log.Println(ijson.Pretty(datapb))
	})

	//msgJoin := message.New()
	//msgJoin.Route = "room.join"
	//msgJoin.Type = message.Request
	//msgJoin.ID = uint64(pb.ID_MSG_JoinRoom)
	//msgJoin.Data, _ = serialize(&pb.C2S_JoinRoomMsg{
	//	RoomId: proto.Uint64(1),
	//})
	//err := connector.sendMessage(msgJoin)

	// 发送消息
	//type UserMessage struct {
	//	Name    string `json:"name"`
	//	Content string `json:"content"`
	//}
	//userMsg := UserMessage{
	//	Name:    "UserMessage",
	//	Content: "xxxxx",
	//}
	//zz, _ := json.Marshal(userMsg)
	//msgJoin2 := message.New()
	//msgJoin2.Route = "room.message"
	//msgJoin2.Type = message.Request
	//msgJoin2.ID = uint64(time.Now().Unix())
	//msgJoin2.Data = zz
	//connector.sendMessage(msgJoin)

	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
	return nil
}

func main() {

	// logger
	loggerV3.New(loggerV3.WithIsOnline(false), loggerV3.WithProject("chatKcpClient"))

	// app
	app := cli.NewApp()

	app.Name = "tankDemo client"
	app.Author = "Chenyunwen"
	app.Version = "0.0.1"
	app.Copyright = "yunquan game reserved"
	app.Usage = "tankDemo client"

	// flags
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config, c",
			Value: "./config.toml",
			Usage: "load configuration from `FILE`",
		},
		cli.Int64Flag{
			Name:  "uid",
			Value: 1,
			Usage: "uid",
		},
		cli.Uint64Flag{
			Name:  "rid,roomId",
			Value: 1,
			Usage: "roomId",
		},
		cli.Int64Flag{
			Name:  "max",
			Value: 1,
			Usage: "room max",
		},
		cli.IntFlag{
			Name:  "net",
			Value: 1,
			Usage: "网络模式， 1 kcp 2 websocket 3 tcp",
		},
	}

	app.Action = serve
	app.Run(os.Args)

}
