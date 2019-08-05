package tog

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// 日志等级
var localLevel = DEBUG

func Init(level string) {
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
	log.SetOutput(os.Stdout)

	level = strings.ToUpper(level)
	fmt.Printf("\033[33mTomahawk log level: %s\n", level)
	switch level {
	case "ERROR":
		localLevel = ERROR
	case "WARN":
		localLevel = WARN
	case "INFO":
		localLevel = INFO
	case "DEBUG":
		localLevel = DEBUG
	default:
		fmt.Printf("Unknown log level [%s], using default level [DEBUG]\n", level)
	}

	fmt.Println(`
#######  ####  #    #   ##   #    #   ##   #    # #    # 
   #    #    # ##  ##  #  #  #    #  #  #  #    # #   #  
   #    #    # # ## # #    # ###### #    # #    # ####   
   #    #    # #    # ###### #    # ###### # ## # #  #   
   #    #    # #    # #    # #    # #    # ##  ## #   #  
   #     ####  #    # #    # #    # #    # #    # #    #`)
	fmt.Println("\033[0m")
}

const (
	ERROR byte = iota
	WARN
	INFO
	DEBUG
)

// 判断日志等级
func LogLevel(level byte) bool {
	return localLevel >= level
}
