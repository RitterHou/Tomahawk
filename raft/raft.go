package raft

import (
	"../common"
	"../node"
	"../tog"
	"log"
	"time"
)

// 执行Raft协议
func Run() {
	for {
		switch common.Role {
		case common.Follower:
			heartbeatTimeout := common.RandomInt(common.HeartbeatTimeoutMin, common.HeartbeatTimeoutMax)
			select {
			case <-common.HeartbeatTimeoutCh: // 收到了心跳
				if tog.LogLevel(tog.DEBUG) {
					log.Printf("%s(me) get heartbeat and reset timer\n", common.LocalNodeId)
				}
			case <-time.After(time.Duration(heartbeatTimeout) * time.Millisecond): // 没能收到心跳，超时
				// 更新为候选人
				common.ChangeRole(common.Candidate)
				if tog.LogLevel(tog.DEBUG) {
					log.Printf("%s(me) heartbeat timeout and become candidate\n", common.LocalNodeId)
				}
			}
		case common.Candidate: // 最复杂 1.成为leader; 2.成为follower; 3.继续下一轮选举
			common.CurrentTerm += 1
			common.Votes = 1 // 首先投票给自己
			common.VoteFor[common.CurrentTerm] = common.LocalNodeId

			nodes := node.GetNodes()
			if len(nodes) == 1 { // 单个节点的情况
				if common.Votes >= common.Quorum {
					common.VoteSuccessCh <- true
				} else {
					common.VoteSuccessCh <- false
				}
			} else {
				// 发送选举请求
				data := append([]byte{common.VoteRequest}, common.Uint32ToBytes(common.CurrentTerm)...)
				lastEntry := common.GetLastEntry()
				data = append(data, common.Uint32ToBytes(lastEntry.Index)...) // Index
				data = append(data, common.Uint32ToBytes(lastEntry.Term)...)  // Term
				node.SendDataToFollowers(nodes, data)
			}

			electionTimeout := common.RandomInt(common.ElectionTimeoutMin, common.ElectionTimeoutMax)
			select {
			case success := <-common.VoteSuccessCh:
				if success { // 选举成功
					common.ChangeRole(common.Leader)
					if tog.LogLevel(tog.DEBUG) {
						log.Printf("%s(me) Vote success and become leader\n", common.LocalNodeId)
					}

					// 刚刚当选的时候添加一条 no operation 的日志记录
					noOp := common.Entry{Key: "", Value: ""}
					common.AppendEntryList([]common.Entry{noOp})

					// 当一个领导人刚获得权力的时候，他初始化所有的nextIndex值为自己的最后一条日志的index加1
					for _, n := range node.GetNodes() {
						nextIndex := common.GetEntriesLength()
						node.UpdateNextIndexByNodeId(n.NodeId, nextIndex)
					}

					// 选举成功立即发送心跳，防止follower再次超时
					// common.LeaderSendEntryCh <- true, leader在这个周期再超时一次也没有关系
					node.SendAppendEntries()
				} else { // 选举失败
					common.ChangeRole(common.Follower)
					if tog.LogLevel(tog.DEBUG) {
						log.Printf("%s(me) Vote failed and becmoe follower\n", common.LocalNodeId)
					}
				}
			case <-time.After(time.Duration(electionTimeout) * time.Millisecond): // 选举超时
				if tog.LogLevel(tog.DEBUG) {
					log.Printf("%s(me) this turn election failed, next turn election will start soon\n",
						common.LocalNodeId)
				}
			}
		case common.Leader:
			select {
			case <-common.LeaderSendEntryCh: // 发送了心跳
				if tog.LogLevel(tog.DEBUG) {
					log.Printf("%s(me) leader has sent data to followers\n", common.LocalNodeId)
				}
			case <-time.After(common.LeaderCycleTimeout * time.Millisecond): // 未发送心跳导致超时
				if tog.LogLevel(tog.DEBUG) {
					log.Printf("%s(me) leader not send data, send empty data as heartbeat\n", common.LocalNodeId)
				}
				// 超时则发送心跳
				node.SendAppendEntries()
			}
		}
	}
}
