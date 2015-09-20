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

//实现了Module接口的Run方法并提供了:
//1.ChanRPC（用于模块间交互）
//2.Command ChanRPC（用于提供命令服务）
//3.Go(避免操作阻塞当前goroutine)
//4.timer（用于定时器）
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
		case cb := <-s.g.ChanCb: //从Go的回调管道中读取回调函数
			s.g.Cb(cb) //执行回调函数（不用自己写 d.Cb(<-d.ChanCb)了 ）
		case t := <-s.dispatcher.ChanTimer: //从分发器中读取到时定时器
			t.Cb() //执行定时器回调
		}
	}
}

//注册定时器
func (s *Skeleton) AfterFunc(d time.Duration, cb func()) *timer.Timer {
	if s.TimerDispatcherLen == 0 { //判断定时器分发管道长度
		panic("invalid TimerDispatcherLen")
	}

	return s.dispatcher.AfterFunc(d, cb)
}

//注册cron
func (s *Skeleton) CronFunc(expr string, cb func()) (*timer.Cron, error) {
	if s.TimerDispatcherLen == 0 { //判断定时器分发管道长度
		panic("invalid TimerDispatcherLen")
	}

	return s.dispatcher.CronFunc(expr, cb)
}

//一般的go
func (s *Skeleton) Go(f func(), cb func()) {
	if s.GoLen == 0 { //如果Go管道为空
		panic("invalid GoLen") //直接panic
	}

	s.g.Go(f, cb) //调用骨架中创建的g(Go类型)的Go函数
}

//创建线性上下文，再执行线性上下文的Go
func (s *Skeleton) NewLinearContext() *g.LinearContext {
	if s.GoLen == 0 {
		panic("invalid GoLen")
	}

	return s.g.NewLinearContext()
}

//向管道RPC注册函数
func (s *Skeleton) RegisterChanRPC(id interface{}, f interface{}) {
	if s.ChanRPCServer == nil { //外部没有传入RPC服务器
		panic("invalid ChanRPCServer") //抛错
	}

	s.server.Register(id, f) //注册函数f
}

//注册命令
func (s *Skeleton) RegisterCommand(name string, help string, f interface{}) {
	console.Register(name, help, f, s.commandServer) //调用控制台的注册功能
	//实际上是将函数注册进s.commandServer,但是控制台也需要注册命令，以向s.commandServer发起rpc调用
}
