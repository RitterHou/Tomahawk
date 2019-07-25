package common

// 数组类型的flag
type arrayFlag []string

func (flag *arrayFlag) String() string {
	return ""
}

func (flag *arrayFlag) Set(value string) error {
	*flag = append(*flag, value)
	return nil
}

// 当前节点的一些参数信息
var (
	LocalNodeId string    // 当前节点id
	Port        uint      // 监听的端口
	HTTPPort    uint      // HTTP服务监听的端口
	Hosts       arrayFlag // 种子节点
)

// 网络传输的数据类型
const (
	ExchangeNodeInfo byte = iota // 传输的数据为节点信息
	ShareNodes                   // 共享节点信息
)

// 节点类型
const (
	follower byte = iota
	candidate
	leader
)

// 选举相关的一些数据
var (
	RoleTimeoutCh = make(chan bool)
)

// 日志等级
var LEVEL = DEBUG

const (
	ERROR byte = iota
	WARN
	INFO
	DEBUG
)
