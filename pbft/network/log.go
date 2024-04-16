package network

import (
	"fmt"
	"simple_pbft/pbft/consensus"
)

func LogMsg(msg interface{}) {
	switch msg.(type) {
	case *consensus.RequestMsg:
		reqMsg := msg.(*consensus.RequestMsg)
		fmt.Printf("[REQUEST] ClientID: %s, Timestamp: %d, Operation: %s\n", reqMsg.ClientID, reqMsg.Timestamp, reqMsg.Operation)
	case *consensus.BatchRequestMsg:
		reqMsg := msg.(*consensus.BatchRequestMsg)
		fmt.Printf("[REQUEST] ClientID: %s, Timestamp: %d,SeqID %d\n", reqMsg.ClientID, reqMsg.Timestamp, reqMsg.Requests[0].SequenceID)
	case *consensus.PrePrepareMsg:
		prePrepareMsg := msg.(*consensus.PrePrepareMsg)
		fmt.Printf("[PREPREPARE] ClientID: %s,  SequenceID: %d\n", prePrepareMsg.RequestMsg.ClientID, prePrepareMsg.SequenceID)
	case *consensus.VoteMsg:
		voteMsg := msg.(*consensus.VoteMsg)
		if voteMsg.MsgType == consensus.PrepareMsg {
			fmt.Printf("[PREPARE] NodeID: %s\n", voteMsg.NodeID)
		} else if voteMsg.MsgType == consensus.CommitMsg {
			fmt.Printf("[COMMIT] NodeID: %s\n", voteMsg.NodeID)
		}
	}
}

func LogStage(stage string, isDone bool) {
	if isDone {
		fmt.Printf("[STAGE-DONE] %s\n", stage)
	} else {
		fmt.Printf("[STAGE-BEGIN] %s\n", stage)
	}
}

func ErrMessage(msg interface{}) {
	fmt.Printf("------\n")
	switch msg.(type) {
	case *consensus.PrePrepareMsg:
		prePrepareMsg := msg.(*consensus.PrePrepareMsg)
		fmt.Printf("Error message: [PREPREPARE] ClientID: %s, SequenceID: %d\n", prePrepareMsg.RequestMsg.ClientID, prePrepareMsg.SequenceID)
	case *consensus.VoteMsg:
		voteMsg := msg.(*consensus.VoteMsg)
		if voteMsg.MsgType == consensus.PrepareMsg {
			fmt.Printf("Error message: PREPAREMsg NodeID: %s, SequenceID: %d\n", voteMsg.NodeID, voteMsg.SequenceID)
		} else if voteMsg.MsgType == consensus.CommitMsg {
			fmt.Printf("Error message: COMMITMsg NodeID: %s,  SequenceID: %d\n", voteMsg.NodeID, voteMsg.SequenceID)
			fmt.Printf("Error message: PREPAREMsg NodeID: %s\n", voteMsg.NodeID)
		} else if voteMsg.MsgType == consensus.CommitMsg {
			fmt.Printf("Error message: COMMITMsg NodeID: %s\n", voteMsg.NodeID)
		}
	}
	fmt.Printf("------\n")
}
