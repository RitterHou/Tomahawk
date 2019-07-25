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

var (
	LocalNodeId string    // 当前节点id
	Port        uint      // 监听的端口
	HTTPPort    uint      // HTTP服务监听的端口
	Hosts       arrayFlag // 种子节点
)

// 网络传输的数据类型
const (
	ExchangeNodeId byte = iota // 传输的数据为nodeId
	ShareNodes                 // 共享节点信息
)

// 节点类型
const (
	follower byte = iota
	candidate
	leader
)
