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
	"sync"
	"time"
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
	GlobalViewID int64
	// 请求消息的锁
	ReqMsgBufLock    sync.Mutex
	PrepreMsgBufLock sync.Mutex
	PreMsgBufLock    sync.Mutex

	PendingMsgsLock sync.Mutex

	GlobalViewIDLock sync.Mutex

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
	PendingMsgs    []*consensus.RequestMsg
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

var Allcluster = []string{"N", "M"}

const ResolvingTimeDuration = time.Millisecond * 1000 // 1 second.

func NewNode(nodeID string, clusterName string) *Node {
	const viewID = 10000000000 // temporary.
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
			PendingMsgs:    make([]*consensus.RequestMsg, 0),
		},
		GlobalLog: &consensus.GlobalLog{
			MsgLogs: make(map[string]map[int64]*consensus.RequestMsg),
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
		ClusterName:  clusterName,
		GlobalViewID: viewID,
	}

	for _, key := range Allcluster {
		if node.GlobalLog.MsgLogs[key] == nil {
			node.GlobalLog.MsgLogs[key] = make(map[int64]*consensus.RequestMsg)
		}
	}

	node.rsaPubKey = node.getPubKey(clusterName, nodeID)
	node.rsaPrivKey = node.getPivKey(clusterName, nodeID)
	node.CurrentState = consensus.CreateState(node.View.ID, -2)

	lastViewId = 0
	lastGlobalId = 0

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

// ShareLocalConsensus 本地达成共识后，主节点调用当前函数发送信息给其他集群的主节点节点
func (node *Node) ShareLocalConsensus(msg *consensus.GlobalShareMsg, path string) error {
	errorMap := make(map[string]map[string]error)

	for cluster, nodeMsg := range node.NodeTable {
		if cluster == node.ClusterName {
			continue
		}
		url := nodeMsg[PrimaryNode[cluster]]
		jsonMsg, err := json.Marshal(msg)
		if err != nil {
			errorMap[cluster][PrimaryNode[cluster]] = err
			continue
		}
		fmt.Printf("Send to %s Size of JSON message: %d bytes\n", url+path, len(jsonMsg))
		send(url+path, jsonMsg)

		//url = nodeMsg[cluster+"1"]
		//fmt.Printf("Send to %s Size of JSON message: %d bytes\n", url+path, len(jsonMsg))
		//send(url+path, jsonMsg)

	}
	return nil
}

func (node *Node) Reply(ViewID int64, ReplyMsg *consensus.RequestMsg) (bool, int64) {
	fmt.Printf("Global View ID : %d 达成全局共识\n", node.GlobalViewID)

	node.CommittedMsgs = append(node.CommittedMsgs, ReplyMsg)
	for _, value := range node.CommittedMsgs {
		fmt.Printf("Committed value: %s, %d, %s, %d--end ", value.ClientID, value.Timestamp, value.Operation, value.SequenceID)
	}
	fmt.Print("\n\n")

	node.GlobalViewID++

	jsonMsg, err := json.Marshal(ReplyMsg)
	if err != nil {
		return false, ViewID
	}

	// 系统中没有设置用户，reply消息直接发送给主节点
	send("127.0.0.1:5000/reply", jsonMsg)

	return true, ViewID + 1
}

// GetReq can be called when the node's CurrentState is nil.
// Consensus start procedure for the Primary.
func (node *Node) GetReq(reqMsg *consensus.RequestMsg, goOn bool) error {
	LogMsg(reqMsg)

	// Create a new state for the new consensus.
	err := node.createStateForNewConsensus(goOn)
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
func (node *Node) GetPrePrepare(prePrepareMsg *consensus.PrePrepareMsg, goOn bool) error {
	LogMsg(prePrepareMsg)

	// Create a new state for the new consensus.
	err := node.createStateForNewConsensus(goOn)
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
	// 达成本地Committed共识
	if replyMsg != nil {

		if committedMsg == nil {
			return errors.New("committed message is nil, even though the reply message is not nil")
		}

		// Attach node ID to the message
		replyMsg.NodeID = node.NodeID

		// Save the last version of committed messages to node.
		// node.CommittedMsgs = append(node.CommittedMsgs, committedMsg)

		LogStage("Commit", true)
		fmt.Printf("ViewID :%d 达成本地共识，存入待执行缓存池\n", node.View.ID)

		// LogStage("Reply", true)

		// Append msg to its logs
		node.PendingMsgsLock.Lock()
		node.MsgBuffer.PendingMsgs = append(node.MsgBuffer.PendingMsgs, committedMsg)
		node.PendingMsgsLock.Unlock()

		if node.NodeID == node.View.Primary { // 本地共识结束后，主节点将本地达成共识的请求发送至其他集群的主节点
			if Allcluster[node.GlobalViewID%int64(len(Allcluster))] == node.ClusterName { // 如果轮询到本地主节点作为代理人，发送消息给全局和本地
				fmt.Printf("send consensus to Global\n")
				// 获取消息摘要
				msg, err := json.Marshal(committedMsg)
				if err != nil {
					return err
				}
				digest := consensus.Hash(msg)
				//取出存在buffer中的消息，发送出去全局共识
				node.PendingMsgsLock.Lock()
				node.MsgBuffer.PendingMsgs = node.MsgBuffer.PendingMsgs[1:]
				node.PendingMsgsLock.Unlock()
				// 节点对消息摘要进行签名
				digestByte, _ := hex.DecodeString(digest)
				signInfo := node.RsaSignWithSha256(digestByte, node.rsaPrivKey)
				// committedMsg.Result = false
				GlobalShareMsg := new(consensus.GlobalShareMsg)
				GlobalShareMsg.RequestMsg = committedMsg
				GlobalShareMsg.NodeID = node.NodeID
				GlobalShareMsg.Sign = signInfo
				GlobalShareMsg.Digest = digest
				GlobalShareMsg.Cluster = node.ClusterName
				GlobalShareMsg.ViewID = node.GlobalViewID
				node.ShareLocalConsensus(GlobalShareMsg, "/global")
				// 附加节点ID,用于数字签名验证
				sendMsg := &consensus.LocalMsg{
					Sign:           signInfo,
					NodeID:         node.NodeID,
					GlobalShareMsg: GlobalShareMsg,
				}
				node.Broadcast(node.ClusterName, sendMsg, "/GlobalToLocal")
				// 达成本地共识，进行全局共识的排序和执行
				node.Reply(node.GlobalViewID, committedMsg)
			}

			node.View.ID++

			//	对于主节点而言，如果请求缓存池中还有请求，需要继续执行本地共识
			var TempReqMsg *consensus.RequestMsg
			node.ReqMsgBufLock.Lock()
			if len(node.MsgBuffer.ReqMsgs) > 0 {
				// 直接获取第一个请求消息
				TempReqMsg = node.MsgBuffer.ReqMsgs[0]
				// 直接更新请求消息缓冲区，去掉已处理的第一个消息
				node.MsgBuffer.ReqMsgs = node.MsgBuffer.ReqMsgs[1:]
			}
			node.ReqMsgBufLock.Unlock()

			// 如果有请求消息，则继续执行相关处理
			if TempReqMsg != nil {
				fmt.Printf("                                  go on                               go on\n")
				node.GetReq(TempReqMsg, true)
			} else {
				node.CurrentState.CurrentStage = consensus.Committed
			}
		} else { // 对于其他本地子节点，如果已经有 Preprepare 缓存消息
			node.View.ID++
			var TempReqMsg *consensus.PrePrepareMsg
			node.PrepreMsgBufLock.Lock()
			if len(node.MsgBuffer.PrePrepareMsgs) > 0 {
				// 直接获取第一个请求消息
				TempReqMsg = node.MsgBuffer.PrePrepareMsgs[0]
				// 直接更新请求消息缓冲区，去掉已处理的第一个消息
				node.MsgBuffer.PrePrepareMsgs = node.MsgBuffer.PrePrepareMsgs[1:]
			}
			node.PrepreMsgBufLock.Unlock()

			// 如果有请求消息，则继续执行相关处理
			if TempReqMsg != nil && TempReqMsg.ViewID >= node.View.ID {
				fmt.Printf("                                  go on                               go on\n")
				node.GetPrePrepare(TempReqMsg, true)
			} else {
				node.CurrentState.CurrentStage = consensus.Committed
			}
		}

	}
	return nil
}

func (node *Node) GetReply(msg *consensus.ReplyMsg) {
	fmt.Printf("Result: %s by %s\n", msg.Result, msg.NodeID)
}

func (node *Node) createStateForNewConsensus(goOn bool) error {
	const viewID = 10000000000 // temporary.
	// Check if there is an ongoing consensus process.
	if node.CurrentState.LastSequenceID != -2 {
		if node.CurrentState.CurrentStage != consensus.Committed && !goOn && node.CurrentState.CurrentStage != consensus.GetRequest {
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
		fmt.Printf("---- Receive the Global Consensus from %s for Global ID:%d\n", m.NodeID, m.ViewID)
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
		fmt.Printf("---- Receive the Local Consensus from %s for cluster %s Global ID:%d\n", m.NodeID, m.GlobalShareMsg.Cluster, m.GlobalShareMsg.ViewID)
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
		if node.CurrentState.LastSequenceID == -2 || node.CurrentState.CurrentStage == consensus.Committed { //一开始没有进行共识的时候，此时 currentstate 为nil
			// Copy buffered messages first.
			msgs := make([]*consensus.RequestMsg, len(node.MsgBuffer.ReqMsgs))
			copy(msgs, node.MsgBuffer.ReqMsgs)
			// Append a newly arrived message.
			msgs = append(msgs, msg.(*consensus.RequestMsg))

			// Empty the buffer.
			node.ReqMsgBufLock.Lock()
			node.MsgBuffer.ReqMsgs = make([]*consensus.RequestMsg, 0)
			node.ReqMsgBufLock.Unlock()
			// Send messages.
			// 注意这行代码，在系统初始化时必须先主动发一次请求达成共识，否则一开始node.CurrentState=nil
			node.CurrentState.CurrentStage = consensus.GetRequest
			node.MsgDelivery <- msgs
		} else {
			node.ReqMsgBufLock.Lock()
			node.MsgBuffer.ReqMsgs = append(node.MsgBuffer.ReqMsgs, msg.(*consensus.RequestMsg))
			node.ReqMsgBufLock.Unlock()
		}
		fmt.Printf("                    request buffer %d\n", len(node.MsgBuffer.ReqMsgs))
	case *consensus.PrePrepareMsg:
		if node.CurrentState.LastSequenceID == -2 || node.CurrentState.CurrentStage == consensus.Committed {
			// Copy buffered messages first.
			node.PrepreMsgBufLock.Lock()
			msgs := make([]*consensus.PrePrepareMsg, len(node.MsgBuffer.PrePrepareMsgs))
			copy(msgs, node.MsgBuffer.PrePrepareMsgs)

			// Append a newly arrived message.
			msgs = append(msgs, msg.(*consensus.PrePrepareMsg))

			// Empty the buffer.
			node.MsgBuffer.PrePrepareMsgs = make([]*consensus.PrePrepareMsg, 0)
			node.PrepreMsgBufLock.Unlock()

			// Send messages.
			node.MsgDelivery <- msgs
		} else {
			node.PrepreMsgBufLock.Lock()
			node.MsgBuffer.PrePrepareMsgs = append(node.MsgBuffer.PrePrepareMsgs, msg.(*consensus.PrePrepareMsg))
			node.PrepreMsgBufLock.Unlock()
		}
	case *consensus.VoteMsg:
		if msg.(*consensus.VoteMsg).MsgType == consensus.PrepareMsg {
			// if node.CurrentState == nil || node.CurrentState.CurrentStage != consensus.PrePrepared
			// 这样的写法会导致当当前节点已经收到2f个节点进入committed阶段时，就会把后来收到的Preprepare消息放到缓冲区中，
			// 这样在下次共识又到prePrepare阶段时就会先去处理上一轮共识的prePrepare协议！
			if node.CurrentState.LastSequenceID == -2 || node.CurrentState.CurrentStage != consensus.PrePrepared {
				node.MsgBuffer.PrepareMsgs = append(node.MsgBuffer.PrepareMsgs, msg.(*consensus.VoteMsg))
			} else {
				// Copy buffered messages first.
				var msgs []*consensus.VoteMsg
				var msgSave []*consensus.VoteMsg
				node.MsgBuffer.PrepareMsgs = append(node.MsgBuffer.PrepareMsgs, msg.(*consensus.VoteMsg))
				copy(msgs, node.MsgBuffer.PrepareMsgs)

				for _, value := range node.MsgBuffer.PrepareMsgs {
					if value.ViewID == node.View.ID {
						msgs = append(msgs, value)
					} else if value.ViewID > node.View.ID {
						msgSave = append(msgSave, value)
					}

				}
				// Append a newly arrived message.
				// msgs = append(msgs, msg.(*consensus.VoteMsg))
				// Empty the buffer.
				node.MsgBuffer.PrepareMsgs = msgSave

				// Send messages.
				node.MsgDelivery <- msgs

			}
		} else if msg.(*consensus.VoteMsg).MsgType == consensus.CommitMsg {
			if node.CurrentState.LastSequenceID == -2 || node.CurrentState.CurrentStage != consensus.Prepared {
				node.MsgBuffer.CommitMsgs = append(node.MsgBuffer.CommitMsgs, msg.(*consensus.VoteMsg))
			} else {
				// Copy buffered messages first.
				var msgs []*consensus.VoteMsg
				var msgSave []*consensus.VoteMsg
				node.MsgBuffer.CommitMsgs = append(node.MsgBuffer.CommitMsgs, msg.(*consensus.VoteMsg))
				copy(msgs, node.MsgBuffer.CommitMsgs)

				for _, value := range node.MsgBuffer.CommitMsgs {
					if value.ViewID == node.View.ID {
						msgs = append(msgs, value)
					} else if value.ViewID > node.View.ID {
						msgSave = append(msgSave, value)
					}

				}
				// Append a newly arrived message.
				// msgs = append(msgs, msg.(*consensus.VoteMsg))
				// Empty the buffer.
				node.MsgBuffer.CommitMsgs = msgSave

				// Send messages.
				node.MsgDelivery <- msgs
			}
		}
	}

	return nil
}

var lastViewId int64
var lastGlobalId int64

func (node *Node) routeMsgWhenAlarmed() []error {
	if node.View.ID != lastViewId || node.GlobalViewID != lastGlobalId {
		fmt.Printf("                                                                View ID %d,Global ID %d\n", node.View.ID, node.GlobalViewID)
		lastViewId = node.View.ID
		lastGlobalId = node.GlobalViewID
	}
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

		// 查看是否要发送Empty消息
		g := []string{"N", "M"}
		flag := false
		// 如果当前的已经有收到其他集群达成的共识，且本集群没有收到request，发送empty消息！
		for _, value := range g {
			if value == node.ClusterName {
				continue
			} else {
				if node.GlobalLog.MsgLogs[value][node.GlobalViewID] != nil {
					flag = true
				}
			}
		}
		if flag && len(node.MsgBuffer.ReqMsgs) == 0 {
			/*
				fmt.Printf("Start Empty consensus for ViewID %d in ShareGlobalMsgToLocal\n", node.GlobalViewID)
				var msg consensus.RequestMsg
				msg.ClientID = node.NodeID
				msg.Operation = "Empty"
				msg.Timestamp = 0
				node.MsgEntrance <- &msg
			*/
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

	err := node.GetReq(msgs[0], false)
	if err != nil {
		return errs
	}

	if len(msgs) > 1 {
		node.ReqMsgBufLock.Lock()
		tempMsg := msgs[1:]
		TempReqBuf := make([]*consensus.RequestMsg, len(node.MsgBuffer.ReqMsgs))
		copy(TempReqBuf, node.MsgBuffer.ReqMsgs)
		// Append a newly arrived message.
		tempMsg = append(tempMsg, TempReqBuf...)
		node.MsgBuffer.ReqMsgs = tempMsg
		node.ReqMsgBufLock.Unlock()
	}

	return nil
}

func (node *Node) resolveGlobalShareMsg(msgs []*consensus.GlobalShareMsg) []error {
	errs := make([]error, 0)

	// Resolve messages
	fmt.Printf("len GlobalShareMsg msg %d\n", len(msgs))

	for _, reqMsg := range msgs {
		// 收到其他组的消息，转发给其他主节点节点
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
	fmt.Printf("len LocalGlobalShareMsg msg %d\n", len(msgs))

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

func (node *Node) GlobalConsensus(msg *consensus.LocalMsg) (*consensus.ReplyMsg, *consensus.RequestMsg, error) {
	// Print current voting status
	fmt.Printf("-----Global-Commit-Execute For %s----\n", msg.GlobalShareMsg.Cluster)

	// This node executes the requested operation locally and gets the result.
	result := "Executed"

	// Change the stage to prepared.
	return &consensus.ReplyMsg{
		ViewID:    msg.GlobalShareMsg.ViewID,
		Timestamp: 0,
		ClientID:  msg.GlobalShareMsg.RequestMsg.ClientID,
		Result:    result,
	}, msg.GlobalShareMsg.RequestMsg, nil

	// return nil, nil, nil
}

// CommitGlobalMsgToLocal 收到本地主节点发来的全局共识消息
func (node *Node) CommitGlobalMsgToLocal(reqMsg *consensus.LocalMsg) error {
	// LogMsg(reqMsg)

	digest, _ := hex.DecodeString(reqMsg.GlobalShareMsg.Digest)
	if !node.RsaVerySignWithSha256(digest, reqMsg.Sign, node.getPubKey(node.ClusterName, reqMsg.NodeID)) {
		fmt.Println("节点签名验证失败！,拒绝执行Global commit")
	}
	if reqMsg.GlobalShareMsg.Cluster == node.ClusterName && node.NodeID != node.View.Primary {
		//如果是本地的共识，取出存在buffer中的第一个消息，说明已经达成共识
		for {
			if len(node.MsgBuffer.PendingMsgs) > 0 {
				break
				fmt.Printf("node.MsgBuffer.PendingMsgs 为0\n")
			}
		}
		node.PendingMsgsLock.Lock()
		node.MsgBuffer.PendingMsgs = node.MsgBuffer.PendingMsgs[1:]
		node.PendingMsgsLock.Unlock()
	}
	// GlobalConsensus 会将msg存入MsgLogs中
	replyMsg, committedMsg, err := node.GlobalConsensus(reqMsg)
	if err != nil {
		ErrMessage(committedMsg)
		return err
	}

	// Attach node ID to the message
	replyMsg.NodeID = node.NodeID
	// Save the last version of committed messages to node.
	// node.CommittedMsgs = append(node.CommittedMsgs, committedMsg)
	fmt.Printf("Global stage ID %s %d\n", reqMsg.GlobalShareMsg.Cluster, reqMsg.GlobalShareMsg.ViewID)
	node.GlobalViewIDLock.Lock()

	node.Reply(node.GlobalViewID, reqMsg.GlobalShareMsg.RequestMsg)

	node.GlobalViewIDLock.Unlock()
	// LogStage("Reply\n", true)

	return nil
}

// 收到其他集群主节点发来的共识消息
func (node *Node) ShareGlobalMsgToLocal(reqMsg *consensus.GlobalShareMsg) error {
	// LogMsg(reqMsg)
	// LogStage(fmt.Sprintf("Consensus Process (ViewID:%d)", node.CurrentState.ViewID), false)
	// 如果是本集群发送的消息不需要接受，如果已经收到过这个消息了也不用接收
	if reqMsg.Cluster == node.ClusterName || node.GlobalLog.MsgLogs[reqMsg.Cluster][reqMsg.ViewID] != nil {
		return nil
	}

	digest, _ := hex.DecodeString(reqMsg.Digest)
	if !node.RsaVerySignWithSha256(digest, reqMsg.Sign, node.getPubKey(reqMsg.Cluster, reqMsg.NodeID)) {
		fmt.Println("节点签名验证失败！,拒绝执行Global commit")
	}

	if reqMsg.NodeID != PrimaryNode[reqMsg.Cluster] {
		fmt.Printf("非 %s 主节点发送的全局共识，拒绝接受", reqMsg.Cluster)
		return nil
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

	// 发送给其他主节点和本地节点
	node.ShareLocalConsensus(reqMsg, "/global")
	node.Broadcast(node.ClusterName, sendMsg, "/GlobalToLocal")

	fmt.Printf("----- 收到其他委员会节点发来的全局共识，已发送给本地节点和其他委员会节点 -----\n")
	node.Reply(node.GlobalViewID, reqMsg.RequestMsg)

	if Allcluster[node.GlobalViewID%int64(len(Allcluster))] == node.ClusterName && len(node.MsgBuffer.PendingMsgs) > 0 { // 如果轮询到本地主节点作为代理人，发送消息给全局和本地
		node.PendingMsgsLock.Lock()
		committedMsg := node.MsgBuffer.PendingMsgs[0]
		node.MsgBuffer.PendingMsgs = node.MsgBuffer.PendingMsgs[1:]
		node.PendingMsgsLock.Unlock()

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
		// committedMsg.Result = false
		GlobalShareMsg := new(consensus.GlobalShareMsg)
		GlobalShareMsg.RequestMsg = committedMsg
		GlobalShareMsg.NodeID = node.NodeID
		GlobalShareMsg.Sign = signInfo
		GlobalShareMsg.Digest = digest
		GlobalShareMsg.Cluster = node.ClusterName
		GlobalShareMsg.ViewID = node.GlobalViewID
		node.ShareLocalConsensus(GlobalShareMsg, "/global")
		// 附加节点ID,用于数字签名验证
		sendMsg := &consensus.LocalMsg{
			Sign:           signInfo,
			NodeID:         node.NodeID,
			GlobalShareMsg: GlobalShareMsg,
		}
		node.Broadcast(node.ClusterName, sendMsg, "/GlobalToLocal")
		// 达成本地共识，进行全局共识的排序和执行
		node.Reply(node.GlobalViewID, committedMsg)
	}
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
		err := node.GetPrePrepare(prePrepareMsg, false)
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
	fmt.Printf("len CommitMsg msg %d node\n", len(msgs))

	for _, commitMsg := range msgs {
		if commitMsg.ViewID < node.View.ID {
			continue
		} else if commitMsg.ViewID > node.View.ID {

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
