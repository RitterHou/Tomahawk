package common

var (
	BuildStamp string
	Version    string
	GoVersion  string
)

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
	Quorum      uint32    // 所谓的“大多数”
)

// 网络传输的数据类型
const (
	ExchangeNodeInfo      byte = iota // 传输的数据为节点信息
	ShareNodes                        // 共享节点信息
	AppendEntries                     // leader向follower发送数据
	AppendEntriesResponse             // follower收到数据之后返回响应
	VoteRequest                       // 投票的请求
	VoteResponse                      // 投票的响应
)

// 根据网络传输类型获取相应的字符串
func GetSocketDataType(socketType byte) string {
	switch socketType {
	case ExchangeNodeInfo:
		return "ExchangeNodeInfo"
	case ShareNodes:
		return "ShareNodes"
	case AppendEntries:
		return "AppendEntries"
	case AppendEntriesResponse:
		return "AppendEntriesResponse"
	case VoteRequest:
		return "VoteRequest"
	case VoteResponse:
		return "VoteResponse"
	default:
		return "Unknown Socket Type"
	}
}

// 定义一个表示角色的类型
type RoleType byte

func (t RoleType) String() string {
	switch t {
	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Leader:
		return "Leader"
	default:
		return "Unknown Role"
	}
}

// 当前的角色类型
var Role = Follower

// 节点类型
const (
	Follower RoleType = iota
	Candidate
	Leader
)

// 选举相关的一些数据
var (
	HeartbeatTimeoutCh  = make(chan bool, 1) // 当前节点心跳超时的channel，用于刷新心跳超时时间
	VoteSuccessCh       = make(chan bool, 1) // 选举情况channel，表示选举成功还是失败、或者超时
	LeaderSendEntryCh   = make(chan bool, 1) // leader发送了信息的channel，意味着在这个周期内不再需要主动发送心跳
	LeaderAppendSuccess = make(chan bool, 1) // leader复制entries给大部分的follower操作成功
)

const (
	HeartbeatTimeoutMin        = 150 // follower心跳超时阈值的下限
	HeartbeatTimeoutMax        = 300 // follower心跳超时阈值的上限
	ElectionTimeoutMin         = 150 // candidate选举超时阈值的下限
	ElectionTimeoutMax         = 300 // candidate选举超时阈值的上限
	LeaderCycleTimeout         = 60  // leader发送心跳包的周期
	LeaderAppendEntriesTimeout = 50  // AppendEntries超时的时间
)

var (
	CurrentTerm uint32 = 0 // 当前任期
	Votes       uint32 = 0 // 获取选票数
)

// leader节点的id
var LeaderNodeId string

// 投票信息 key为term，value为nodeId
var VoteFor = make(map[uint32]string)

// 一条数据
type Entry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Term  uint32 // 添加该条数据时的任期号
	Index uint32 // 在log中的索引
	Time  uint32 // 创建时间
}

// 已经提交的entry索引
var CommittedIndex = uint32(0)
var AppliedIndex = uint32(0)
