package leaf

import (
	"github.com/name5566/leaf/conf"
	"github.com/name5566/leaf/console"
	"github.com/name5566/leaf/log"
	"github.com/name5566/leaf/module"
	"os"
	"os/signal"
)

func Run(mods ...module.Module) { //...不定参数语法，参数类型都为module.Module
	// logger
	if conf.LogLevel != "" { //日志级别不为空
		logger, err := log.New(conf.LogLevel, conf.LogPath) //创建一个logger
		if err != nil {
			panic(err)
		}
		log.Export(logger)
		defer logger.Close()
	}

	log.Release("Leaf starting up") //关键日志

	// module
	for i := 0; i < len(mods); i++ { //遍历传入的所有module
		module.Register(mods[i]) //注册module
	}
	module.Init()

	// console
	console.Init()

	// close
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	sig := <-c
	log.Release("Leaf closing down (signal: %v)", sig)
	console.Destroy()
	module.Destroy()
}
