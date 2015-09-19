package g_test

import (
	"fmt"
	"github.com/name5566/leaf/go"
	"time"
)

func Example() {
	d := g.New(10) //创建一个Go实例，回调管道长度为10

	// go 1
	var res int //接收结果
	d.Go(func() {
		fmt.Println("1 + 1 = ?")
		res = 1 + 1 //在一个新的goroutine执行运算
	}, func() {
		fmt.Println(res) //打印结果
	})

	d.Cb(<-d.ChanCb) //读出回调执行，即上方的打印结果函数

	// go 2
	//没有显示的调用d.Cb(<-d.ChanCb),但是关闭d的时候会执行完全部的回调函数
	d.Go(func() {
		fmt.Print("My name is ")
	}, func() {
		fmt.Println("Leaf")
	})

	d.Close()

	// Output:
	// 1 + 1 = ?
	// 2
	// My name is Leaf
}

func ExampleLinearContext() {
	d := g.New(10) //创建一个go实例

	// parallel
	//并发
	d.Go(func() {
		time.Sleep(time.Second / 2) //因为这里有延时操作,所以并发里的例子是先打印2，后打印1
		fmt.Println("1")
	}, nil)
	d.Go(func() {
		fmt.Println("2")
	}, nil)

	d.Cb(<-d.ChanCb) //读取出的是nil
	d.Cb(<-d.ChanCb) //读取出的是nil

	// linear
	//串行
	c := d.NewLinearContext() //创建一个线性上下文,注意这里是在go实例上调用NewLinearContext，不是g
	c.Go(func() {
		time.Sleep(time.Second / 2) //因为是线性的，所以这里有延时操作，也会按顺序执行
		fmt.Println("1")
	}, nil)
	c.Go(func() {
		fmt.Println("2")
	}, nil)

	d.Close() //关闭d的时候会执行完全部的回调函数

	// Output:
	// 2
	// 1
	// 1
	// 2
}
