package chanrpc

import (
	"errors"
	"fmt"
	"github.com/name5566/leaf/conf"
	"runtime"
)

// one server per goroutine (goroutine not safe)
// one client per goroutine (goroutine not safe)
//rpc服务器定义
type Server struct {
	// id -> function
	//
	// function:
	// func(args []interface{})
	// func(args []interface{}) interface{}
	// func(args []interface{}) []interface{}
	functions map[interface{}]interface{} //id->func映射
	ChanCall  chan *CallInfo              //管道调用（用于传递调用信息）
}

//调用信息
type CallInfo struct {
	f       interface{}   //函数
	args    []interface{} //参数
	chanRet chan *RetInfo //返回值管道，用于传输返回值
	cb      interface{}   //回调
}

//返回信息
type RetInfo struct {
	// nil，无返回值
	// interface{}，一个任意返回值
	// []interface{}，多个任意返回值
	ret interface{} //返回值
	err error       //错误
	// callback:回调均带error
	// func(err error)，无返回值
	// func(ret interface{}, err error)，一个返回值
	// func(ret []interface{}, err error)，多个返回值
	cb interface{} //回调
}

//rpc客户端定义
type Client struct {
	s               *Server       //rpc服务器引用
	chanSyncRet     chan *RetInfo //同步返回信息
	ChanAsynRet     chan *RetInfo //异步返回信息
	pendingAsynCall int
}

//创建服务器函数
func NewServer(l int) *Server {
	s := new(Server)                                //创建服务器
	s.functions = make(map[interface{}]interface{}) //id->func映射
	s.ChanCall = make(chan *CallInfo, l)            //创建管道，用于传递调用信息
	return s
}

// you must call the function before calling Open and Go
//注册f(函数)
func (s *Server) Register(id interface{}, f interface{}) {
	switch f.(type) { //判断f的类型
	case func([]interface{}): //参数是切片，值任意。无返回值
	case func([]interface{}) interface{}: //参数是切片，值任意。返回值为一个任意值
	case func([]interface{}) []interface{}: //参数是切片，返回值也是切片，值均为任意
	default:
		panic(fmt.Sprintf("function id %v: definition of function is invalid", id)) //id对应的函数定义非法
	}

	if _, ok := s.functions[id]; ok { //判断映射是否存在
		panic(fmt.Sprintf("function id %v: already registered", id))
	}

	s.functions[id] = f //存储映射
}

//返回
func (s *Server) ret(ci *CallInfo, ri *RetInfo) (err error) {
	if ci.chanRet == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	ri.cb = ci.cb
	ci.chanRet <- ri
	return
}

//执行
func (s *Server) Exec(ci *CallInfo) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if conf.LenStackBuf > 0 {
				buf := make([]byte, conf.LenStackBuf)
				l := runtime.Stack(buf, false)
				err = fmt.Errorf("%v: %s", r, buf[:l])
			} else {
				err = fmt.Errorf("%v", r)
			}

			s.ret(ci, &RetInfo{err: fmt.Errorf("%v", r)})
		}
	}()

	// execute
	switch ci.f.(type) {
	case func([]interface{}):
		ci.f.(func([]interface{}))(ci.args)
		return s.ret(ci, &RetInfo{})
	case func([]interface{}) interface{}:
		ret := ci.f.(func([]interface{}) interface{})(ci.args)
		return s.ret(ci, &RetInfo{ret: ret})
	case func([]interface{}) []interface{}:
		ret := ci.f.(func([]interface{}) []interface{})(ci.args)
		return s.ret(ci, &RetInfo{ret: ret})
	}

	panic("bug")
}

// goroutine safe
func (s *Server) Go(id interface{}, args ...interface{}) {
	f := s.functions[id]
	if f == nil {
		return
	}

	defer func() {
		recover()
	}()

	s.ChanCall <- &CallInfo{
		f:    f,
		args: args,
	}
}

//关闭
func (s *Server) Close() {
	close(s.ChanCall)

	for ci := range s.ChanCall {
		s.ret(ci, &RetInfo{
			err: errors.New("chanrpc server closed"),
		})
	}
}

// goroutine safe
//打开
func (s *Server) Open(l int) *Client {
	c := new(Client)                       //创建一个rpc客户端
	c.s = s                                //保存rpc服务器引用
	c.chanSyncRet = make(chan *RetInfo, 1) //创建一个管道用于传输同步返回信息
	c.ChanAsynRet = make(chan *RetInfo, l) //创建一个管道用于传输异步返回信息
	return c                               //返回rpc客户端
}

//发起调用
func (c *Client) call(ci *CallInfo, block bool) (err error) {
	//捕获异常
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	if block { //阻塞的。当管道满时，阻塞
		c.s.ChanCall <- ci //将调用消息通过管道传输到rpc服务器
	} else { //非阻塞的。当管道满时，返回管道已满错误，利用default特性检测chan是否已满
		select {
		case c.s.ChanCall <- ci:
		default:
			err = errors.New("chanrpc channel full")
		}
	}
	return
}

//获取f
func (c *Client) f(id interface{}, n int) (f interface{}, err error) {
	f = c.s.functions[id] //根据id取得对应的f

	//函数f未注册
	if f == nil {
		err = fmt.Errorf("function id %v: function not registered", id)
		return
	}

	var ok bool
	//根据n的值判断f类型是否正确
	switch n {
	case 0:
		_, ok = f.(func([]interface{}))
	case 1:
		_, ok = f.(func([]interface{}) interface{})
	case 2:
		_, ok = f.(func([]interface{}) []interface{})
	default:
		panic("bug")
	}

	//类型不匹配
	if !ok {
		err = fmt.Errorf("function id %v: return type mismatch", id)
	}
	return
}

//调用0
//适合参数是切片，值任意。无返回值
func (c *Client) Call0(id interface{}, args ...interface{}) error {
	//获取f
	f, err := c.f(id, 0)
	if err != nil {
		return err
	}
	//发起调用
	err = c.call(&CallInfo{
		f:       f,
		args:    args,
		chanRet: c.chanSyncRet,
	}, true)
	if err != nil {
		return err
	}
	//读取结果
	ri := <-c.chanSyncRet
	//返回错误字段，代表是否有错
	return ri.err
}

//调用1
//适合参数是切片，值任意。返回值为一个任意值
func (c *Client) Call1(id interface{}, args ...interface{}) (interface{}, error) {
	//读取f
	f, err := c.f(id, 1)
	if err != nil {
		return nil, err
	}
	//发起调用
	err = c.call(&CallInfo{
		f:       f,
		args:    args,
		chanRet: c.chanSyncRet,
	}, true)
	if err != nil {
		return nil, err
	}
	//读取结果
	ri := <-c.chanSyncRet
	//返回返回值字段和错误字段
	return ri.ret, ri.err
}

//调用N
//适合参数是切片，返回值也是切片，值均为任意
func (c *Client) CallN(id interface{}, args ...interface{}) ([]interface{}, error) {
	f, err := c.f(id, 2)
	if err != nil {
		return nil, err
	}

	err = c.call(&CallInfo{
		f:       f,
		args:    args,
		chanRet: c.chanSyncRet,
	}, true)
	if err != nil {
		return nil, err
	}

	ri := <-c.chanSyncRet
	return ri.ret.([]interface{}), ri.err
}

func (c *Client) asynCall(id interface{}, args []interface{}, cb interface{}, n int) error {
	f, err := c.f(id, n)
	if err != nil {
		return err
	}

	err = c.call(&CallInfo{
		f:       f,
		args:    args,
		chanRet: c.ChanAsynRet,
		cb:      cb,
	}, false)
	if err != nil {
		return err
	}

	c.pendingAsynCall++
	return nil
}

func (c *Client) AsynCall(id interface{}, _args ...interface{}) {
	if len(_args) < 1 {
		panic("callback function not found")
	}

	// args
	var args []interface{}
	if len(_args) > 1 {
		args = _args[:len(_args)-1]
	}

	// cb
	cb := _args[len(_args)-1]
	switch cb.(type) {
	case func(error):
		err := c.asynCall(id, args, cb, 0)
		if err != nil {
			cb.(func(error))(err)
		}
	case func(interface{}, error):
		err := c.asynCall(id, args, cb, 1)
		if err != nil {
			cb.(func(interface{}, error))(nil, err)
		}
	case func([]interface{}, error):
		err := c.asynCall(id, args, cb, 2)
		if err != nil {
			cb.(func([]interface{}, error))(nil, err)
		}
	default:
		panic("definition of callback function is invalid")
	}
}

func (c *Client) Cb(ri *RetInfo) {
	switch ri.cb.(type) {
	case func(error):
		ri.cb.(func(error))(ri.err)
	case func(interface{}, error):
		ri.cb.(func(interface{}, error))(ri.ret, ri.err)
	case func([]interface{}, error):
		ri.cb.(func([]interface{}, error))(ri.ret.([]interface{}), ri.err)
	default:
		panic("bug")
	}

	c.pendingAsynCall--
}

func (c *Client) Close() {
	for c.pendingAsynCall > 0 {
		c.Cb(<-c.ChanAsynRet)
	}
}
