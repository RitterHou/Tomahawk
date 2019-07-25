package election

import (
	"../common"
	"../node"
	"log"
	"time"
)

func Do() {
	for {
		switch common.Role {
		case common.Follower:
			heartbeatTimeout := common.RandomInt(common.HeartbeatTimeoutMin, common.HeartbeatTimeoutMax)
			select {
			case <-common.HeartbeatTimeoutCh:
				if common.LEVEL >= common.DEBUG {
					log.Printf("%s(me) get heartbeat and reset timer\n", common.LocalNodeId)
				}
			case <-time.After(time.Duration(heartbeatTimeout) * time.Millisecond):
				common.Role = common.Candidate // 更新为候选人
				if common.LEVEL >= common.DEBUG {
					log.Printf("%s(me) become candidate\n", common.LocalNodeId)
				}
			}
		case common.Candidate: // 最复杂 1.成为leader; 2.成为follower; 3.继续下一轮选举
			electionTimeout := common.RandomInt(common.ElectionTimeoutMin, common.ElectionTimeoutMax)
			select {
			case success := <-common.VoteSuccessCh:
				if success {
					common.Role = common.Leader
				} else {
					common.Role = common.Follower
				}
			case <-time.After(time.Duration(electionTimeout) * time.Millisecond):
				if common.LEVEL >= common.DEBUG {
					log.Printf("%s(me) this turn election failed, next turn election will start soon\n",
						common.LocalNodeId)
				}
			}
		case common.Leader:
			select {
			case <-common.LeaderSendEntryCh:
				if common.LEVEL >= common.DEBUG {
					log.Printf("%s(me) leader has sent data to followers\n", common.LocalNodeId)
				}
			case <-time.After(common.LeaderCycleTimeout * time.Millisecond):
				if common.LEVEL >= common.DEBUG {
					log.Printf("%s(me) leader not send data, send empty data as heartbeat\n", common.LocalNodeId)
				}
				// 发送心跳数据，防止follower超时
				nodes := node.GetNodes()
				for _, n := range nodes {
					if n.Conn != nil {
						_, err := n.Conn.Write([]byte{common.AppendEntries})
						if err != nil {
							log.Fatal(err)
						}
					}
				}
			}
		}
	}
}
