// 保存了一些变量

package common

var (
	BuildStamp string // 可执行文件的编译时间
	Version    string // 可执行文件的版本
	GoVersion  string // 编译可执行文件的golang编译器版本
)

// 进程启动的时间戳
var StartingTimeStamp uint32

// 当前节点的一些参数信息，声明时都附带了默认值
var (
	LocalNodeId        = RandomString(10)  // 当前节点id
	Port        uint   = 6300              // 监听的TCP端口
	HTTPPort    uint   = 6200              // HTTP服务监听的端口
	Hosts              = make([]string, 0) // 种子节点
	Quorum      uint32 = 1                 // 投票时所需要的法定人数
	LogLevel           = "debug"           // 日志等级
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

// 定义一个表示角色的类型
type RoleType byte

// 为了打印角色时显示的更加直观
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

// 所有的角色类型
const (
	Follower  RoleType = iota // 跟随者
	Candidate                 // 候选人
	Leader                    // 领导人
)

// 一些用于协程间同步的channel
var (
	HeartbeatTimeoutCh = make(chan bool, 1) // 当前节点心跳超时的channel，表示此周期内已经接收到了心跳，follower不需要超时
	VoteSuccessCh      = make(chan bool, 1) // 选举情况channel，表示选举成功还是失败、或者超时
	LeaderSendEntryCh  = make(chan bool, 1) // leader发送了信息的channel，意味着在这个周期内leader不再需要主动发送心跳
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
	Votes       uint32 = 0 // 作为Candidate所获取的选票数
)

var (
	LeaderNodeId   string                    // leader节点的id
	VoteFor        = make(map[uint32]string) // 投票信息 key为term，value为nodeId，表示在这个任期内投给了谁，可以避免重复投票
	CommittedIndex = uint32(0)               // 已经提交的entry索引
	AppliedIndex   = uint32(0)
)
