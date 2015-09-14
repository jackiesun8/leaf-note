package base

import (
	"github.com/name5566/leaf/chanrpc"
	"github.com/name5566/leaf/module"
	"server/conf"
)

//创建骨架
func NewSkeleton() *module.Skeleton {
	skeleton := &module.Skeleton{ //创建骨架
		GoLen:              conf.GoLen,
		TimerDispatcherLen: conf.TimerDispatcherLen,
		ChanRPCServer:      chanrpc.NewServer(conf.ChanRPCLen),
	}
	skeleton.Init() //初始化骨架
	return skeleton //返回骨架
}
