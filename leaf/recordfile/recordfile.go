package recordfile

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
)

//默认值
var Comma = '\t'  //分隔符,默认是制表符
var Comment = '#' //注释符

type Index map[interface{}]interface{} //索引类型

//记录文件类型定义
type RecordFile struct {
	Comma      rune         //字符类型
	Comment    rune         //字符类型
	typeRecord reflect.Type //反射类型
	records    []interface{}
	indexes    []Index
}

//创建一个记录文件,一个记录文件对应一个结构体,一行记录则对应结构体的一个值
func New(st interface{}) (*RecordFile, error) {
	typeRecord := reflect.TypeOf(st)                              //获取st类型
	if typeRecord == nil || typeRecord.Kind() != reflect.Struct { //判断st合法性，必须是个结构体
		return nil, errors.New("st must be a struct")
	}

	//只是检查类型是否正确
	for i := 0; i < typeRecord.NumField(); i++ { //遍历结构体内的所有字段
		f := typeRecord.Field(i) //取得对应的字段

		kind := f.Type.Kind() //判断字段类型
		switch kind {
		case reflect.Bool: //布尔型
		case reflect.Int: //整型(有符号)
		case reflect.Int8: //有符号8位
		case reflect.Int16: //有符号16位
		case reflect.Int32: //有符号32位
		case reflect.Int64: //有符号64位
		case reflect.Uint8: //无符号8位
		case reflect.Uint16: //无符号16位
		case reflect.Uint32: //无符号32位
		case reflect.Uint64: //无符号32位
		case reflect.Float32: //32位浮点数
		case reflect.Float64: //64位浮点数
		case reflect.String: //字符串
		case reflect.Struct: //结构体
		case reflect.Array: //数组
		case reflect.Slice: //切片
		//如果不是上面的类型，就属于非法类型
		default:
			return nil, fmt.Errorf("invalid type: %v %s",
				f.Name, kind)
		}

		tag := f.Tag        //获取字段的标签
		if tag == "index" { //如果是索引标签
			switch kind { //判断类型
			case reflect.Struct, reflect.Array, reflect.Slice: //索引字段不能是结构体 数组 切片
				return nil, fmt.Errorf("could not index %s field %v %v",
					kind, i, f.Name)
			}
		}
	}

	rf := new(RecordFile)      //创建一个记录文件
	rf.typeRecord = typeRecord //保存Type

	return rf, nil //返回
}

//读取记录文件
func (rf *RecordFile) Read(name string) error {
	file, err := os.Open(name) //打开文件
	if err != nil {
		return err
	}
	defer file.Close() //延迟关闭文件

	if rf.Comma == 0 { //设置分隔符
		rf.Comma = Comma
	}
	if rf.Comment == 0 { //设置注释符
		rf.Comment = Comment
	}
	reader := csv.NewReader(file)  //创建一个csv reader
	reader.Comma = rf.Comma        //设置分隔符
	reader.Comment = rf.Comment    //设置注释符
	lines, err := reader.ReadAll() //读取所有记录
	if err != nil {
		return err
	}

	typeRecord := rf.typeRecord //获取Type

	// make records
	records := make([]interface{}, len(lines)-1) //创建切片保存记录,第一行（中文说明字段）不用保存

	// make indexes
	indexes := []Index{}                         //索引切片，存储多个索引，索引本身实际上是一个map
	for i := 0; i < typeRecord.NumField(); i++ { //遍历所有字段
		tag := typeRecord.Field(i).Tag //获取每个字段的标签
		if tag == "index" {            //如果标签是索引标签
			indexes = append(indexes, make(Index)) //添加索引到索引切片
		}
	}

	for n := 1; n < len(lines); n++ { //遍历所有记录，一行对应一个typeRecord类型的值
		value := reflect.New(typeRecord) //创建一个指针指向特定类型的值(零值)
		records[n-1] = value.Interface() //转化指针为interface并保存在records内
		record := value.Elem()           //获取值本身,value是interface或pointer

		line := lines[n]                        //该行的数据
		if len(line) != typeRecord.NumField() { //判断该行字段数是否匹配
			return fmt.Errorf("line %v, field count mismatch: %v %v",
				n, len(line), typeRecord.NumField())
		}

		iIndex := 0

		for i := 0; i < typeRecord.NumField(); i++ { //遍历所有字段
			f := typeRecord.Field(i) //获得字段

			// records
			strField := line[i]      //字段值(字符串)
			field := record.Field(i) //获得字段
			if !field.CanSet() {     //如果字段不可设置
				continue //继续循环
			}

			var err error

			kind := f.Type.Kind()     //字段的类型
			if kind == reflect.Bool { //布尔型
				var v bool
				v, err = strconv.ParseBool(strField) //转化成Bool
				if err == nil {
					field.SetBool(v) //保存值
				}
			} else if kind == reflect.Int || //有符号整型，少了Int8判断
				kind == reflect.Int16 ||
				kind == reflect.Int32 ||
				kind == reflect.Int64 {
				var v int64
				v, err = strconv.ParseInt(strField, 0, f.Type.Bits()) //转化成整型
				if err == nil {
					field.SetInt(v) //保存值
				}
			} else if kind == reflect.Uint8 || //无符号整型
				kind == reflect.Uint16 ||
				kind == reflect.Uint32 ||
				kind == reflect.Uint64 {
				var v uint64
				v, err = strconv.ParseUint(strField, 0, f.Type.Bits()) //转化
				if err == nil {
					field.SetUint(v) //保存
				}
			} else if kind == reflect.Float32 || //浮点型
				kind == reflect.Float64 {
				var v float64
				v, err = strconv.ParseFloat(strField, f.Type.Bits())
				if err == nil {
					field.SetFloat(v)
				}
			} else if kind == reflect.String { //字符串
				field.SetString(strField)
			} else if kind == reflect.Struct || //结构体 数组 切片，用JSON表达
				kind == reflect.Array ||
				kind == reflect.Slice {
				err = json.Unmarshal([]byte(strField), field.Addr().Interface()) //解析JSON串
			}

			if err != nil { //出错
				return fmt.Errorf("parse field (row=%v, col=%v) error: %v",
					n, i, err)
			}

			// indexes
			//设置索引
			if f.Tag == "index" { //如果该字段是索引字段
				index := indexes[iIndex]                   //获取Index
				iIndex++                                   //自增索引切片的索引
				if _, ok := index[field.Interface()]; ok { //判断多条记录之间的索引字段是否重复
					return fmt.Errorf("index error: duplicate at (row=%v, col=%v)",
						n, i)
				}
				index[field.Interface()] = records[n-1] //保存索引，实际上是保存了一个指针
			}
		}
	}

	rf.records = records //设置记录字段，其实是指向typeRecord类型的值的指针切片
	rf.indexes = indexes //设置索引字段，一个Index的切片，Index又是一个map:索引字段值到该行记录的指针的映射

	return nil
}

//获取记录指针
func (rf *RecordFile) Record(i int) interface{} {
	return rf.records[i]
}

//获取记录的数目
func (rf *RecordFile) NumRecord() int {
	return len(rf.records) //取records（切片）的长度即可
}

//获取Index(一个map) 多索引的时候用到
func (rf *RecordFile) Indexes(i int) Index {
	if i >= len(rf.indexes) {
		return nil
	}
	return rf.indexes[i]
}

//获取记录指针
func (rf *RecordFile) Index(i interface{}) interface{} {
	index := rf.Indexes(0) //单索引
	if index == nil {      //没有Index
		return nil //空值
	}
	return index[i] //返回该行记录的指针
}
