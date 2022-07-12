package game

import (
	"fmt"
	"github.com/cute-angelia/go-utils/components/loggerV3"
	"github.com/lonng/nano/examples/demo/tankDemo/cmd/server/internal/logger"
	"math/rand"
	"time"

	"github.com/lonng/nano"
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/serialize/json"
	"github.com/spf13/viper"
)

// Startup 初始化游戏服务器
func Startup() {
	rand.Seed(time.Now().Unix())

	heartbeat := viper.GetInt("core.heartbeat")
	if heartbeat < 5 {
		heartbeat = 5
	}

	loggerV3.GetLogger().Info().Str("component", "game").Msgf("当前游戏服务器版本: %s, 是否强制更新: %t, 当前心跳时间间隔: %d秒", viper.GetString("update.version"), viper.GetBool("update.force"), heartbeat)

	// register game handler
	comps := &component.Components{}
	comps.Register(defaultManager)
	comps.Register(defaultRoomManager)
	//comps.Register(defaultNewTest)
	comps.Register(defaultStats)

	// 加密管道
	//c := newCrypto()
	//pip := pipeline.New()
	//pip.Inbound().PushBack(c.inbound)
	//pip.Outbound().PushBack(c.outbound)

	addr := fmt.Sprintf(":%d", viper.GetInt("game-server.port"))
	nano.Listen(addr,
		nano.WithIsKcpSocket(true),
		// nano.WithPipeline(pip),
		nano.WithHeartbeatInterval(time.Duration(heartbeat)*time.Second),
		nano.WithLogger(logger.NewLogger()),
		nano.WithSerializer(json.NewSerializer()),
		nano.WithComponents(comps),
	)
}
