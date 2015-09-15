package conf

import (
	"encoding/json"
	"github.com/name5566/leaf/log"
	"io/ioutil"
)

var Server struct {
	LogLevel     string //日志级别
	LogPath      string //日志路径
	Addr         string //游戏服务器地址
	MaxConnNum   int    //最大连接数
	DBUrl        string //数据库地址
	DBMaxConnNum int    //数据库最大连接数
}

func init() {
	data, err := ioutil.ReadFile("conf/server.json") //读取bin/conf/server.json
	if err != nil {
		log.Fatal("%v", err)
	}
	err = json.Unmarshal(data, &Server) //解析JSON数据到Server中
	if err != nil {
		log.Fatal("%v", err)
	}
}
