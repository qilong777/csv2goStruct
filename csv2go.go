package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

// CsvReader 存放读取csv文件格式的结构体
type CsvReader struct {
	// csv表名
	Name string

	// 存放key
	Key []string

	// 存放类型
	Type []string

	// 存放描述信息
	Desc []string

	// 存放数据信息
	Record [][]string
}

// 定义模板函数，去除带#的列
func isShow(str string) bool {
	return !(str[0] == '#')
}

// 判断是否要导入tool包
func isTool(forms []string) bool {
	for _, str := range forms {
		if str == "json" || str == "json[]" {
			return true
		}
	}
	return false
}

// 判断是否要导入tool包
func getID(form string, val string) string {
	if form == "string" {
		return `"` + val + `"`
	}
	return val
}

// 根据json字符串生成interface
func getDataByJSON(form string, str string) string {
	var result string = str
	reg := regexp.MustCompile(`\]`)
	result = reg.ReplaceAllString(result, `}`)

	reg = regexp.MustCompile(`{`)
	result = reg.ReplaceAllString(result, `algo.JSONMap{`)

	reg = regexp.MustCompile(`\[`)
	result = reg.ReplaceAllString(result, `algo.JSONArray{`)
	return result
}

// Output 输出为go文件 参数为表名和目录名
func (a *CsvReader) Output(name string, pathName string) {

	fileName := pathName + name + ".go"
	funcMap := template.FuncMap{"isShow": isShow, "getDataByJSON": getDataByJSON, "isTool": isTool, "getID": getID}
	tmpl, err := template.New("min-template.tmpl").Funcs(funcMap).ParseFiles("./template/min-template.tmpl")
	if err != nil {
		fmt.Println(err)
		return
	}
	// 打开或者创建文件
	fileObj, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println("Failed to open the file", err.Error())
		os.Exit(2)
	}
	// 延迟关闭
	defer fileObj.Close()

	writer := bufio.NewWriter(fileObj)

	// 把数据传进模板字符串并把文件创给writer内容（二进制的流）
	err = tmpl.Execute(writer, a)
	if err != nil {
		fmt.Println(err)
	} else {
		// 刷新文件
		writer.Flush()
	}
}

// Capitalize 字符串首字母大写
func Capitalize(str string) string {
	var upperStr string
	vv := []rune(str) // 后文有介绍
	for i := 0; i < len(vv); i++ {
		if i == 0 {
			if vv[i] >= 97 && vv[i] <= 122 { // 后文有介绍
				vv[i] -= 32 // string的码表相差32位
				upperStr += string(vv[i])
			} else {
				fmt.Println("Not begins with lowercase letter,")
				return str
			}
		} else {
			upperStr += string(vv[i])
		}
	}
	return upperStr
}

// RemoveContents 删除文件夹下所有文件
func RemoveContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

// InputCsv 读取csv，处理数据。参数是csv表路径和生成的go代码的路径。
// 路径的最后面都要加/
func InputCsv(originPath string, targetPath string) {
	//以只读的方式打开目录

	RemoveContents(targetPath)
	f, err := os.OpenFile(originPath, os.O_RDONLY, os.ModeDir)
	if err != nil {
		fmt.Println(err.Error())
	}
	//延迟关闭目录
	defer f.Close()
	fileInfo, _ := f.Readdir(-1)
	//操作系统指定的路径分隔符

	// 循环读取目录下的每个文件
	for _, info := range fileInfo {

		// 获取文件名字
		fileName := info.Name()
		fmt.Println("文件：" + fileName)
		file, err := os.Open(originPath + fileName)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer file.Close()
		reader := csv.NewReader(file)
		// 初始化
		csvData := CsvReader{}
		i := 1
		// 一行一行读，方便数据处理
		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			} else if err != nil {
				fmt.Println("Error:", err)
				return
			}
			// 第一行获取Key
			if i == 1 {
				csvData.Key = record
				length := len(record)
				for i := 0; i < length; i++ {
					csvData.Key[i] = Capitalize(record[i])
				}
				// 第二行获取数据类型
			} else if i == 2 {
				csvData.Type = record
				// 第三行获取描述
			} else if i == 3 {
				csvData.Desc = record
				// 后面的都是获取数据
			} else {
				if isShow(record[0]) {
					csvData.Record = append(csvData.Record, record)
				}
			}
			i++
		}
		// 获取表名
		lastIndex := len(fileName) - 4
		fileName = string([]rune(fileName)[:lastIndex])

		//
		length := len(csvData.Type)
		for i := 0; i < length; i++ {
			form := csvData.Type[i]
			if form == "" {
				csvData.Type[i] = "string"
				continue
			}
			if form == "json" || form == "json[]" {
				continue
			}
			if form[len(form)-1] == ']' {
				startIndex := strings.Index(form, "[")
				formPrefix := string([]rune(form)[:startIndex])
				if formPrefix == "float" {
					formPrefix = "float32"
				}
				formSuffix := string([]rune(form)[startIndex:])

				csvData.Type[i] = string([]rune(form)[startIndex:]) + formPrefix

				length1 := len(csvData.Record)
				form = formSuffix + formPrefix
				for j := 0; j < length1; j++ {
					item := csvData.Record[j][i]
					if item == "" || item == "[]" {
						csvData.Record[j][i] = form + "{}"
					} else {
						reg := regexp.MustCompile(`\[`)
						item = reg.ReplaceAllString(item, `{`)
						reg = regexp.MustCompile(`\]`)
						item = reg.ReplaceAllString(item, `}`)
						csvData.Record[j][i] = form + item
					}
				}
			}
		}

		csvData.Name = fileName

		csvData.Output(fileName, targetPath)

	}
}

func main() {
	InputCsv("E:\\work\\my_civ\\client\\trunk\\config\\csv\\", "./csv/")
}
