package gate

import (
	"server/login"
	"server/msg"
)

func init() {
	// login
	msg.JSONProcessor.SetRouter(&msg.C2S_Auth{}, login.ChanRPC)

	// game
}
