// 参数的优先级：命令行 > 配置文件 > 默认参数

package common

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

// 解析并保存参数信息
func InitParams() {
	args := os.Args
	for i := 1; i < len(args); i++ {
		arg := args[i]
		// 打印帮助信息
		if arg == "-h" || arg == "--help" {
			fmt.Print(helpInformation)
			os.Exit(0)
		}
		// 从配置文件中取到配置信息并做修改
		if arg == "-c" || arg == "--conf" {
			if len(args) == i+1 {
				fmt.Println("Please specify configuration file.")
				os.Exit(1)
			}
			updateConfFromFile(args[i+1])
			break // 配置文件读取配置已经结束，跳出循环
		}
	}

	// 从命令行获取属性参数
	for i := 1; i < len(args); i += 2 {
		if args[i] == "-c" || args[i] == "--conf" {
			continue
		}
		if len(args) == i+1 {
			fmt.Printf("Please specify value for key [%s]\n", args[i])
			os.Exit(1)
		}
		updateConf(strings.Replace(args[i], "--", "", 1), args[i+1])
	}
}

// 根据配置文件更新配置信息
func updateConfFromFile(confFile string) {
	content, err := ioutil.ReadFile(confFile)
	if err != nil {
		fmt.Printf("Read configuration file failed: %v\n", err)
		os.Exit(1)
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.Replace(line, " ", "", -1)
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		configuration := strings.Split(line, "=")
		if len(configuration) != 2 {
			continue
		}
		updateConf(configuration[0], configuration[1])
	}
}

// 更新配置信息
func updateConf(name, value string) {
	switch name {
	case "port":
		Port = uint(string2Num(value))
	case "http":
		HTTPPort = uint(string2Num(value))
	case "id":
		LocalNodeId = value
	case "hosts":
		Hosts = strings.Split(value, ",")
	case "quorum":
		Quorum = uint32(string2Num(value))
	case "level":
		LogLevel = value
	default:
		fmt.Printf("Unknown flag [%s]\n", value)
		os.Exit(1)
	}
}

func string2Num(input string) (output int) {
	var err error
	output, err = strconv.Atoi(input)
	if err != nil {
		fmt.Printf("Parse [%s] to number failed\n", input)
		os.Exit(1)
	}
	return
}
