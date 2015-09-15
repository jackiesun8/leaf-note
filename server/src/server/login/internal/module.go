package internal

import (
	"github.com/name5566/leaf/module"
	"server/base"
)

var (
	skeleton = base.NewSkeleton()     //创建骨架
	ChanRPC  = skeleton.ChanRPCServer //引用骨架中的RPC服务器
)

//模型类型定义
type Module struct {
	*module.Skeleton //匿名组合骨架
}

//初始化
func (m *Module) OnInit() {
	m.Skeleton = skeleton //保存骨架
}

//销毁
func (m *Module) OnDestroy() {

}
