package recordfile_test

import (
	"fmt"
	"github.com/name5566/leaf/recordfile"
)

func Example() {
	type Record struct {
		// index 0
		IndexInt int "index" //数字索引
		// index 1
		IndexStr string    "index" //字符串索引
		_Number  int32     //数字类型
		Str      string    //字符串类型
		Arr1     [2]int    //数组类型
		Arr2     [3][2]int //嵌套数组
		Arr3     []int     //变长数组
		St       struct {  //结构体类型
			Name string "name"
			Num  int    "num"
		}
	}

	rf, err := recordfile.New(Record{})
	if err != nil {
		return
	}

	err = rf.Read("test.txt")
	if err != nil {
		return
	}

	for i := 0; i < rf.NumRecord(); i++ {
		r := rf.Record(i).(*Record)
		fmt.Println(r.IndexInt)
	}

	r := rf.Index(2).(*Record)
	fmt.Println(r.Str)

	r = rf.Indexes(0)[2].(*Record)
	fmt.Println(r.Str)

	r = rf.Indexes(1)["three"].(*Record)
	fmt.Println(r.Str)
	fmt.Println(r.Arr1[1])
	fmt.Println(r.Arr2[2][0])
	fmt.Println(r.Arr3[0])
	fmt.Println(r.St.Name)

	// Output:
	// 1
	// 2
	// 3
	// cat
	// cat
	// book
	// 6
	// 4
	// 6
	// name5566
}
