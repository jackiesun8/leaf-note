package g

import (
	"container/list"
	"github.com/name5566/leaf/conf"
	"github.com/name5566/leaf/log"
	"runtime"
	"sync"
)

// 善用 goroutine 能够充分利用多核资源，Leaf 提供的 Go 机制解决了原生 goroutine 存在的一些问题：

// 能够恢复 goroutine 运行过程中的错误（因为处理了异常）
// 游戏服务器会等待所有 goroutine 执行结束后才关闭(Close的时候会处理完所有的待处理回调函数)
// 非常方便的获取 goroutine 执行的结果数据（通过定义一个变量接收结果）
// 在一些特殊场合保证 goroutine 按创建顺序执行（通过线性上下文）

// one Go per goroutine (goroutine not safe)
//每个goroutine里一个Go
//Go类型定义
type Go struct {
	ChanCb    chan func() //回调管道，用于传输回调函数
	pendingGo int         //待处理回调函数计数器
}

//线性Go类型定义
type LinearGo struct {
	f  func() //执行函数
	cb func() //回调函数
}

//线性上下文类型定义
type LinearContext struct {
	g              *Go        //包含一个Go
	linearGo       *list.List //链表
	mutexLinearGo  sync.Mutex //链表互斥锁
	mutexExecution sync.Mutex //执行互斥锁
}

//创建函数
func New(l int) *Go {
	g := new(Go)                    //创建Go
	g.ChanCb = make(chan func(), l) //创建回调管道
	return g
}

//一般的Go函数
//执行一个比较耗时的操作，并在执行完成后，将回调函数通过回调管道发送回原goroutine执行
func (g *Go) Go(f func(), cb func()) {
	g.pendingGo++ //增加待处理回调函数计数器

	go func() { //在一个新的goroutine内执行
		defer func() { //在f执行完后执行
			g.ChanCb <- cb //发送回调函数到回调管道中
			//异常处理
			if r := recover(); r != nil {
				if conf.LenStackBuf > 0 {
					buf := make([]byte, conf.LenStackBuf)
					l := runtime.Stack(buf, false)
					log.Error("%v: %s", r, buf[:l])
				} else {
					log.Error("%v", r)
				}
			}
		}()

		f() //在新的gouroutine执行f(比如一个很耗时的操作)
	}()
}

//执行回调函数
func (g *Go) Cb(cb func()) {
	defer func() {
		g.pendingGo-- //处理完一个，减少待处理回调函数计数器
		//异常处理
		if r := recover(); r != nil {
			if conf.LenStackBuf > 0 {
				buf := make([]byte, conf.LenStackBuf)
				l := runtime.Stack(buf, false)
				log.Error("%v: %s", r, buf[:l])
			} else {
				log.Error("%v", r)
			}
		}
	}()

	if cb != nil { //如果回调函数不为空
		cb() //执行回调函数
	}
}

//关闭Go
func (g *Go) Close() {
	for g.pendingGo > 0 { //如果有待处理的回调函数
		g.Cb(<-g.ChanCb) //从管道中读出来进行执行
	}
}

//创建线性上下文
func (g *Go) NewLinearContext() *LinearContext {
	c := new(LinearContext) //创建一个线性上下文
	c.g = g                 //引用Go实例
	c.linearGo = list.New() //返回一个初始化过的链表
	return c
}

//线性上下文的Go函数
func (c *LinearContext) Go(f func(), cb func()) {
	c.g.pendingGo++ //增加待处理回调函数计数器

	c.mutexLinearGo.Lock()                       //链表加锁
	c.linearGo.PushBack(&LinearGo{f: f, cb: cb}) //向链表添加元素
	c.mutexLinearGo.Unlock()                     //链表解锁

	go func() { //在新的goroutine中执行
		c.mutexExecution.Lock()         //执行加锁，后来的Go将会阻塞在这里，直到该Go执行完成
		defer c.mutexExecution.Unlock() //延迟 执行解锁

		c.mutexLinearGo.Lock()                                 //链表加锁
		e := c.linearGo.Remove(c.linearGo.Front()).(*LinearGo) //从表头移出一个元素
		c.mutexLinearGo.Unlock()                               //链表解锁

		defer func() {
			c.g.ChanCb <- e.cb //当f执行完成后，将回调发送到回调管道中
			//捕获异常
			if r := recover(); r != nil {
				if conf.LenStackBuf > 0 {
					buf := make([]byte, conf.LenStackBuf)
					l := runtime.Stack(buf, false)
					log.Error("%v: %s", r, buf[:l])
				} else {
					log.Error("%v", r)
				}
			}
		}()

		e.f() //执行函数
	}()
}
