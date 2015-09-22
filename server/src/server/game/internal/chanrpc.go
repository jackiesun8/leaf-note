package internal

import (
	"github.com/name5566/leaf/gate"
	"github.com/name5566/leaf/log"
	"server/msg"
)

type AgentInfo struct {
	accID  string //acc:account
	userID int
}

func init() {
	skeleton.RegisterChanRPC("NewAgent", rpcNewAgent)
	skeleton.RegisterChanRPC("CloseAgent", rpcCloseAgent)
	skeleton.RegisterChanRPC("UserLogin", rpcUserLogin)
}

func rpcNewAgent(args []interface{}) {
	a := args[0].(gate.Agent)

	a.SetUserData(new(AgentInfo))
}

func rpcUserLogin(args []interface{}) {
	a := args[0].(gate.Agent)
	accID := args[1].(string)

	// network closed
	if a.UserData() == nil {
		return
	}

	// login repeated
	oldUser := accIDUsers[accID]
	if oldUser != nil {
		m := &msg.S2C_Close{Err: msg.S2C_Close_LoginRepeated}
		a.WriteMsg(m)
		oldUser.WriteMsg(m)
		a.Close()
		oldUser.Close()
		log.Debug("acc %v login repeated", accID)
		return
	}

	log.Debug("acc %v login", accID)

	// login
	newUser := new(User)
	newUser.Agent = a
	newUser.LinearContext = skeleton.NewLinearContext()
	newUser.state = userLogin
	a.UserData().(*AgentInfo).accID = accID
	accIDUsers[accID] = newUser

	newUser.login(accID)
}

func rpcCloseAgent(args []interface{}) {
	a := args[0].(gate.Agent)

	accID := a.UserData().(*AgentInfo).accID
	a.SetUserData(nil)

	user := accIDUsers[accID]
	if user == nil {
		return
	}

	log.Debug("acc %v logout", accID)

	// logout
	if user.state == userLogin {
		user.state = userLogout
	} else {
		user.state = userLogout
		user.logout(accID)
	}
}
