// 身为leader向所有的follower发送AppendEntries信息

package node

import (
	"../common"
	"../tog"
	"log"
	"sync/atomic"
	"time"
)

// 向指定节点发送数据
func sendData(n Node, data []byte) bool {
	success := n.Conn.Write(data)
	if !success {
		RemoveNodeById(n.NodeId) // 连接出现异常，移除该节点
		return false
	}
	return true
}

// 给所有的节点发送数据
func SendDataToFollowers(nodes []Node, data []byte) {
	for _, n := range nodes {
		if n.Conn == nil {
			continue
		}
		// 并行的发送数据到follower，提升发送效率
		go func(n Node) { // 这里的变量不能使用闭包，否则会有问题
			_ = sendData(n, data)
		}(n)
	}
}

// 发送entries信息给follower，即AppendEntries操作
func SendAppendEntries(appendSuccessChannel chan bool) {
	replicatedNum := uint32(1) // 记录已经成功复制的follower

	for _, n := range GetNodes() {
		if n.Conn == nil {
			if tog.LogLevel(tog.DEBUG) {
				log.Printf("%s's conn is nil\n", n.NodeId)
			}
			continue
		}
		// 这里的变量不能使用闭包，否则会有问题
		go func(n Node) {
			// 如果上次的leader超时了，但是后来follower又返回了数据，则channel不为空
			// 因此为了排除影响，需要清空channel
			for len(n.AppendSuccess) > 0 {
				if tog.LogLevel(tog.DEBUG) {
					log.Printf("Channel AppendSuccess Length: %d\n", len(n.AppendSuccess))
				}
				<-n.AppendSuccess
			}

			entries := common.GetEntries()[n.NextIndex:]
			entriesLength := len(entries)
			if tog.LogLevel(tog.DEBUG) {
				log.Printf(">>>>>>>>>> Starting Send AppendEntries (%s -> %s)\n", common.LocalNodeId, n.NodeId)
				log.Printf("%s(me) Settings for follower: %s, entriesLength: %d and nextIndex: %d\n", common.LocalNodeId, n.NodeId, entriesLength, n.NextIndex)
			}

			// 该follower在leader中所记录的最后一个节点的index
			followerIndex := n.NextIndex - 1

			data := []byte{common.AppendEntries}
			data = append(data, common.Uint32ToBytes(common.CurrentTerm)...)                         // 当前任期
			data = append(data, common.Uint32ToBytes(followerIndex)...)                              // 此follower的最后一个节点
			data = append(data, common.Uint32ToBytes(common.GetEntryByIndex(followerIndex).Term)...) // 此follower的最后一个节点的任期
			data = append(data, common.Uint32ToBytes(common.CommittedIndex)...)                      // leader的commitIndex
			data = append(data, common.Uint32ToBytes(uint32(entriesLength))...)                      // 发送的entries的长度

			commitIndex := uint32(0)
			for i := 0; i < entriesLength; i++ {
				data = append(data, common.EncodeEntry(entries[i])...)
				commitIndex = common.Max(commitIndex, entries[i].Index) // 获取到最大的Index
			}

			// 如果运行中角色发生了变化，那么就不再允许向follower发送数据
			if common.Role != common.Leader {
				if tog.LogLevel(tog.WARN) {
					log.Printf("Can't send AppendEntries to followers, current role is %v", common.Role)
				}
				return
			}

			// 发送appendEntries给follower
			success := sendData(n, data)
			if !success {
				return
			}

			if tog.LogLevel(tog.DEBUG) {
				log.Printf("%s(me) send AppendEntries to %s\n", common.LocalNodeId, n.NodeId)
			}
			select {
			case appendSuccess := <-n.AppendSuccess:
				if tog.LogLevel(tog.DEBUG) {
					log.Printf("%s(me) Get AppendEntries Response from %s: %v\n",
						common.LocalNodeId, n.NodeId, appendSuccess)
				}
				if appendSuccess {
					if entriesLength == 0 {
						// 空的心跳包不需要做任何操作
						if tog.LogLevel(tog.DEBUG) {
							log.Println("Heartbeat AppendEntries Response and nothing to do.")
						}
					} else {
						// Append成功，该follower已经追上leader的entries进度
						UpdateNextIndexByNodeId(n.NodeId, common.GetEntriesLength())
						UpdateMatchIndex(n.NodeId)
						atomic.AddUint32(&replicatedNum, 1)
						if replicatedNum == common.Quorum { // 只触发一次
							appendSuccessChannel <- true        // client返回
							common.CommittedIndex = commitIndex // commit
						}
					}
				} else {
					// 发送失败，减小nextIndex进行重试

					// 回退nextIndex到前一个term的最后一个entry的index
					// 相较于把nextIndex直接减一，这样做更加高效
					var lastTerm = uint32(0)                        // 后一个entry的term
					allEntries := common.GetEntries()[:n.NextIndex] // 剔除当前的nextIndex的后面的内容
					for i := len(allEntries) - 1; i >= 0; i-- {
						e := allEntries[i]
						if lastTerm > e.Term { // 找到当前的term所对应初始index
							n.NextIndex = e.Index + 1
							break
						}
						lastTerm = e.Term
					}

					UpdateNextIndexByNodeId(n.NodeId, n.NextIndex)
				}
			case <-time.After(time.Duration(common.LeaderAppendEntriesTimeout) * time.Millisecond):
				// AppendEntries超时，不需要手动重试，因为leader的下一次心跳
				// 将在 LeaderCycleTimeout - LeaderAppendEntriesTimeout 后发生，心跳会实现下一次重试
				if tog.LogLevel(tog.DEBUG) {
					log.Printf("%s(me) to %s AppendEntries timeout and retry\n", common.LocalNodeId, n.NodeId)
				}
			}
		}(n)
	}
}
