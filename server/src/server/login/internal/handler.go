package internal

import (
	"github.com/name5566/leaf/gate"
	"reflect"
	"server/game"
	"server/gamedata"
	"server/msg"
)

//处理消息
func handleMsg(m interface{}, h interface{}) {
	skeleton.RegisterChanRPC(reflect.TypeOf(m), h) //注册管道RPC
}

//初始化
func init() {
	handleMsg(&msg.C2S_Auth{}, handleAuth) //注册消息处理函数
}

//消息处理函数，处理Auth
func handleAuth(args []interface{}) {
	m := args[0].(*msg.C2S_Auth)
	a := args[1].(gate.Agent)

	if len(m.AccID) < gamedata.AccIDMin || len(m.AccID) > gamedata.AccIDMax {
		a.WriteMsg(&msg.S2C_Auth{Err: msg.S2C_Auth_AccIDInvalid})
		return
	}

	// login
	game.ChanRPC.Go("UserLogin", a, m.AccID)

	a.WriteMsg(&msg.S2C_Auth{Err: msg.S2C_Auth_OK})
}
