package tank

import (
	"github.com/cute-angelia/go-utils/components/loggerV3"
	"github.com/lonng/nano/examples/demo/tankDemo/cmd/server/internal/common"
	"github.com/lonng/nano/examples/demo/tankDemo/pb"
	"github.com/lonng/nano/session"
	"google.golang.org/protobuf/proto"
	"time"
)

const (
	Frequency = 15
	TickTimer = time.Second / Frequency // 帧率
	TimeOut   = time.Minute * 10        // 30分钟超时

	BroadcastOffsetFrames = 3 // 每隔多少帧广播一次
	kMaxFrameDataPerMsg   = 4 // 每个消息包最多包含多少个帧数据
)

type tank struct {
	common.GameInterface
	gameStatus common.GameStatus
	logic      *lockstep

	players map[int64]*session.Session // 玩家

	forceSendFrame   bool             // 强制发送帧
	frameCountClient uint32           // 客户端最新帧
	frameCountPlayer map[int64]uint32 // 用户帧

	hp map[int64]int // 血量
}

func NewTank() *tank {
	return &tank{
		players:          make(map[int64]*session.Session),
		hp:               make(map[int64]int),
		logic:            newLockstep(),
		frameCountClient: 0,
		frameCountPlayer: make(map[int64]uint32),
		forceSendFrame:   true,
	}
}

func (tk *tank) getPlayerFrameCount(uid int64) uint32 {
	if v, ok := tk.frameCountPlayer[uid]; ok {
		return v
	} else {
		return 0
	}
}

func (tk *tank) setPlayerFrameCount(uid int64, frameCount uint32) {
	tk.frameCountPlayer[uid] = frameCount
}

func (tk *tank) OnJoinGame(s *session.Session) {
	tk.gameStatus = common.GameStatusReady
	tk.players[s.UID()] = s
	tk.frameCountPlayer[s.UID()] = 0

	s.Push("OnTest", []byte("good == == => 222222222"))
}

func (tk *tank) OnGameStart() {
	tk.gameStatus = common.GameStatusGameIng
	go func() {
		tk.Run()
	}()
}

func (tk *tank) OnLeaveGame() {
}

func (tk *tank) OnGameOver() {
	tk.gameStatus = common.GameStatusOver
}

func (tk *tank) OnGameLockStepPush(data *pb.Input_Notify) {
	tk.logic.pushCmd(data)
}

func (tk *tank) ReduceHp(uid int64, hp int) int {
	if userHp, ok := tk.hp[uid]; ok {
		if userHp > hp {
			tk.hp[uid] = userHp - hp
		} else {
			tk.hp[uid] = 0
		}
		return tk.hp[uid]
	} else {
		return 0
	}
}

// Run 主循环
func (tk *tank) Run() {
	// 心跳
	tickerTick := time.NewTicker(TickTimer)
	defer tickerTick.Stop()

	// 超时timer
	timeoutTimer := time.NewTimer(TimeOut)

LOOP:
	for {
		select {
		case <-timeoutTimer.C:
			loggerV3.GetLogger().Error().Msgf("[lockstep] time out")
			tk.gameStatus = common.GameStatusOver

			break LOOP
		case <-tickerTick.C:
			if !tk.Tick(time.Now().Unix()) {
				loggerV3.GetLogger().Info().Msgf("[room(%d)] tick over")
				break LOOP
			}
		}
	}

	for i := 3; i > 0; i-- {
		<-time.After(time.Second)
		loggerV3.GetLogger().Info().Msgf("[room(%d)] quiting %d...", i)
	}
}

func (tk *tank) Tick(now int64) bool {
	switch tk.gameStatus {
	case common.GameStatusReady:
		return true
	case common.GameStatusGameIng:
		if tk.checkIsOver() {
			tk.gameStatus = common.GameStatusOver
			loggerV3.GetLogger().Info().Msgf("[game] game over successfully!!")
			return true
		}

		// 发帧
		tk.logic.tick()
		tk.broadcastFrameData()

		return true
	case common.GameStatusOver:
		//g.doGameOver()
		//g.State = k_Stop
		//loggerV3.GetLogger().Info().Msgf("[game(%d)] do game over", g.roomID)
		return true
	case common.GameStatusStop:
		return false
	}

	return false
}

//
func (tk *tank) checkIsOver() bool {
	for _, hp := range tk.hp {
		if hp <= 0 {
			return true
		}
	}
	return false
}

func (tk *tank) broadcastFrameData() {
	framesCount := tk.logic.getFrameCount()
	if !tk.forceSendFrame && framesCount-tk.frameCountClient < BroadcastOffsetFrames {
		return
	}

	defer func() {
		tk.forceSendFrame = false
		tk.frameCountClient = framesCount
	}()

	// now := time.Now().Unix()
	for _, p := range tk.players {

		// 掉线的
		if p == nil || p.ID() == 0 {
			continue
		}

		//if err := p.Push("OnTest", []byte("good == =33333333= =>")); err != nil {
		//	log.Println("33333333", err)
		//}

		// 获得这个玩家已经发到哪一帧
		i := tk.getPlayerFrameCount(p.UID())
		c := 0
		msg := &pb.FrameMsg_Notify{}

		for ; i < framesCount; i++ {
			frameData := tk.logic.getFrame(i)
			if nil == frameData && i != (framesCount-1) {
				continue
			}

			f := &pb.FrameData{
				FrameID: proto.Uint32(i),
			}

			if nil != frameData {
				f.Input = frameData.cmds

				//if p.id == 2 && nil != frameData {
				//	log.Println("玩家当前帧:", p.id, "=", i, "/")
				//	for _, cmd := range frameData.cmds {
				//		log.Printf("玩家当前帧--------: %d , %d, %f, %f", *cmd.PlayerID, *cmd.Sid, *cmd.X, *cmd.Y)
				//	}
				//}

			}
			msg.Frames = append(msg.Frames, f)
			c++

			// 如果是最后一帧或者达到这个消息包能装下的最大帧数，就发送
			if i == (framesCount-1) || c >= kMaxFrameDataPerMsg {
				if err := p.Push("OnFrameMsgNotify", msg); err != nil {
					loggerV3.GetLogger().Err(err).Send()
				}
				c = 0
				msg = &pb.FrameMsg_Notify{}
			}

		}
		tk.setPlayerFrameCount(p.UID(), framesCount)
	}
}
