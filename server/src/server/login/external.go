package login

import (
	"server/login/internal"
)

var (
	Module  = new(internal.Module) //创建模块
	ChanRPC = internal.ChanRPC     //引用RPC服务器
)
