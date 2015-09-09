package log

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"time"
)

// levels
//日志级别定义
const (
	debugLevel   = 0 //非关键日志
	releaseLevel = 1 //关键日志
	errorLevel   = 2 //错误日志
	fatalLevel   = 3 //致命错误日志。Fatal 日志比较特殊，每次输出 Fatal 日志之后游戏服务器进程就会结束
)

//Debug < Release < Error < Fatal（日志级别高低）

//日志输出前缀字符串定义
const (
	printDebugLevel   = "[debug  ] "
	printReleaseLevel = "[release] "
	printErrorLevel   = "[error  ] "
	printFatalLevel   = "[fatal  ] "
)

//上层Logger定义
type Logger struct {
	level      int         //日志级别
	baseLogger *log.Logger //底层logger,基于go的log包
	baseFile   *os.File    //日志写入的文件
}

func New(strLevel string, pathname string) (*Logger, error) { //上层logger创建函数
	// level
	var level int
	switch strings.ToLower(strLevel) { //根据传入的日志级别，设置日志级别
	case "debug":
		level = debugLevel
	case "release":
		level = releaseLevel
	case "error":
		level = errorLevel
	case "fatal":
		level = fatalLevel
	default:
		return nil, errors.New("unknown level: " + strLevel)
	}

	// logger
	var baseLogger *log.Logger
	var baseFile *os.File
	if pathname != "" { //写入文件路径名
		now := time.Now()

		filename := fmt.Sprintf("%d%02d%02d_%02d_%02d_%02d.log", //文件名,时间命名
			now.Year(),
			now.Month(),
			now.Day(),
			now.Hour(),
			now.Minute(),
			now.Second())

		file, err := os.Create(path.Join(pathname, filename)) //创建文件
		if err != nil {
			return nil, err
		}

		baseLogger = log.New(file, "", log.LstdFlags) //创建logger(底层的)，LstdFlags只显示日期和时间
		baseFile = file                               //保存文件引用
	} else {
		baseLogger = log.New(os.Stdout, "", log.LstdFlags) //输出log到标准输出
	}

	// new
	//创建上层logger
	logger := new(Logger)
	//设置字段值
	logger.level = level           //日志级别字段
	logger.baseLogger = baseLogger //底层logger
	logger.baseFile = baseFile     //文件引用

	return logger, nil
}

// It's dangerous to call the method on logging
func (logger *Logger) Close() {
	if logger.baseFile != nil { //写入文件存在
		logger.baseFile.Close() //关闭文件
	}
	//置空字段
	logger.baseLogger = nil
	logger.baseFile = nil
}

//最终调用的日志输出函数
func (logger *Logger) doPrintf(level int, printLevel string, format string, a ...interface{}) {
	if level < logger.level { //日志级别小于设定的日志级别
		return //不输出
	}
	//底层logger为空
	if logger.baseLogger == nil {
		panic("logger closed") //抛出一个异常，在defer中通过recover可以捕获异常
	}

	format = printLevel + format           //前缀+格式
	logger.baseLogger.Printf(format, a...) //输出日志

	if level == fatalLevel { //如果为fatal日志
		os.Exit(1) //退出程序
	}
}

//不同级别的日志函数
func (logger *Logger) Debug(format string, a ...interface{}) {
	logger.doPrintf(debugLevel, printDebugLevel, format, a...)
}

func (logger *Logger) Release(format string, a ...interface{}) {
	logger.doPrintf(releaseLevel, printReleaseLevel, format, a...)
}

func (logger *Logger) Error(format string, a ...interface{}) {
	logger.doPrintf(errorLevel, printErrorLevel, format, a...)
}

func (logger *Logger) Fatal(format string, a ...interface{}) {
	logger.doPrintf(fatalLevel, printFatalLevel, format, a...)
}

//创建一个默认的logger，日志级别为debug，使用者就可以不用自己定义logger，而是直接引入包，使用包导出函数即可。
var gLogger, _ = New("debug", "")

//包级导出日志函数
// It's dangerous to call the method on logging
//导出函数定义，传入一个logger,替换默认的gLogger
func Export(logger *Logger) {
	if logger != nil {
		gLogger = logger
	}
}

func Debug(format string, a ...interface{}) {
	gLogger.Debug(format, a...)
}

func Release(format string, a ...interface{}) {
	gLogger.Release(format, a...)
}

func Error(format string, a ...interface{}) {
	gLogger.Error(format, a...)
}

func Fatal(format string, a ...interface{}) {
	gLogger.Fatal(format, a...)
}

func Close() {
	gLogger.Close()
}
