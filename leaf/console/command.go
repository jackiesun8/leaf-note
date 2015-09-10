package console

import (
	"fmt"
	"github.com/name5566/leaf/chanrpc"
	"github.com/name5566/leaf/conf"
	"github.com/name5566/leaf/log"
	"os"
	"path"
	"runtime/pprof"
	"time"
)

//命令列表
//先添加内置命令，后续可以使用Register注册外部命令
var commands = []Command{
	new(CommandHelp),    //帮助命令
	new(CommandCPUProf), //CPU profile
	new(CommandProf),    //profile
}

//命令接口定义
type Command interface {
	// must goroutine safe
	//命令名字用于匹配
	name() string
	// must goroutine safe
	//显示命令帮助信息
	help() string
	// must goroutine safe
	//执行命令功能
	run(args []string) string
}

//外部命令类型定义
type ExternalCommand struct {
	_name  string
	_help  string
	server *chanrpc.Server
}

//返回命令名字
func (c *ExternalCommand) name() string {
	return c._name
}

//返回帮助命令信息
func (c *ExternalCommand) help() string {
	return c._help
}

//执行命令
func (c *ExternalCommand) run(_args []string) string {
	args := make([]interface{}, len(_args))
	for i, v := range _args {
		args[i] = v
	}

	ret, err := c.server.Open(0).Call1(c._name, args...)
	if err != nil {
		return err.Error()
	}
	output, ok := ret.(string)
	if !ok {
		return "invalid output type"
	}

	return output
}

// you must call the function before calling console.Init，也就是在leaf.Run之前
// goroutine not safe
// 注册命令
func Register(name string, help string, f interface{}, server *chanrpc.Server) {
	for _, c := range commands {
		if c.name() == name {
			log.Fatal("command %v is already registered", name)
		}
	}

	server.Register(name, f)

	c := new(ExternalCommand)
	c._name = name
	c._help = help
	c.server = server
	commands = append(commands, c)
}

// help
//帮助命令类型定义
type CommandHelp struct{}

//名字为help
func (c *CommandHelp) name() string {
	return "help"
}

//帮助信息为this help text
func (c *CommandHelp) help() string {
	return "this help text"
}

//帮助命令忽略传入的命令参数
func (c *CommandHelp) run([]string) string {
	output := "Commands:\r\n"    //前缀
	for _, c := range commands { //遍历所有命令
		output += c.name() + " - " + c.help() + "\r\n" //添加所有命令的命令名+命令帮助到输出信息中
	}
	output += "quit - exit console" //后缀，quit是退出控制台

	return output //返回输出
}

// cpuprof
type CommandCPUProf struct{}

//名字
func (c *CommandCPUProf) name() string {
	return "cpuprof"
}

//帮助
func (c *CommandCPUProf) help() string {
	return "CPU profiling for the current process"
}

//用法信息
func (c *CommandCPUProf) usage() string {
	return "cpuprof writes runtime profiling data in the format expected by \r\n" +
		"the pprof visualization tool\r\n\r\n" +
		"Usage: cpuprof start|stop\r\n" +
		"  start - enables CPU profiling\r\n" +
		"  stop  - stops the current CPU profile"
}

//执行
func (c *CommandCPUProf) run(args []string) string {
	if len(args) == 0 { //如果参数为空
		return c.usage() //返回用法信息
	}

	switch args[0] { //判断参数
	case "start": //启动
		fn := profileName() + ".cpuprof" //写入文件名
		f, err := os.Create(fn)          //创建文件
		if err != nil {                  //出错
			return err.Error() //返回出错信息
		}
		err = pprof.StartCPUProfile(f) //开始profile
		if err != nil {                //出错
			f.Close()          //关闭文件
			return err.Error() //返回出错信息
		}
		return fn //开始返回profile文件名
	case "stop": //停止
		pprof.StopCPUProfile() //停止profile
		return ""              //结束什么都不返回
	default:
		return c.usage() //输入其他不识别命令，返回用法信息
	}
}

//获取profile名字
func profileName() string {
	now := time.Now()
	return path.Join(conf.ProfilePath,
		fmt.Sprintf("%d%02d%02d_%02d_%02d_%02d",
			now.Year(),
			now.Month(),
			now.Day(),
			now.Hour(),
			now.Minute(),
			now.Second())) //profile路径+日期 时间
}

// prof
type CommandProf struct{}

//名字
func (c *CommandProf) name() string {
	return "prof"
}

//帮助
func (c *CommandProf) help() string {
	return "writes a pprof-formatted snapshot"
}

//用法信息
func (c *CommandProf) usage() string {
	return "prof writes runtime profiling data in the format expected by \r\n" +
		"the pprof visualization tool\r\n\r\n" +
		"Usage: prof goroutine|heap|thread|block\r\n" +
		"  goroutine - stack traces of all current goroutines\r\n" +
		"  heap      - a sampling of all heap allocations\r\n" +
		"  thread    - stack traces that led to the creation of new OS threads\r\n" +
		"  block     - stack traces that led to blocking on synchronization primitives"
}

func (c *CommandProf) run(args []string) string {
	if len(args) == 0 { //参数为0
		return c.usage() //返回用法信息
	}

	var (
		p  *pprof.Profile
		fn string
	)
	switch args[0] { //识别命令
	case "goroutine":
		p = pprof.Lookup("goroutine") //查找goroutine的profile
		fn = profileName() + ".gprof" //文件名
	case "heap":
		p = pprof.Lookup("heap")      //查找heap的profile
		fn = profileName() + ".hprof" //文件名
	case "thread":
		p = pprof.Lookup("threadcreate") //查找threadcreate的profile
		fn = profileName() + ".tprof"    //文件名
	case "block":
		p = pprof.Lookup("block")     //查找block的profile
		fn = profileName() + ".bprof" //文件名
	default: //未识别命令
		return c.usage() //返回用法信息
	}

	f, err := os.Create(fn) //创建文件
	if err != nil {         //出错
		return err.Error() //返回错误信息
	}
	defer f.Close() //延迟关闭文件
	err = p.WriteTo(f, 0)
	if err != nil {
		return err.Error()
	}

	return fn
}
