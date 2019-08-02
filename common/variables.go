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
	Quorum      uint      // 所谓的“大多数”
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

var Role = Follower

// 节点类型
const (
	Follower byte = iota
	Candidate
	Leader
)

// 选举相关的一些数据
var (
	HeartbeatTimeoutCh = make(chan bool, 1) // 当前节点心跳超时的channel
	VoteSuccessCh      = make(chan bool, 1) // 选举情况channel
	LeaderSendEntryCh  = make(chan bool, 1) // leader发送了信息的channel
)

const (
	HeartbeatTimeoutMin = 150
	HeartbeatTimeoutMax = 300
	ElectionTimeoutMin  = 150
	ElectionTimeoutMax  = 300
	LeaderCycleTimeout  = 100
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
	Key   string
	Value string
	Term  uint32 // 添加该条数据时的任期号
	Index uint32 // 在log中的索引
}

// 已经提交的entry索引
var CommittedIndex = uint32(0)
var AppliedIndex = uint32(0)

// 所有的数据
var entries = make([]Entry, 0)

// 添加一条数据
func AppendEntry(key, value string) {
	entries = append(entries, Entry{Key: key, Value: value, Term: CurrentTerm, Index: uint32(len(entries))})
}

// 根据key获取entry
func GetEntryByKey(key string) string {
	for i := len(entries); i > 0; i-- {
		if entries[i].Key == key {
			return entries[i].Value
		}
	}
	return ""
}

func GetLastEntry() Entry {
	return entries[len(entries)-1]
}

// 作为leader
var (
	nextIndexMap  = make(map[string]uint32) // 需要发送给指定follower的下一个log entry下标
	matchIndexMap = make(map[string]uint32) // 针对指定follower，已经复制的最大的log entry下标
)

var (
	PrevLogIndex uint32 // 剔除entries最新的log的index
	PrevLogTerm  uint32 // 剔除entries最新的log的term
)

// AppendEntries
var (
	leaderTerm uint32
	leaderId   string

	entries0          []Entry // 发送的entries，可以为空（心跳）
	leaderCommitIndex uint32  // leader的commitIndex
)
