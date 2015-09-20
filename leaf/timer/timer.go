package timer

import (
	"errors"
	"github.com/name5566/leaf/conf"
	"github.com/name5566/leaf/log"
	"runtime"
	"time"
)

// one dispatcher per goroutine (goroutine not safe)
//分发器类型定义
type Dispatcher struct {
	ChanTimer chan *Timer //管道，用于传输定时器
}

//创建分发器
func NewDispatcher(l int) *Dispatcher {
	disp := new(Dispatcher)               //创建分发器
	disp.ChanTimer = make(chan *Timer, l) //创建管道，传输到时的定时器
	return disp                           //返回分发器
}

// Timer
//定时器类型定义
type Timer struct {
	t  *time.Timer //底层定时器
	cb func()      //回调函数
}

//停止定时器
func (t *Timer) Stop() {
	t.t.Stop() //停止底层定时器
	t.cb = nil //置空回调函数
}

//调用定时器的回调函数
func (t *Timer) Cb() {
	defer func() { //延迟执行
		t.cb = nil                    //置空回调
		if r := recover(); r != nil { //捕获异常
			if conf.LenStackBuf > 0 { //堆栈buf长度大于0
				//打印堆栈信息
				buf := make([]byte, conf.LenStackBuf)
				l := runtime.Stack(buf, false)
				log.Error("%v: %s", r, buf[:l])
			} else {
				log.Error("%v", r) //打印异常
			}
		}
	}()
	//回调不为空
	if t.cb != nil {
		t.cb() //调用回调
	}
}

//注册定时器
func (disp *Dispatcher) AfterFunc(d time.Duration, cb func()) *Timer {
	t := new(Timer)                  //创建定时器
	t.cb = cb                        //设置回调函数
	t.t = time.AfterFunc(d, func() { //注意，这里的func是在定时器自己的goroutine中执行的
		disp.ChanTimer <- t //定时器到时，将定时器发送到管道中
	})
	return t //返回自定义的定时器
}

// Cron
//计划任务类型定义
type Cron struct {
	t *Timer //自定义定时器
}

//停止计划任务
func (c *Cron) Stop() {
	c.t.Stop() //关闭自定义定时器
}

//注册计划任务
func (disp *Dispatcher) CronFunc(expr string, _cb func()) (*Cron, error) {
	cronExpr, err := NewCronExpr(expr) //创建一个计划任务表达式
	if err != nil {
		return nil, err
	}

	//第一次计划任务
	now := time.Now()              //当前时间
	nextTime := cronExpr.Next(now) //下一个时间
	if nextTime.IsZero() {
		return nil, errors.New("next time not found")
	}

	cron := new(Cron) //创建一个计划任务

	// callback
	var cb func() //定义一个回调函数，执行后续计划任务
	cb = func() { //回调函数定义
		defer _cb() //延迟执行计划任务用户回调。第一次计划任务到时到第二次计划任务注册完毕才执行用户回调

		now := time.Now()              //当前时间
		nextTime := cronExpr.Next(now) //下一个时间
		if nextTime.IsZero() {         //如果为零值
			return //直接返回，不注册后续的计划任务，会再执行一次用户回调
		}
		cron.t = disp.AfterFunc(nextTime.Sub(now), cb) //计算时间差值，注册定时器
	}

	cron.t = disp.AfterFunc(nextTime.Sub(now), cb) //第一次计划任务
	return cron, nil
}
