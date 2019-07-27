package tog

// 日志等级
var LEVEL = DEBUG

const (
	ERROR byte = iota
	WARN
	INFO
	DEBUG
)

// 判断日志等级
func LogLevel(level byte) bool {
	return LEVEL >= level
}
