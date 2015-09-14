package module

import (
	"github.com/name5566/leaf/chanrpc"
	"github.com/name5566/leaf/console"
	"github.com/name5566/leaf/go" //包名实际为g
	"github.com/name5566/leaf/log"
	"github.com/name5566/leaf/timer"
	"time"
)

//骨架类型定义
type Skeleton struct {
	GoLen              int               //Go管道长度
	TimerDispatcherLen int               //定时器分发器管道长度
	ChanRPCServer      *chanrpc.Server   //RPC服务器引用（外部传入）
	g                  *g.Go             //leaf的Go机制
	dispatcher         *timer.Dispatcher //定时器分发器
	server             *chanrpc.Server   //RPC服务器引用(内部引用)
	commandServer      *chanrpc.Server   //命令RPC服务器引用
}

//初始化
func (s *Skeleton) Init() {
	//检查Go管道长度
	if s.GoLen <= 0 {
		s.GoLen = 0
	}
	//检查定时器分发器管道长度
	if s.TimerDispatcherLen <= 0 {
		s.TimerDispatcherLen = 0
	}

	s.g = g.New(s.GoLen)                                     //创建Go
	s.dispatcher = timer.NewDispatcher(s.TimerDispatcherLen) //创建分发器
	s.server = s.ChanRPCServer                               //外部传入的，内部引用

	if s.server == nil { //外部传入的为空
		s.server = chanrpc.NewServer(0) //内部创建一个
	}
	s.commandServer = chanrpc.NewServer(0) //创建命令RPC服务器
}

//实现了Module接口的Run方法
func (s *Skeleton) Run(closeSig chan bool) {
	for { //死循环
		select {
		case <-closeSig: //读取关闭信号
			s.commandServer.Close() //关闭命令rpc服务器
			s.server.Close()        //关闭rpc服务器
			s.g.Close()             //关闭Go
			return
		case ci := <-s.server.ChanCall: //从rpc服务器读取调用信息
			err := s.server.Exec(ci) //执行调用
			if err != nil {
				log.Error("%v", err)
			}
		case ci := <-s.commandServer.ChanCall: //从命令rpc服务器读取调用信息
			err := s.commandServer.Exec(ci) //执行命令调用
			if err != nil {
				log.Error("%v", err)
			}
		case cb := <-s.g.ChanCb: //从Go中读取回调
			s.g.Cb(cb) //执行回调
		case t := <-s.dispatcher.ChanTimer: //从分发器中读取到时定时器
			t.Cb() //执行定时器回调
		}
	}
}

func (s *Skeleton) AfterFunc(d time.Duration, cb func()) *timer.Timer {
	if s.TimerDispatcherLen == 0 {
		panic("invalid TimerDispatcherLen")
	}

	return s.dispatcher.AfterFunc(d, cb)
}

func (s *Skeleton) CronFunc(expr string, cb func()) (*timer.Cron, error) {
	if s.TimerDispatcherLen == 0 {
		panic("invalid TimerDispatcherLen")
	}

	return s.dispatcher.CronFunc(expr, cb)
}

func (s *Skeleton) Go(f func(), cb func()) {
	if s.GoLen == 0 {
		panic("invalid GoLen")
	}

	s.g.Go(f, cb)
}

func (s *Skeleton) NewLinearContext() *g.LinearContext {
	if s.GoLen == 0 {
		panic("invalid GoLen")
	}

	return s.g.NewLinearContext()
}

//注册管道RPC
func (s *Skeleton) RegisterChanRPC(id interface{}, f interface{}) {
	if s.ChanRPCServer == nil {
		panic("invalid ChanRPCServer")
	}

	s.server.Register(id, f) //注册函数f
}

//注册命令
func (s *Skeleton) RegisterCommand(name string, help string, f interface{}) {
	console.Register(name, help, f, s.commandServer) //调用控制台的注册功能
}
