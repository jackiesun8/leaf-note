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
		log.Export(logger)   //替换默认的gLogger
		defer logger.Close() //Run函数返回,关闭logger
	}

	log.Release("Leaf starting up") //关键日志

	// module
	for i := 0; i < len(mods); i++ { //遍历传入的所有module
		module.Register(mods[i]) //注册module
	}
	module.Init() //初始化模块，并执行各个模块(在各个不同的goroutine里)

	// console
	console.Init() //初始化控制台

	// close
	c := make(chan os.Signal, 1)                       //新建一个管道用于接收系统Signal
	signal.Notify(c, os.Interrupt, os.Kill)            //监听SIGINT和SIGKILL信号(linux下叫这个名字)
	sig := <-c                                         //读信号，没有信号时会阻塞goroutine
	log.Release("Leaf closing down (signal: %v)", sig) //关键日志 服务器关闭
	console.Destroy()                                  //销毁控制台
	module.Destroy()                                   //销毁模块
}
