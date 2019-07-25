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
	ShareNodes                 // 节点与节点之间共享自身的节点列表 // TODO 共享节点暂时不支持，因此每个节点必须要完整的添加好其它的节点IP，否则会导致节点无法被发现
)

// 节点类型
const (
	follower byte = iota
	candidate
	leader
)
