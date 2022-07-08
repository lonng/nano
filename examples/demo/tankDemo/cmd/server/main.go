package main

import (
	"fmt"
	"github.com/cute-angelia/go-utils/components/loggerV3"
	"github.com/cute-angelia/go-utils/utils/conf"
	"github.com/lonng/nano/examples/demo/tankDemo/cmd/server/internal/game"
	"github.com/urfave/cli"
	"log"
	"os"
	"runtime/pprof"
	"sync"
	"time"
)

func main() {
	log.SetFlags(log.Lshortfile)
	// 日志
	loggerV3.New(
		loggerV3.WithProject("tankDemo"),
		loggerV3.WithFileJson(false),
		loggerV3.WithIsOnline(false),
	)

	// app
	app := cli.NewApp()

	// base application info
	app.Name = "tankDemo server"
	app.Author = "Chenyunwen"
	app.Version = "0.0.1"
	app.Copyright = "yunquan game reserved"
	app.Usage = "tankDemo server"

	// flags
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config, c",
			Value: "./config.toml",
			Usage: "load configuration from `FILE`",
		},
		cli.BoolFlag{
			Name:  "cpuprofile",
			Usage: "enable cpu profile",
		},
	}

	app.Action = serve
	app.Run(os.Args)
}

func serve(c *cli.Context) error {
	// 初始化config
	conf.MustLoadConfigFile(c.String("config"))
	conf.MergeConfigWithPath("./")

	if c.Bool("cpuprofile") {
		filename := fmt.Sprintf("cpuprofile-%d.pprof", time.Now().Unix())
		f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, os.ModePerm)
		if err != nil {
			panic(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() { defer wg.Done(); game.Startup() }() // 开启游戏服

	wg.Wait()
	return nil
}
