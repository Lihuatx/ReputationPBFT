package network

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"simple_pbft/pbft/consensus"
	"time"
	"unicode"
)

type Node struct {
	NodeID        string
	NodeTable     map[string]map[string]string // key=nodeID, value=url
	View          *View
	CurrentState  *consensus.State
	CommittedMsgs []*consensus.RequestMsg // kinda block.
	MsgBuffer     *MsgBuffer
	MsgEntrance   chan interface{}
	MsgDelivery   chan interface{}
	Alarm         chan bool

	// 全局消息日志和临时消息缓冲区
	GlobalLog    *consensus.GlobalLog
	GlobalBuffer *GlobalBuffer
	//RSA私钥
	rsaPrivKey []byte
	//RSA公钥
	rsaPubKey []byte

	//所属集群
	ClusterName string

	//全局消息接受通道和处理通道
	MsgGlobal         chan interface{}
	MsgGlobalDelivery chan interface{}
}

type GlobalBuffer struct {
	ReqMsg       []*consensus.GlobalShareMsg //其他集群的请求消息缓存
	consensusMsg []*consensus.LocalMsg       //本地节点的全局共识消息缓存
}

type MsgBuffer struct {
	ReqMsgs        []*consensus.RequestMsg
	PrePrepareMsgs []*consensus.PrePrepareMsg
	PrepareMsgs    []*consensus.VoteMsg
	CommitMsgs     []*consensus.VoteMsg
}

type View struct {
	ID      int64
	Primary string
}

var PrimaryNode = map[string]string{
	"N": "N0",
	"M": "M0",
	"P": "P0",
}

func getLastDigit(input string) (rune, bool) {
	var lastDigit rune
	found := false
	for _, char := range input {
		if unicode.IsDigit(char) {
			lastDigit = char
			found = true
		}
	}
	return lastDigit, found
}

const ResolvingTimeDuration = time.Millisecond * 1000 // 1 second.

func NewNode(nodeID string, clusterName string) *Node {
	const viewID = 10000000000 // temporary.
	consensus.GlobalViewID = viewID
	node := &Node{
		// Hard-coded for test.
		NodeID: nodeID,
		NodeTable: map[string]map[string]string{
			"N": {
				"N0": "localhost:1111",
				"N1": "localhost:1112",
				"N2": "localhost:1113",
				"N3": "localhost:1114",
				"N4": "localhost:1115",
			},
			"M": {
				"M0": "localhost:1116",
				"M1": "localhost:1117",
				"M2": "localhost:1118",
				"M3": "localhost:1119",
				"M4": "localhost:1120",
			},
			"P": {
				"P0": "localhost:1121",
				"P1": "localhost:1122",
				"P2": "localhost:1123",
				"P3": "localhost:1124",
				"P4": "localhost:1125",
			},
		},
		View: &View{
			ID:      viewID,
			Primary: PrimaryNode[clusterName],
		},

		// Consensus-related struct
		CurrentState:  nil,
		CommittedMsgs: make([]*consensus.RequestMsg, 0),
		MsgBuffer: &MsgBuffer{
			ReqMsgs:        make([]*consensus.RequestMsg, 0),
			PrePrepareMsgs: make([]*consensus.PrePrepareMsg, 0),
			PrepareMsgs:    make([]*consensus.VoteMsg, 0),
			CommitMsgs:     make([]*consensus.VoteMsg, 0),
		},
		GlobalLog: &consensus.GlobalLog{
			MsgLogs:  make(map[string]map[int64]*consensus.RequestMsg),
			VoteLogs: make(map[string]map[int64]string),
		},
		GlobalBuffer: &GlobalBuffer{
			ReqMsg:       make([]*consensus.GlobalShareMsg, 0),
			consensusMsg: make([]*consensus.LocalMsg, 0),
		},

		// Channels
		MsgEntrance:       make(chan interface{}, 5),
		MsgDelivery:       make(chan interface{}, 5),
		MsgGlobal:         make(chan interface{}, 5),
		MsgGlobalDelivery: make(chan interface{}, 5),
		Alarm:             make(chan bool),

		// 所属集群
		ClusterName: clusterName,
	}

	node.rsaPubKey = node.getPubKey(clusterName, nodeID)
	node.rsaPrivKey = node.getPivKey(clusterName, nodeID)

	// Start message dispatcher
	go node.dispatchMsg()

	// Start alarm trigger
	go node.alarmToDispatcher()

	// Start message resolver
	go node.resolveMsg()

	// Start solve Global message
	go node.resolveGlobalMsg()

	return node
}

func (node *Node) Broadcast(cluster string, msg interface{}, path string) map[string]error {
	errorMap := make(map[string]error)

	for nodeID, url := range node.NodeTable[cluster] {
		if nodeID == node.NodeID {
			continue
		}

		jsonMsg, err := json.Marshal(msg)
		if err != nil {
			errorMap[nodeID] = err
			continue
		}
		fmt.Printf("Send to %s Size of JSON message: %d bytes\n", url+path, len(jsonMsg))
		send(url+path, jsonMsg)
	}

	if len(errorMap) == 0 {
		return nil
	} else {
		return errorMap
	}
}

// ShareLocalConsensus 本地达成共识后，主节点调用当前函数发送信息给其他集群的f+1个节点
func (node *Node) ShareLocalConsensus(msg *consensus.GlobalShareMsg, path string) error {
	errorMap := make(map[string]map[string]error)

	for cluster, nodeMsg := range node.NodeTable {
		if cluster == node.ClusterName {
			continue
		}
		_cnt := 0
		for nodeID, url := range nodeMsg {
			_cnt++
			if _cnt > 2 { // f=1 f+1 = 2
				continue
			}
			jsonMsg, err := json.Marshal(msg)
			if err != nil {
				errorMap[cluster][nodeID] = err
				continue
			}
			fmt.Printf("Send to %s Size of JSON message: %d bytes\n", url+path, len(jsonMsg))
			send(url+path, jsonMsg)
		}
	}
	return nil
}

func (node *Node) Reply(msg *consensus.ReplyMsg, cluster string) error {
	// Print all committed messages.
	for _, value := range node.CommittedMsgs {
		fmt.Printf("Committed value: %s, %d, %s, %d", value.ClientID, value.Timestamp, value.Operation, value.SequenceID)
	}
	fmt.Print("\n")

	node.View.ID++

	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// 系统中没有设置用户，reply消息直接发送给主节点
	send(node.NodeTable[cluster][node.View.Primary]+"/reply", jsonMsg)

	return nil
}

// GetReq can be called when the node's CurrentState is nil.
// Consensus start procedure for the Primary.
func (node *Node) GetReq(reqMsg *consensus.RequestMsg) error {
	LogMsg(reqMsg)

	// Create a new state for the new consensus.
	err := node.createStateForNewConsensus()
	if err != nil {
		return err
	}

	// Start the consensus process.
	prePrepareMsg, err := node.CurrentState.StartConsensus(reqMsg)
	if err != nil {
		return err
	}

	// 主节点对消息摘要进行签名
	digestByte, _ := hex.DecodeString(prePrepareMsg.Digest)
	signInfo := node.RsaSignWithSha256(digestByte, node.rsaPrivKey)
	prePrepareMsg.Sign = signInfo

	LogStage(fmt.Sprintf("Consensus Process (ViewID:%d)", node.CurrentState.ViewID), false)

	// Send getPrePrepare message
	if prePrepareMsg != nil {
		// 附加主节点ID,用于数字签名验证
		prePrepareMsg.NodeID = node.NodeID

		node.Broadcast(node.ClusterName, prePrepareMsg, "/preprepare")
		LogStage("Pre-prepare", true)
	}

	return nil
}

// GetPrePrepare can be called when the node's CurrentState is nil.
// Consensus start procedure for normal participants.
func (node *Node) GetPrePrepare(prePrepareMsg *consensus.PrePrepareMsg) error {
	LogMsg(prePrepareMsg)

	// Create a new state for the new consensus.
	err := node.createStateForNewConsensus()
	if err != nil {
		return err
	}
	// fmt.Printf("get Pre\n")
	digest, _ := hex.DecodeString(prePrepareMsg.Digest)
	if !node.RsaVerySignWithSha256(digest, prePrepareMsg.Sign, node.getPubKey(node.ClusterName, prePrepareMsg.NodeID)) {
		fmt.Println("节点签名验证失败！,拒绝执行Preprepare")
	}
	prePareMsg, err := node.CurrentState.PrePrepare(prePrepareMsg)
	if err != nil {
		return err
	}

	if prePareMsg != nil {
		// Attach node ID to the message 同时对摘要签名
		prePareMsg.NodeID = node.NodeID
		signInfo := node.RsaSignWithSha256(digest, node.rsaPrivKey)
		prePareMsg.Sign = signInfo

		LogStage("Pre-prepare", true)
		node.Broadcast(node.ClusterName, prePareMsg, "/prepare")
		LogStage("Prepare", false)
	}

	return nil
}

func (node *Node) GetPrepare(prepareMsg *consensus.VoteMsg) error {
	LogMsg(prepareMsg)

	digest, _ := hex.DecodeString(prepareMsg.Digest)
	if !node.RsaVerySignWithSha256(digest, prepareMsg.Sign, node.getPubKey(node.ClusterName, prepareMsg.NodeID)) {
		fmt.Println("节点签名验证失败！,拒绝执行prepare")
	}

	commitMsg, err := node.CurrentState.Prepare(prepareMsg)
	if err != nil {
		ErrMessage(prepareMsg)
		return err
	}
	if commitMsg != nil {
		// Attach node ID to the message 同时对摘要签名
		commitMsg.NodeID = node.NodeID
		signInfo := node.RsaSignWithSha256(digest, node.rsaPrivKey)
		commitMsg.Sign = signInfo

		LogStage("Prepare", true)
		node.Broadcast(node.ClusterName, commitMsg, "/commit")
		LogStage("Commit", false)
	}

	return nil
}

func (node *Node) GetCommit(commitMsg *consensus.VoteMsg) error {
	// 当节点已经完成Committed阶段后就停止接收其他节点的Committed消息
	if node.CurrentState.CurrentStage == consensus.Committed {
		return nil
	}

	LogMsg(commitMsg)

	digest, _ := hex.DecodeString(commitMsg.Digest)
	if !node.RsaVerySignWithSha256(digest, commitMsg.Sign, node.getPubKey(node.ClusterName, commitMsg.NodeID)) {
		fmt.Println("节点签名验证失败！,拒绝执行commit")
	}

	replyMsg, committedMsg, err := node.CurrentState.Commit(commitMsg)
	if err != nil {
		ErrMessage(committedMsg)
		return err
	}

	if replyMsg != nil {

		if committedMsg == nil {
			return errors.New("committed message is nil, even though the reply message is not nil")
		}

		// Attach node ID to the message
		replyMsg.NodeID = node.NodeID

		// Save the last version of committed messages to node.
		node.CommittedMsgs = append(node.CommittedMsgs, committedMsg)

		LogStage("Commit", true)
		fmt.Printf("ViewID :%d 达成本地共识，存入待执行缓存池\n")

		// node.Reply(replyMsg, node.ClusterName)
		// LogStage("Reply", true)
		if node.GlobalLog.MsgLogs[node.ClusterName] == nil {
			node.GlobalLog.MsgLogs[node.ClusterName] = make(map[int64]*consensus.RequestMsg)
		}
		// Append msg to its logs
		node.GlobalLog.MsgLogs[node.ClusterName][node.View.ID] = committedMsg

		if node.NodeID == node.View.Primary { // 本地共识结束后，主节点将本地达成共识的请求发送至其他集群的主节点
			fmt.Printf("send consensus to Global\n")
			// 获取消息摘要
			msg, err := json.Marshal(committedMsg)
			if err != nil {
				return err
			}
			digest := consensus.Hash(msg)

			// 节点对消息摘要进行签名
			digestByte, _ := hex.DecodeString(digest)
			signInfo := node.RsaSignWithSha256(digestByte, node.rsaPrivKey)

			GlobalShareMsg := new(consensus.GlobalShareMsg)
			GlobalShareMsg.RequestMsg = committedMsg
			GlobalShareMsg.NodeID = node.NodeID
			GlobalShareMsg.Sign = signInfo
			GlobalShareMsg.Digest = digest
			GlobalShareMsg.Cluster = node.ClusterName
			GlobalShareMsg.ViewID = node.View.ID
			node.ShareLocalConsensus(GlobalShareMsg, "/global")
		}
	}

	node.View.ID++

	return nil
}

func (node *Node) GetReply(msg *consensus.ReplyMsg) {
	fmt.Printf("Result: %s by %s\n", msg.Result, msg.NodeID)
}

func (node *Node) createStateForNewConsensus() error {
	// Check if there is an ongoing consensus process.
	if node.CurrentState != nil {
		if node.CurrentState.CurrentStage != consensus.Committed {
			return errors.New("another consensus is ongoing")
		}
	}

	// Get the last sequence ID
	var lastSequenceID int64
	if len(node.CommittedMsgs) == 0 {
		lastSequenceID = -1
	} else {
		lastSequenceID = node.CommittedMsgs[len(node.CommittedMsgs)-1].SequenceID
	}

	// Create a new state for this new consensus process in the Primary
	node.CurrentState = consensus.CreateState(node.View.ID, lastSequenceID)

	LogStage("Create the replica status", true)

	return nil
}

func (node *Node) dispatchMsg() {
	for {
		select {
		case msg := <-node.MsgEntrance:
			err := node.routeMsg(msg)
			if err != nil {
				fmt.Println(err)
				// TODO: send err to ErrorChannel
			}
		case <-node.Alarm:
			err := node.routeMsgWhenAlarmed()
			if err != nil {
				fmt.Println(err)
				// TODO: send err to ErrorChannel
			}
		case msg := <-node.MsgGlobal:
			err := node.routeGlobalMsg(msg)
			if err != nil {
				fmt.Println(err)
				// TODO: send err to ErrorChannel
			}
		}
	}
}

func (node *Node) routeGlobalMsg(msg interface{}) []error {
	switch m := msg.(type) {
	case *consensus.GlobalShareMsg:
		fmt.Printf("---- Receive the Global Consensus from %s\n", m.NodeID)
		// Copy buffered messages first.
		msgs := make([]*consensus.GlobalShareMsg, len(node.GlobalBuffer.ReqMsg))
		copy(msgs, node.GlobalBuffer.ReqMsg)

		// Append a newly arrived message.
		msgs = append(msgs, msg.(*consensus.GlobalShareMsg))

		// Empty the buffer.
		node.GlobalBuffer.ReqMsg = make([]*consensus.GlobalShareMsg, 0)

		// Send messages.
		node.MsgGlobalDelivery <- msgs
	case *consensus.LocalMsg:
		fmt.Printf("---- Receive the Local Consensus from %s for cluster %s\n", m.NodeID, m.GlobalShareMsg.Cluster)
		// Copy buffered messages first.
		msgs := make([]*consensus.LocalMsg, len(node.GlobalBuffer.consensusMsg))
		copy(msgs, node.GlobalBuffer.consensusMsg)

		// Append a newly arrived message.
		msgs = append(msgs, msg.(*consensus.LocalMsg))

		// Empty the buffer.
		node.GlobalBuffer.consensusMsg = make([]*consensus.LocalMsg, 0)

		// Send messages.
		node.MsgGlobalDelivery <- msgs
	}

	return nil
}

func (node *Node) routeMsg(msg interface{}) []error {
	switch msg.(type) {
	case *consensus.RequestMsg:
		if node.CurrentState == nil || node.CurrentState.CurrentStage == consensus.Committed { //一开始没有进行共识的时候，此时 currentstate 为nil
			if len(node.MsgBuffer.ReqMsgs) > 0 { //保证每次只从buffer中取出一个request消息给主节点
				// 取出第一个消息
				firstMsg := node.MsgBuffer.ReqMsgs[0]

				// 将剩余的消息保留在缓冲区中
				node.MsgBuffer.ReqMsgs = node.MsgBuffer.ReqMsgs[1:]

				// 创建一个新的消息切片并将取出的第一个消息加入其中
				msgs := []*consensus.RequestMsg{firstMsg}

				// 如果有新到达的消息，将其追加到buffer中
				if msg != nil {
					node.MsgBuffer.ReqMsgs = append(node.MsgBuffer.ReqMsgs, msg.(*consensus.RequestMsg))
				}
				// 发送消息
				node.MsgDelivery <- msgs
			} else {
				// Copy buffered messages first.
				msgs := make([]*consensus.RequestMsg, len(node.MsgBuffer.ReqMsgs))

				// Append a newly arrived message.
				msgs = append(msgs, msg.(*consensus.RequestMsg))

				// Empty the buffer.
				node.MsgBuffer.ReqMsgs = make([]*consensus.RequestMsg, 0)

				// Send messages.
				node.MsgDelivery <- msgs
			}
		} else {
			node.MsgBuffer.ReqMsgs = append(node.MsgBuffer.ReqMsgs, msg.(*consensus.RequestMsg))
		}
	case *consensus.PrePrepareMsg:
		if node.CurrentState == nil || node.CurrentState.CurrentStage == consensus.Committed {
			// Copy buffered messages first.
			msgs := make([]*consensus.PrePrepareMsg, len(node.MsgBuffer.PrePrepareMsgs))
			copy(msgs, node.MsgBuffer.PrePrepareMsgs)

			// Append a newly arrived message.
			msgs = append(msgs, msg.(*consensus.PrePrepareMsg))

			// Empty the buffer.
			node.MsgBuffer.PrePrepareMsgs = make([]*consensus.PrePrepareMsg, 0)

			// Send messages.
			node.MsgDelivery <- msgs
		} else {
			node.MsgBuffer.PrePrepareMsgs = append(node.MsgBuffer.PrePrepareMsgs, msg.(*consensus.PrePrepareMsg))
		}
	case *consensus.VoteMsg:
		if msg.(*consensus.VoteMsg).MsgType == consensus.PrepareMsg {
			// if node.CurrentState == nil || node.CurrentState.CurrentStage != consensus.PrePrepared
			// 这样的写法会导致当当前节点已经收到2f个节点进入committed阶段时，就会把后来收到的Preprepare消息放到缓冲区中，
			// 这样在下次共识又到prePrepare阶段时就会先去处理上一轮共识的prePrepare协议！
			if node.CurrentState == nil || node.CurrentState.CurrentStage != consensus.PrePrepared {
				node.MsgBuffer.PrepareMsgs = append(node.MsgBuffer.PrepareMsgs, msg.(*consensus.VoteMsg))
			} else {
				// Copy buffered messages first.
				msgs := make([]*consensus.VoteMsg, len(node.MsgBuffer.PrepareMsgs))
				copy(msgs, node.MsgBuffer.PrepareMsgs)

				// Append a newly arrived message.
				msgs = append(msgs, msg.(*consensus.VoteMsg))

				// Empty the buffer.
				node.MsgBuffer.PrepareMsgs = make([]*consensus.VoteMsg, 0)

				// Send messages.
				node.MsgDelivery <- msgs
			}
		} else if msg.(*consensus.VoteMsg).MsgType == consensus.CommitMsg {
			if node.CurrentState == nil || node.CurrentState.CurrentStage != consensus.Prepared {
				node.MsgBuffer.CommitMsgs = append(node.MsgBuffer.CommitMsgs, msg.(*consensus.VoteMsg))
			} else {
				// Copy buffered messages first.
				msgs := make([]*consensus.VoteMsg, len(node.MsgBuffer.CommitMsgs))
				copy(msgs, node.MsgBuffer.CommitMsgs)

				// Append a newly arrived message.
				msgs = append(msgs, msg.(*consensus.VoteMsg))

				// Empty the buffer.
				node.MsgBuffer.CommitMsgs = make([]*consensus.VoteMsg, 0)

				// Send messages.
				node.MsgDelivery <- msgs
			}
		}
	}

	return nil
}

func (node *Node) routeMsgWhenAlarmed() []error {
	if node.CurrentState == nil || node.CurrentState.CurrentStage == consensus.Committed {
		// Check ReqMsgs, send them.
		if len(node.MsgBuffer.ReqMsgs) != 0 {
			msgs := make([]*consensus.RequestMsg, len(node.MsgBuffer.ReqMsgs))
			copy(msgs, node.MsgBuffer.ReqMsgs)

			node.MsgDelivery <- msgs
		}

		// Check PrePrepareMsgs, send them.
		if len(node.MsgBuffer.PrePrepareMsgs) != 0 {
			msgs := make([]*consensus.PrePrepareMsg, len(node.MsgBuffer.PrePrepareMsgs))
			copy(msgs, node.MsgBuffer.PrePrepareMsgs)

			node.MsgDelivery <- msgs
		}
	} else {
		switch node.CurrentState.CurrentStage {
		case consensus.PrePrepared:
			// Check PrepareMsgs, send them.
			if len(node.MsgBuffer.PrepareMsgs) != 0 {
				msgs := make([]*consensus.VoteMsg, len(node.MsgBuffer.PrepareMsgs))
				copy(msgs, node.MsgBuffer.PrepareMsgs)

				node.MsgDelivery <- msgs
			}
		case consensus.Prepared:
			// Check CommitMsgs, send them.
			if len(node.MsgBuffer.CommitMsgs) != 0 {
				msgs := make([]*consensus.VoteMsg, len(node.MsgBuffer.CommitMsgs))
				copy(msgs, node.MsgBuffer.CommitMsgs)

				node.MsgDelivery <- msgs
			}
		}
	}

	return nil
}

func (node *Node) resolveGlobalMsg() {
	for {
		msg := <-node.MsgGlobalDelivery
		switch msg.(type) {
		case []*consensus.GlobalShareMsg:
			errs := node.resolveGlobalShareMsg(msg.([]*consensus.GlobalShareMsg))
			if len(errs) != 0 {
				for _, err := range errs {
					fmt.Println(err)
				}
				// TODO: send err to ErrorChannel
			}
		case []*consensus.LocalMsg:
			errs := node.resolveLocalMsg(msg.([]*consensus.LocalMsg))
			if len(errs) != 0 {
				for _, err := range errs {
					fmt.Println(err)
				}
				// TODO: send err to ErrorChannel
			}
		}
	}
}

func (node *Node) resolveMsg() {
	for {
		// Get buffered messages from the dispatcher.
		msgs := <-node.MsgDelivery
		switch msgs.(type) {
		case []*consensus.RequestMsg:
			errs := node.resolveRequestMsg(msgs.([]*consensus.RequestMsg))
			if len(errs) != 0 {
				for _, err := range errs {
					fmt.Println(err)
				}
				// TODO: send err to ErrorChannel
			}
		case []*consensus.PrePrepareMsg:
			errs := node.resolvePrePrepareMsg(msgs.([]*consensus.PrePrepareMsg))
			if len(errs) != 0 {
				for _, err := range errs {
					fmt.Println(err)
				}
				// TODO: send err to ErrorChannel
			}
		case []*consensus.VoteMsg:
			voteMsgs := msgs.([]*consensus.VoteMsg)
			if len(voteMsgs) == 0 {
				break
			}

			if voteMsgs[0].MsgType == consensus.PrepareMsg {
				errs := node.resolvePrepareMsg(voteMsgs)
				if len(errs) != 0 {
					for _, err := range errs {
						fmt.Println(err)
					}
					// TODO: send err to ErrorChannel
				}
			} else if voteMsgs[0].MsgType == consensus.CommitMsg {
				errs := node.resolveCommitMsg(voteMsgs)
				if len(errs) != 0 {
					for _, err := range errs {
						fmt.Println(err)
					}
					// TODO: send err to ErrorChannel
				}

			}
		}
	}
}

func (node *Node) alarmToDispatcher() {
	for {
		time.Sleep(ResolvingTimeDuration)
		node.Alarm <- true
	}
}

func (node *Node) resolveRequestMsg(msgs []*consensus.RequestMsg) []error {
	errs := make([]error, 0)

	// Resolve messages
	fmt.Printf("len RequestMsg msg %d\n", len(msgs))

	for _, reqMsg := range msgs {
		err := node.GetReq(reqMsg)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		return errs
	}

	return nil
}

func (node *Node) resolveGlobalShareMsg(msgs []*consensus.GlobalShareMsg) []error {
	errs := make([]error, 0)

	// Resolve messages
	fmt.Printf("len GlobalShareMsg msg %d\n", len(msgs))

	for _, reqMsg := range msgs {
		// 收到其他组的消息，转发给本地节点
		err := node.ShareGlobalMsgToLocal(reqMsg)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		return errs
	}

	return nil
}

func (node *Node) resolveLocalMsg(msgs []*consensus.LocalMsg) []error {
	errs := make([]error, 0)

	// Resolve messages
	fmt.Printf("len GlobalShareMsg msg %d\n", len(msgs))

	for _, reqMsg := range msgs {

		// 收到本地节点发来的全局共识消息，投票
		err := node.CommitGlobalMsgToLocal(reqMsg)
		if err != nil {
			errs = append(errs, err)
		}

	}

	if len(errs) != 0 {
		return errs
	}

	return nil
}

func calculateVote(q []bool) int {
	voteNum := 0
	for _, vote := range q {
		if vote {
			voteNum++
		}
	}
	return voteNum
}

func (node *Node) GlobalConsensus(msg *consensus.LocalMsg) (*consensus.ReplyMsg, *consensus.RequestMsg, error) {

	voteID, _ := getLastDigit(msg.NodeID)
	// Append msg to its logs

	// Print current voting status
	fmt.Printf("[Global-Commit-Vote For %s]: %d\n", msg.GlobalShareMsg.Cluster, len(node.GlobalLog.MsgLogs[msg.GlobalShareMsg.Cluster]))

	if calculateVote(msg.GlobalShareMsg.VoteNode) > 2 {
		// This node executes the requested operation locally and gets the result.
		result := "Executed"

		// Change the stage to prepared.
		return &consensus.ReplyMsg{
			ViewID:    msg.GlobalShareMsg.ViewID,
			Timestamp: 0,
			ClientID:  msg.GlobalShareMsg.RequestMsg.ClientID,
			Result:    result,
		}, msg.GlobalShareMsg.RequestMsg, nil
	}

	return nil, nil, nil
}

// CommitGlobalMsgToLocal 收到本地节点发来的全局共识消息，投票
func (node *Node) CommitGlobalMsgToLocal(reqMsg *consensus.LocalMsg) error {
	// LogMsg(reqMsg)

	// LogStage(fmt.Sprintf("Consensus Process (ViewID:%d)", node.CurrentState.ViewID), false)
	digest, _ := hex.DecodeString(reqMsg.GlobalShareMsg.Digest)
	if !node.RsaVerySignWithSha256(digest, reqMsg.Sign, node.getPubKey(node.ClusterName, reqMsg.NodeID)) {
		fmt.Println("节点签名验证失败！,拒绝执行Global commit")
	}

	// 如果是首次接受到这个集群的消息，广播给本地集群中的其他节点
	_, ok := node.GlobalLog.MsgLogs[reqMsg.GlobalShareMsg.Cluster]
	if !ok {
		fmt.Printf("首次接收到全局共识验证，广播给本地集群")
		// 附加节点ID,用于数字签名验证
		// 节点对消息摘要进行签名
		digestByte, _ := hex.DecodeString(reqMsg.GlobalShareMsg.Digest)
		signInfo := node.RsaSignWithSha256(digestByte, node.rsaPrivKey)
		sendmsg := &consensus.LocalMsg{
			Sign:           signInfo,
			NodeID:         node.NodeID,
			GlobalShareMsg: reqMsg.GlobalShareMsg,
		}

		node.Broadcast(node.ClusterName, sendmsg, "/GlobalToLocal")
		// 存入待执行缓冲区
		if node.GlobalLog.MsgLogs[reqMsg.GlobalShareMsg.Cluster] == nil {
			node.GlobalLog.MsgLogs[reqMsg.GlobalShareMsg.Cluster] = make(map[int64]*consensus.RequestMsg)
		}
		// Append msg to its logs
		node.GlobalLog.MsgLogs[reqMsg.GlobalShareMsg.Cluster][reqMsg.GlobalShareMsg.ViewID] = reqMsg.GlobalShareMsg.RequestMsg
	} else { // 如果是首次接受到这个集群的这个viewID的消息，广播给本地集群中的其他节点
		_, ok := node.GlobalLog.MsgLogs[reqMsg.GlobalShareMsg.Cluster][reqMsg.GlobalShareMsg.ViewID]
		if !ok {
		}
	}

	// GlobalConsensus 会将msg存入MsgLogs中
	replyMsg, committedMsg, err := node.GlobalConsensus(reqMsg)
	if err != nil {
		ErrMessage(committedMsg)
		return err
	}

	if replyMsg != nil {
		if committedMsg == nil {
			return errors.New("committed message is nil, even though the reply message is not nil")
		}

		// Attach node ID to the message
		replyMsg.NodeID = node.NodeID

		// Save the last version of committed messages to node.
		node.CommittedMsgs = append(node.CommittedMsgs, committedMsg)

		LogStage("Overall consensus", true)
		node.Reply(replyMsg, node.ClusterName)
		LogStage("Reply\n", true)
	}

	return nil
}

func (node *Node) ShareGlobalMsgToLocal(reqMsg *consensus.GlobalShareMsg) error {
	// LogMsg(reqMsg)
	// LogStage(fmt.Sprintf("Consensus Process (ViewID:%d)", node.CurrentState.ViewID), false)
	digest, _ := hex.DecodeString(reqMsg.Digest)
	if !node.RsaVerySignWithSha256(digest, reqMsg.Sign, node.getPubKey(reqMsg.Cluster, reqMsg.NodeID)) {
		fmt.Println("节点签名验证失败！,拒绝执行Global commit")
	}

	// 节点对消息摘要进行签名
	signInfo := node.RsaSignWithSha256(digest, node.rsaPrivKey)
	// LogStage(fmt.Sprintf("Consensus Process (ViewID:%d)", node.CurrentState.ViewID), false)

	// Send getPrePrepare message

	// 附加节点ID,用于数字签名验证
	sendMsg := &consensus.LocalMsg{
		Sign:           signInfo,
		NodeID:         node.NodeID,
		GlobalShareMsg: reqMsg,
	}

	// 将消息存入log中
	if node.GlobalLog.MsgLogs[reqMsg.Cluster] == nil {
		node.GlobalLog.MsgLogs[reqMsg.Cluster] = make(map[int64]*consensus.RequestMsg)
	}
	node.GlobalLog.MsgLogs[reqMsg.Cluster][reqMsg.ViewID] = reqMsg.RequestMsg

	node.Broadcast(node.ClusterName, sendMsg, "/GlobalToLocal")
	LogStage("GlobalToLocal", true)

	return nil
}

func (node *Node) resolvePrePrepareMsg(msgs []*consensus.PrePrepareMsg) []error {
	errs := make([]error, 0)

	// Resolve messages
	// 从下标num_of_event_to_resolve开始执行，之前执行过的PrePrepareMsg不需要再执行
	fmt.Printf("len PrePrepareMsg msg %d\n", len(msgs))

	for _, prePrepareMsg := range msgs {
		if prePrepareMsg.ViewID != node.View.ID {
			continue
		}
		err := node.GetPrePrepare(prePrepareMsg)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		return errs
	}

	return nil
}

func (node *Node) resolvePrepareMsg(msgs []*consensus.VoteMsg) []error {
	errs := make([]error, 0)

	// Resolve messages
	fmt.Printf("len PrepareMsg msg %d\n", len(msgs))
	for _, prepareMsg := range msgs {
		if prepareMsg.ViewID != node.View.ID {
			continue
		}
		err := node.GetPrepare(prepareMsg)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		return errs
	}

	return nil
}

func (node *Node) resolveCommitMsg(msgs []*consensus.VoteMsg) []error {
	errs := make([]error, 0)

	// Resolve messages
	fmt.Printf("len CommitMsg msg %d\n", len(msgs))

	for _, commitMsg := range msgs {
		if commitMsg.ViewID != node.View.ID {
			continue
		}
		err := node.GetCommit(commitMsg)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		return errs
	}

	return nil
}

// 传入节点编号， 获取对应的公钥
func (node *Node) getPubKey(ClusterName string, nodeID string) []byte {
	key, err := ioutil.ReadFile("Keys/" + ClusterName + "/" + nodeID + "/" + nodeID + "_RSA_PUB")
	if err != nil {
		log.Panic(err)
	}
	return key
}

// 传入节点编号， 获取对应的私钥
func (node *Node) getPivKey(ClusterName string, nodeID string) []byte {
	key, err := ioutil.ReadFile("Keys/" + ClusterName + "/" + nodeID + "/" + nodeID + "_RSA_PIV")
	if err != nil {
		log.Panic(err)
	}
	return key
}

// 数字签名
func (node *Node) RsaSignWithSha256(data []byte, keyBytes []byte) []byte {
	h := sha256.New()
	h.Write(data)
	hashed := h.Sum(nil)
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		panic(errors.New("private key error"))
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		fmt.Println("ParsePKCS8PrivateKey err", err)
		panic(err)
	}

	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed)
	if err != nil {
		fmt.Printf("Error from signing: %s\n", err)
		panic(err)
	}

	return signature
}

// 签名验证
func (node *Node) RsaVerySignWithSha256(data, signData, keyBytes []byte) bool {
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		panic(errors.New("public key error"))
	}
	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		panic(err)
	}

	hashed := sha256.Sum256(data)
	err = rsa.VerifyPKCS1v15(pubKey.(*rsa.PublicKey), crypto.SHA256, hashed[:], signData)
	if err != nil {
		panic(err)
	}
	return true
}
