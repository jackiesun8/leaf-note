package timer

// reference: https://github.com/robfig/cron
import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Field name   | Mandatory? | Allowed values | Allowed special characters
// ----------   | ---------- | -------------- | --------------------------
// Seconds      | No         | 0-59           | * / , -
// Minutes      | Yes        | 0-59           | * / , -
// Hours        | Yes        | 0-23           | * / , -
// Day of month | Yes        | 1-31           | * / , -
// Month        | Yes        | 1-12           | * / , -
// Day of week  | Yes        | 0-6            | * / , -

// const (
//     Sunday Weekday = iota
//     Monday
//     Tuesday
//     Wednesday
//     Thursday
//     Friday
//     Saturday
// )

//cron表达式定义
type CronExpr struct {
	sec   uint64
	min   uint64
	hour  uint64
	dom   uint64
	month uint64
	dow   uint64
}

// goroutine safe
//创建cron表达式
func NewCronExpr(expr string) (cronExpr *CronExpr, err error) {
	fields := strings.Fields(expr)            //用空格分割表达式
	if len(fields) != 5 && len(fields) != 6 { //数组长度为5或者6，因为Seconds不是强制设置的
		err = fmt.Errorf("invalid expr %v: expected 5 or 6 fields, got %v", expr, len(fields))
		return
	}

	if len(fields) == 5 { //没有设置Seconds
		fields = append([]string{"0"}, fields...) //自己在最前面添加一个0
	}

	cronExpr = new(CronExpr) //创建一个cron表达式

	//解析字段
	// Seconds
	cronExpr.sec, err = parseCronField(fields[0], 0, 59)
	if err != nil {
		goto onError
	}
	// Minutes
	cronExpr.min, err = parseCronField(fields[1], 0, 59)
	if err != nil {
		goto onError
	}
	// Hours
	cronExpr.hour, err = parseCronField(fields[2], 0, 23)
	if err != nil {
		goto onError
	}
	// Day of month
	cronExpr.dom, err = parseCronField(fields[3], 1, 31)
	if err != nil {
		goto onError
	}
	// Month
	cronExpr.month, err = parseCronField(fields[4], 1, 12)
	if err != nil {
		goto onError
	}
	// Day of week
	cronExpr.dow, err = parseCronField(fields[5], 0, 6)
	if err != nil {
		goto onError
	}
	return

onError:
	err = fmt.Errorf("invalid expr %v: %v", expr, err)
	return
}

// 1. *
// 2. num
// 3. num-num
// 4. */num
// 5. num/num (means num-max/num)
// 6. num-num/num
//解析cron字段
func parseCronField(field string, min int, max int) (cronField uint64, err error) {
	fields := strings.Split(field, ",") //先使用符号“,”分割字段。“，”用于分割一个字段的多个项
	for _, field := range fields {      //遍历每个项
		rangeAndIncr := strings.Split(field, "/") //使用符号“/”分割。获得范围和增幅
		if len(rangeAndIncr) > 2 {                //肯定不大于2
			err = fmt.Errorf("too many slashes: %v", field)
			return
		}

		// range
		startAndEnd := strings.Split(rangeAndIncr[0], "-") //使用符号“-”分割。获得范围的起始值和结束值
		if len(startAndEnd) > 2 {                          //肯定不大于2
			err = fmt.Errorf("too many hyphens: %v", rangeAndIncr[0])
			return
		}

		var start, end int //用于存储起始值和结束值

		//The form "*\/..." is equivalent to the form "first-last/...",that is, an increment over the largest possible range of the field
		if startAndEnd[0] == "*" { //如果起始值为*
			if len(startAndEnd) != 1 { //范围必须只有一个*，而不是first-last形式
				err = fmt.Errorf("invalid range: %v", rangeAndIncr[0])
				return
			}
			start = min //起始值等于最小值
			end = max   //结束值等于最大值
		} else {
			// start
			start, err = strconv.Atoi(startAndEnd[0]) //转化为整数
			if err != nil {
				err = fmt.Errorf("invalid range: %v", rangeAndIncr[0])
				return
			}
			// end
			//The form "N/..." is accepted as meaning "N-MAX/...", that is, starting at N, use the increment until the end of that specific range.
			if len(startAndEnd) == 1 {
				if len(rangeAndIncr) == 2 { //有增幅
					end = max //结束值等于最大值
				} else { //没有增幅
					end = start //结束值等于起始值
				}
			} else {
				//For example 3-59/15 in the 1st field (minutes) would indicate the 3rd minute of the hour and every 15 minutes thereafter
				end, err = strconv.Atoi(startAndEnd[1]) //获取结束值
				if err != nil {
					err = fmt.Errorf("invalid range: %v", rangeAndIncr[0])
					return
				}
			}
		}

		if start > end { //起始值不能大于结束值
			err = fmt.Errorf("invalid range: %v", rangeAndIncr[0])
			return
		}
		if start < min { //起始值不能小于最小值
			err = fmt.Errorf("out of range [%v, %v]: %v", min, max, rangeAndIncr[0])
			return
		}
		if end > max { //结束值不能大于最大值
			err = fmt.Errorf("out of range [%v, %v]: %v", min, max, rangeAndIncr[0])
			return
		}
		//没有检查增幅的有效性

		// increment
		var incr int                //用于存储增幅
		if len(rangeAndIncr) == 1 { //没有增幅
			incr = 1 //增幅为1，为什么不是0，如果用户设置增幅为1怎么办？
		} else { //有增幅
			incr, err = strconv.Atoi(rangeAndIncr[1]) //获取增幅
			if err != nil {
				err = fmt.Errorf("invalid increment: %v", rangeAndIncr[1])
				return
			}
			if incr <= 0 { //增幅不能小于等于0
				err = fmt.Errorf("invalid increment: %v", rangeAndIncr[1])
				return
			}
		}

		// cronField
		if incr == 1 { //没有增幅，增幅为1
			cronField |= ^(math.MaxUint64 << uint(end+1)) & (math.MaxUint64 << uint(start))
			//比如start和end都等于2（没有增幅，start和end相等）
			//^(math.MaxUint64 << uint(end+1))等于：
			//0000 0000 0000 0000 0000 0000 0000 0000 0000 0000 0000 0000 0000 0000 0000 0111
			//(math.MaxUint64 << uint(start))等于：
			//1111 1111 1111 1111 1111 1111 1111 1111 1111 1111 1111 1111 1111 1111 1111 1100
			//两者与操作后等于：
			//0000 0000 0000 0000 0000 0000 0000 0000 0000 0000 0000 0000 0000 0000 0000 0100
			//就是相当于左移了2位

			//比如start等于0，end等于6
			//^(math.MaxUint64 << uint(end+1))等于：
			//0000 0000 0000 0000 0000 0000 0000 0000 0000 0000 0000 0000 0000 0000 0111 1111
			//(math.MaxUint64 << uint(start))等于：
			//1111 1111 1111 1111 1111 1111 1111 1111 1111 1111 1111 1111 1111 1111 1111 1111
			//两者与操作后等于：
			//0000 0000 0000 0000 0000 0000 0000 0000 0000 0000 0000 0000 0000 0000 0111 1111
			//
		} else {
			for i := start; i <= end; i += incr {
				cronField |= 1 << uint(i) //根据增幅计算关键值再移位
			}
		}
	}

	return
}

func (e *CronExpr) matchDay(t time.Time) bool {
	// day-of-month blank 1-31
	// day-of-month标志位都设置了
	if e.dom == 0xfffffffe {
		return 1<<uint(t.Weekday())&e.dow != 0
	}

	// day-of-week blank 0-6
	// day-of-week标志位都设置了
	if e.dow == 0x7f {
		return 1<<uint(t.Day())&e.dom != 0
	}
	//不确定哪个能够匹配到
	return 1<<uint(t.Weekday())&e.dow != 0 ||
		1<<uint(t.Day())&e.dom != 0
}

// goroutine safe
// 计算下一次时间
func (e *CronExpr) Next(t time.Time) time.Time {
	// the upcoming second
	t = t.Truncate(time.Second).Add(time.Second) //计算下一秒。先截断到秒再加一秒

	year := t.Year()  //保存当前的年份
	initFlag := false //标志位

retry: //重试标签
	// Year
	if t.Year() > year+1 { //跨年了
		return time.Time{} //返回零值
	}

	// Month
	//匹配月
	for 1<<uint(t.Month())&e.month == 0 { //没有匹配到，就执行循环，匹配的话就跳过
		if !initFlag { //没有初始化
			initFlag = true                                                 //设置为已经初始化
			t = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location()) //新建一个时间，就是初始化
		}

		t = t.AddDate(0, 1, 0)         //加一个月
		if t.Month() == time.January { //已经到了1月。已经遍历了从当前月份到12月份
			goto retry //跳出循环，继续匹配从1月到12月份
		}
	}

	// Day
	//匹配日 day-of-month和day-of-week
	for !e.matchDay(t) {
		if !initFlag {
			initFlag = true
			t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
		}

		t = t.AddDate(0, 0, 1) //加一日
		if t.Day() == 1 {
			goto retry
		}
	}

	// Hours
	//匹配小时
	for 1<<uint(t.Hour())&e.hour == 0 {
		if !initFlag {
			initFlag = true
			t = t.Truncate(time.Hour)
		}

		t = t.Add(time.Hour)
		if t.Hour() == 0 {
			goto retry
		}
	}

	// Minutes
	//匹配分钟
	for 1<<uint(t.Minute())&e.min == 0 {
		if !initFlag {
			initFlag = true
			t = t.Truncate(time.Minute)
		}

		t = t.Add(time.Minute)
		if t.Minute() == 0 {
			goto retry
		}
	}

	// Seconds
	//匹配秒
	for 1<<uint(t.Second())&e.sec == 0 {
		if !initFlag {
			initFlag = true
			//不用截断了，程序开头已经截断到秒了
		}

		t = t.Add(time.Second)
		if t.Second() == 0 {
			goto retry
		}
	}

	return t
}
