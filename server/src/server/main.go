package main

import (
	"github.com/name5566/leaf"
	lconf "github.com/name5566/leaf/conf"
	"server/conf"
	"server/game"
	"server/gate"
	"server/login"
)

func main() {
	lconf.LogLevel = conf.Server.LogLevel //设置日志级别 lconf为leaf框架conf包的别名
	lconf.LogPath = conf.Server.LogPath   //设置日志路径

	leaf.Run( //游戏服务器启动，进行模块的注册
		game.Module,
		gate.Module,
		login.Module,
	)
}
