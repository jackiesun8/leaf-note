package chanrpc_test

import (
	"fmt"
	"github.com/name5566/leaf/chanrpc"
	"sync"
)

func Example() {
	s := chanrpc.NewServer(10) //创建一个RPC服务器,调用管道的长度为10
	//调用管道：用于传输调用信息
	//一个rpc服务器可以对应多个rpc客户端，rpc客户端将调用信息传输到RPC服务器的管道（共用一个），rpc服务器再将响应消息返回到rpc客户端各自的管道内
	var wg sync.WaitGroup //声明等待组
	wg.Add(1)             //等待组+1

	// goroutine 1
	go func() {
		//注册函数:id->function
		s.Register("f0", func(args []interface{}) {

		})

		s.Register("f1", func(args []interface{}) interface{} {
			return 1
		})

		s.Register("fn", func(args []interface{}) []interface{} {
			return []interface{}{1, 2, 3}
		})

		s.Register("add", func(args []interface{}) interface{} {
			n1 := args[0].(int)
			n2 := args[1].(int)
			return n1 + n2
		})
		//注册完成，释放等待组
		wg.Done()

		//死循环，处理rpc服务器调用
		for {
			err := s.Exec(<-s.ChanCall) //从管道里读取一个调用信息并执行
			if err != nil {
				fmt.Println(err)
			}
		}
	}()

	wg.Wait() //等待等待组，阻塞在这里，等待注册函数完成
	wg.Add(1) //注册函数完成，等待组再+1

	// goroutine 2
	go func() {
		c := s.Open(10) //打开一个rpc客户端

		// sync 同步调用
		//同步调用0，无返回值（0个返回值）
		err := c.Call0("f0")
		if err != nil {
			fmt.Println(err)
		}
		//同步调用1，1个返回值
		r1, err := c.Call1("f1")
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(r1)
		}
		//同步调用N，N个返回值
		rn, err := c.CallN("fn")
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(rn[0], rn[1], rn[2])
		}
		//同步调用1，1个返回值
		ra, err := c.Call1("add", 1, 2)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(ra)
		}

		// asyn 异步调用
		c.AsynCall("f0", func(err error) {
			if err != nil {
				fmt.Println(err)
			}
		})

		c.AsynCall("f1", func(ret interface{}, err error) {
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(ret)
			}
		})

		c.AsynCall("fn", func(ret []interface{}, err error) {
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(ret[0], ret[1], ret[2])
			}
		})

		c.AsynCall("add", 1, 2, func(ret interface{}, err error) {
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(ret)
			}
		})

		c.Cb(<-c.ChanAsynRet)
		c.Cb(<-c.ChanAsynRet)
		c.Cb(<-c.ChanAsynRet)
		c.Cb(<-c.ChanAsynRet)

		// go Go调用
		s.Go("f0")

		wg.Done() //goroutine2完成
	}()

	wg.Wait() //等待goroutine2完成

	// Output:
	// 1
	// 1 2 3
	// 3
	// 1
	// 1 2 3
	// 3
}
