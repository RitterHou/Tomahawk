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
	AppendEntries                // leader向follower发送数据
	VoteRequest
)

var Role = Follower

// 节点类型
const (
	Follower byte = iota
	Candidate
	Leader
)

// 选举相关的一些数据
var (
	HeartbeatTimeoutCh = make(chan bool) // 当前节点心跳超时的channel
	VoteSuccessCh      = make(chan bool) // 选举情况channel
	LeaderSendEntryCh  = make(chan bool) // leader发送了信息的channel
)

// 日志等级
var LEVEL = DEBUG

const (
	ERROR byte = iota
	WARN
	INFO
	DEBUG
)

const (
	HeartbeatTimeoutMin = 150
	HeartbeatTimeoutMax = 300
	ElectionTimeoutMin  = 150
	ElectionTimeoutMax  = 300
	LeaderCycleTimeout  = 100
)

// 当前任期
var currentTerm uint32 = 0
