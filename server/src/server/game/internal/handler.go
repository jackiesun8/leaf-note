package internal

import (
	"github.com/name5566/leaf/gate"
	"reflect"
)

func handleMsg(m interface{}, h interface{}) {
	skeleton.RegisterChanRPC(reflect.TypeOf(m), func(args []interface{}) {
		// user
		a := args[1].(gate.Agent)
		user := users[a.UserData().(*AgentInfo).userID]
		if user == nil {
			return
		}

		// agent to user
		args[1] = user
		h.(func([]interface{}))(args)
	})
}

func init() {

}
