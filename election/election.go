package election

import (
	"../common"
	"log"
	"time"
)

func Do() {
	for {
		// follower的超时时间在150ms到300ms之间
		timeout := common.RandomInt(150, 300)
		select {
		case <-common.RoleTimeoutCh:
			if common.LEVEL >= common.DEBUG {
				log.Printf("%s(me) get heartbeat and reset timer\n", common.LocalNodeId)
			}
		case <-time.After(time.Duration(timeout) * time.Millisecond):
			if common.LEVEL >= common.DEBUG {
				log.Printf("%s(me) become candidate\n", common.LocalNodeId)
			}
		}
	}
}
