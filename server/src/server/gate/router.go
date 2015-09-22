package gate

import (
	"server/login"
	"server/msg"
)

func init() {
	//路由设置
	// login
	msg.JSONProcessor.SetRouter(&msg.C2S_Auth{}, login.ChanRPC)

	// game
}
