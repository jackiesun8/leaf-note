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
	disp.ChanTimer = make(chan *Timer, l) //创建管道，传输定时器
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

//
func (disp *Dispatcher) AfterFunc(d time.Duration, cb func()) *Timer {
	t := new(Timer)
	t.cb = cb
	t.t = time.AfterFunc(d, func() {
		disp.ChanTimer <- t
	})
	return t
}

// Cron
//计划任务类型定义
type Cron struct {
	t *Timer
}

//
func (c *Cron) Stop() {
	c.t.Stop()
}

func (disp *Dispatcher) CronFunc(expr string, _cb func()) (*Cron, error) {
	cronExpr, err := NewCronExpr(expr)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	nextTime := cronExpr.Next(now)
	if nextTime.IsZero() {
		return nil, errors.New("next time not found")
	}

	cron := new(Cron)

	// callback
	var cb func()
	cb = func() {
		defer _cb()

		now := time.Now()
		nextTime := cronExpr.Next(now)
		if nextTime.IsZero() {
			return
		}
		cron.t = disp.AfterFunc(nextTime.Sub(now), cb)
	}

	cron.t = disp.AfterFunc(nextTime.Sub(now), cb)
	return cron, nil
}
