package login

import (
	"server/login/internal"
)

var (
	Module  = new(internal.Module) //创建并导出登录模块
	ChanRPC = internal.ChanRPC     //引用RPC服务器并导出
)
