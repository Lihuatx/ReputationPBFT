package network

import (
	"bufio"
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
	"os"
	"simple_pbft/pbft/consensus"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Node struct {
	NodeID    string
	NodeTable map[string]map[string]string // key=nodeID, value=url
	ReScore   map[string]map[string]uint8

	View           *View
	CurrentState   *consensus.State
	CommittedMsgs  []*consensus.RequestMsg // kinda block.
	MsgBuffer      *MsgBuffer
	MsgEntrance    chan interface{}
	MsgDelivery    chan interface{}
	MsgRequsetchan chan interface{}
	Alarm          chan bool
	// 全局消息日志和临时消息缓冲区
	GlobalLog    *consensus.GlobalLog
	GlobalBuffer *GlobalBuffer
	GlobalViewID int64
	// 请求消息的锁
	// 请求消息的锁
	MsgBufferLock *MsgBufferLock

	GlobalBufferReqMsgs sync.Mutex
	PendingMsgsLock     sync.Mutex
	PrimaryNodeExeLock  sync.Mutex

	GlobalViewIDLock sync.Mutex

	NodeType NodeType

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

type NodeType int

const (
	CommitteeNode    NodeType = iota // Node is created successfully, but the consensus process is not started yet.
	NonCommittedNode                 // The ReqMsgs is processed successfully. The node is ready to head to the Prepare stage.
)

type MsgBufferLock struct {
	ReqMsgsLock        sync.Mutex
	PrePrepareMsgsLock sync.Mutex
	PrepareMsgsLock    sync.Mutex
	CommitMsgsLock     sync.Mutex
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

var Allcluster = []string{"N", "M", "P"}

const ResolvingTimeDuration = time.Millisecond * 1000 // 1 second.
const CommitteeNodeNumber = 4

func NewNode(nodeID string, clusterName string) *Node {
	const viewID = 10000000000 // temporary.
	node := &Node{
		// Hard-coded for test.
		NodeID: nodeID,
		/*
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

		*/
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
		MsgBufferLock: &MsgBufferLock{},
		ReScore:       make(map[string]map[string]uint8),

		GlobalLog: &consensus.GlobalLog{
			MsgLogs: make(map[string]map[int64]*consensus.RequestMsg),
		},
		GlobalBuffer: &GlobalBuffer{
			ReqMsg:       make([]*consensus.GlobalShareMsg, 0),
			consensusMsg: make([]*consensus.LocalMsg, 0),
		},

		// Channels
		MsgEntrance:       make(chan interface{}, 50),
		MsgDelivery:       make(chan interface{}, 10),
		MsgGlobal:         make(chan interface{}, 50),
		MsgGlobalDelivery: make(chan interface{}, 50),
		MsgRequsetchan:    make(chan interface{}, 50),
		Alarm:             make(chan bool),

		// 所属集群
		ClusterName:  clusterName,
		GlobalViewID: viewID,
	}

	node.NodeTable = LoadNodeTable("nodetable.txt")

	// 初始化全局消息日志
	for _, key := range Allcluster {
		if node.GlobalLog.MsgLogs[key] == nil {
			node.GlobalLog.MsgLogs[key] = make(map[int64]*consensus.RequestMsg)
		}
	}

	//初始化每个节点的分数为70分
	for cluster, nodes := range node.NodeTable {
		node.ReScore[cluster] = make(map[string]uint8) // 为每个集群初始化内部 map
		for key, _ := range nodes {
			node.ReScore[cluster][key] = 70
		}
	}

	// 为每个集群初始化GlobalLog
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

	numberStr := strings.TrimPrefix(nodeID, clusterName)
	// 将剩余的字符串转换为数字
	number, err := strconv.Atoi(numberStr)

	if err != nil {
		fmt.Println("Conversion error:", err)
	}

	// 暂时默认前四个为委员会节点
	if number < CommitteeNodeNumber {
		node.NodeType = CommitteeNode
		fmt.Printf("节点 %s 是委员会节点!\n", node.NodeID)
	} else {
		node.NodeType = NonCommittedNode
		fmt.Printf("节点 %s 是非委员会节点！\n", node.NodeID)
	}

	// 专门用于收取客户端请求,防止堵塞其他线程
	go node.resolveClientRequest()

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

// LoadNodeTable 从指定的文件路径加载 NodeTable
func LoadNodeTable(filePath string) map[string]map[string]string {
	file, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer file.Close()

	// 初始化 NodeTable
	nodeTable := make(map[string]map[string]string)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) == 3 {
			cluster, nodeID, address := parts[0], parts[1], parts[2]

			if _, ok := nodeTable[cluster]; !ok {
				nodeTable[cluster] = make(map[string]string)
			}

			nodeTable[cluster][nodeID] = address
		}
	}

	if err := scanner.Err(); err != nil {
		return nil
	}

	return nodeTable
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
		// fmt.Printf("Send to %s Size of JSON message: %d bytes\n", url+path, len(jsonMsg))
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
		// fmt.Printf("GloablMsg Send to %s Size of JSON message: %d bytes\n", url+path, len(jsonMsg))
		send(url+path, jsonMsg)

	}
	return nil
}

var start time.Time
var duration time.Duration

func (node *Node) Reply(ViewID int64, ReplyMsg *consensus.RequestMsg, GloID int64) (bool, int64) {
	fmt.Printf("Global View ID : %d == %d 达成全局共识\n", node.GlobalViewID, GloID)
	//re := regexp.MustCompile(`\d+`)

	node.CommittedMsgs = append(node.CommittedMsgs, ReplyMsg)
	//for _, value := range node.CommittedMsgs {
	//	//matches := re.FindString(value.Operation)
	//	//if matches == "1" && node.ClusterName == "N" {
	//	//fmt.Printf("Committed value: %s, %d, %s, %d--end ", value.ClientID, value.Timestamp, value.Operation, value.SequenceID)
	//	fmt.Printf("Committed value: %s, --end ", value.Operation)
	//
	//	/*else if matches == "2" && node.ClusterName == "M" {
	//		//fmt.Printf("Committed value: %s, %d, %s, %d--end ", value.ClientID, value.Timestamp, value.Operation, value.SequenceID)
	//		fmt.Printf("Committed value: %s, --end ", value.Operation)
	//	} else if matches == "3" && node.ClusterName == "P" {
	//		fmt.Printf("Committed value: %s, %d, %s, %d--end ", value.ClientID, value.Timestamp, value.Operation, value.SequenceID)
	//	}*/
	//}
	fmt.Print("\n\n")
	/*for _, cluster := range Allcluster {
		for key, value := range node.ReScore[cluster] {
			fmt.Printf("node %s score %d \n", key, value)
		}
	}

	*/

	node.GlobalViewID++

	const viewID = 10000000000 // temporary.
	if node.GlobalViewID == viewID+33 {
		start = time.Now()
	} else if node.GlobalViewID == 10000000066 && node.NodeID == "N0" {
		duration = time.Since(start)
		// 打开文件，如果文件不存在则创建，如果文件存在则追加内容
		fmt.Printf("  Function took %s\n", duration)
		//fmt.Printf("  Function took %s\n", duration)
		//fmt.Printf("  Function took %s\n", duration)

		file, err := os.OpenFile("example.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		// 使用fmt.Fprintf格式化写入内容到文件
		_, err = fmt.Fprintf(file, "durtion: %s\n", duration)
		if err != nil {
			log.Fatal(err)
		}

	} else if node.GlobalViewID > 10000000066 && node.NodeID == "N0" {
		fmt.Printf("  Function took %s\n", duration)
		//fmt.Printf("  Function took %s\n", duration)
		//fmt.Printf("  Function took %s\n", duration)
	}

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
	//LogMsg(reqMsg)

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

	//LogStage(fmt.Sprintf("Consensus Process (ViewID:%d)", node.CurrentState.ViewID), false)

	// Send getPrePrepare message
	if prePrepareMsg != nil {
		// 附加主节点ID,用于数字签名验证
		prePrepareMsg.NodeID = node.NodeID

		node.Broadcast(node.ClusterName, prePrepareMsg, "/preprepare")
		//LogStage("Pre-prepare", true)
	}

	return nil
}

// GetPrePrepare can be called when the node's CurrentState is nil.
// Consensus start procedure for normal participants.
func (node *Node) GetPrePrepare(prePrepareMsg *consensus.PrePrepareMsg, goOn bool) error {
	if node.CurrentState.CurrentStage == consensus.PrePrepared {
		return nil
	}

	//LogMsg(prePrepareMsg)

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

		//LogStage("Pre-prepare", true)
		if node.NodeType == CommitteeNode {
			node.Broadcast(node.ClusterName, prePareMsg, "/prepare")
		}
		//LogStage("Prepare", false)
	}

	return nil
}

func (node *Node) GetPrepare(prepareMsg *consensus.VoteMsg) error {
	if node.CurrentState.CurrentStage == consensus.Prepared {
		return nil
	}

	//LogMsg(prepareMsg)

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

		// 将投票状态共享给其他节点
		for _, value := range node.CurrentState.MsgLogs.PrepareMsgs {
			commitMsg.Score[value.NodeID] = true
		}

		//LogStage("Prepare", true)
		if node.NodeType == CommitteeNode {
			node.Broadcast(node.ClusterName, commitMsg, "/commit")

		}
		//LogStage("Commit", false)
	}

	return nil
}

func (node *Node) GetCommit(commitMsg *consensus.VoteMsg) error {
	// 当节点已经完成Committed阶段后就停止接收其他节点的Committed消息
	if node.CurrentState.CurrentStage == consensus.Committed {
		return nil
	}

	//LogMsg(commitMsg)

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
	node.PrimaryNodeExeLock.Lock()
	if replyMsg != nil {

		if committedMsg == nil {
			return errors.New("committed message is nil, even though the reply message is not nil")
		}

		// Attach node ID to the message
		replyMsg.NodeID = node.NodeID

		// Save the last version of committed messages to node.
		// node.CommittedMsgs = append(node.CommittedMsgs, committedMsg)
		// 更新每个节点的信誉值，首先初始化每个节点的都还没更新分数
		updateFlag := make(map[string]bool)
		for key, _ := range node.NodeTable[node.ClusterName] {
			updateFlag[key] = false
		}
		// 首先增加投票节点的分数，然后增加节点中收到的其他节点的投票的信息
		for _, value := range node.CurrentState.MsgLogs.CommitMsgs {
			id := value.NodeID
			if !updateFlag[id] {
				updateFlag[id] = true // 更新节点需要将标志置位true
				if node.ReScore[node.ClusterName][id] < 90 {
					node.ReScore[node.ClusterName][id] += 1
				}
			}
			for key, vote := range value.Score {
				if vote == true {
					if !updateFlag[key] {
						updateFlag[key] = true
						if node.ReScore[node.ClusterName][key] < 90 {
							node.ReScore[node.ClusterName][key] += 1
						}
					}
				}
			}
		}

		//LogStage("Commit", true)
		fmt.Printf("ViewID :%d %s 达成本地共识，存入待执行缓存池\n", node.View.ID, committedMsg.Operation)

		// Append msg to its logs
		//node.PendingMsgsLock.Lock()
		node.MsgBuffer.PendingMsgs = append(node.MsgBuffer.PendingMsgs, committedMsg)

		if node.NodeID == node.View.Primary { // 本地共识结束后，主节点将本地达成共识的请求发送至其他集群的主节点
			node.PrimaryNodeShareMsg()
		}

		node.View.ID++
		node.CurrentState.CurrentStage = consensus.Committed

	}
	node.PrimaryNodeExeLock.Unlock()
	return nil
}

func (node *Node) PrimaryNodeShareMsg() error {
	// 判断当前节点是代理执行节点，且有要共享的本地共识
	if Allcluster[node.GlobalViewID%int64(len(Allcluster))] == node.ClusterName && len(node.MsgBuffer.PendingMsgs) > 0 && node.MsgBuffer.PendingMsgs[len(node.MsgBuffer.PendingMsgs)-1].Send == false { // 如果轮询到本地主节点作为代理人，发送消息给全局和本地
		//node.PendingMsgsLock.Lock()
		// 找到本地缓存中第一个需要发送的本地共识
		index := len(node.MsgBuffer.PendingMsgs) - 1
		for {
			if index == -1 {
				break
			} else if node.MsgBuffer.PendingMsgs[index].Send == true {
				break
			}
			index--
		}
		node.MsgBuffer.PendingMsgs[index+1].Send = true
		committedMsg := node.MsgBuffer.PendingMsgs[index+1]
		//node.MsgBuffer.PendingMsgs = node.MsgBuffer.PendingMsgs[1:]
		//node.PendingMsgsLock.Unlock()

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
		// 先发送给本地节点执行共识，再发送给委员会中的主节点
		GlobalShareMsg := new(consensus.GlobalShareMsg)
		GlobalShareMsg.RequestMsg = committedMsg
		GlobalShareMsg.NodeID = node.NodeID
		GlobalShareMsg.Sign = signInfo
		GlobalShareMsg.Digest = digest
		GlobalShareMsg.Cluster = node.ClusterName
		GlobalShareMsg.ViewID = node.GlobalViewID

		if GlobalShareMsg.Score == nil {
			GlobalShareMsg.Score = make(map[string]uint8)
		}
		for key, value := range node.ReScore[node.ClusterName] {
			GlobalShareMsg.Score[key] = value
		}

		// 附加节点ID,用于数字签名验证
		sendMsg := &consensus.LocalMsg{
			Sign:           signInfo,
			NodeID:         node.NodeID,
			GlobalShareMsg: GlobalShareMsg,
		}
		node.Broadcast(node.ClusterName, sendMsg, "/GlobalToLocal")
		// 主节点达成本地共识，进行全局共识的排序和执行
		node.GlobalViewIDLock.Lock()
		node.Reply(node.GlobalViewID, committedMsg, node.GlobalViewID)
		node.GlobalViewIDLock.Unlock()

		node.ShareLocalConsensus(GlobalShareMsg, "/global")
		node.GlobalBufferReqMsgs.Lock()
		//最后检查缓存中有没有收到其他代理主节点的消息，执行
		if len(node.GlobalBuffer.ReqMsg) != 0 {
			tempmsg := node.GlobalBuffer.ReqMsg[0]
			if Allcluster[node.GlobalViewID%int64(len(Allcluster))] == tempmsg.Cluster && tempmsg.ViewID == node.GlobalViewID {
				node.GlobalBuffer.ReqMsg = node.GlobalBuffer.ReqMsg[1:]
				node.ShareGlobalMsgToLocal(tempmsg)

			}
		}
		node.GlobalBufferReqMsgs.Unlock()
	}
	return nil
}

func (node *Node) GetReply(msg *consensus.ReplyMsg) {
	fmt.Printf("Result: %s by %s\n", msg.Result, msg.NodeID)
}

func (node *Node) createStateForNewConsensus(goOn bool) error {
	// Check if there is an ongoing consensus process.
	if node.CurrentState.LastSequenceID != -2 {
		if node.CurrentState.CurrentStage != consensus.Committed && !goOn && node.CurrentState.CurrentStage != consensus.GetRequest {
			return errors.New("another consensus is ongoing")
		}
	}
	// Get the last sequence ID
	var lastSequenceID int64
	if len(node.MsgBuffer.PendingMsgs) == 0 {
		lastSequenceID = -1
	} else {
		lastSequenceID = node.MsgBuffer.PendingMsgs[len(node.MsgBuffer.PendingMsgs)-1].SequenceID
	}

	// Create a new state for this new consensus process in the Primary
	node.CurrentState = consensus.CreateState(node.View.ID, lastSequenceID)

	//LogStage("Create the replica status", true)
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

func (node *Node) SaveClientRequest(msg interface{}) {
	switch msg.(type) {
	case *consensus.RequestMsg:

		//一开始没有进行共识的时候，此时 currentstate 为nil
		node.MsgBufferLock.ReqMsgsLock.Lock()
		node.MsgBuffer.ReqMsgs = append(node.MsgBuffer.ReqMsgs, msg.(*consensus.RequestMsg))
		node.MsgBufferLock.ReqMsgsLock.Unlock()
		fmt.Printf("缓存中收到 %d 条客户端请求\n", len(node.MsgBuffer.ReqMsgs))
	}
}

func (node *Node) resolveClientRequest() {
	for {
		select {
		case msg := <-node.MsgRequsetchan:

			node.SaveClientRequest(msg)
			//time.Sleep(50 * time.Millisecond) // 程序暂停100毫秒
		}
	}
}

func (node *Node) routeGlobalMsg(msg interface{}) []error {
	switch m := msg.(type) {
	case *consensus.GlobalShareMsg:
		//fmt.Printf("---- Receive the Global Consensus from %s for Global ID:%d\n", m.NodeID, m.ViewID)
		if m.Cluster != node.ClusterName {
			// Copy buffered messages first.
			node.GlobalBufferReqMsgs.Lock()
			msgs := make([]*consensus.GlobalShareMsg, len(node.GlobalBuffer.ReqMsg))
			copy(msgs, node.GlobalBuffer.ReqMsg)

			// Append a newly arrived message.
			msgs = append(msgs, msg.(*consensus.GlobalShareMsg))

			// Empty the buffer.
			node.GlobalBuffer.ReqMsg = make([]*consensus.GlobalShareMsg, 0)
			node.GlobalBufferReqMsgs.Unlock()
			// Send messages.
			node.MsgGlobalDelivery <- msgs
		}
	case *consensus.LocalMsg:
		//fmt.Printf("---- Receive the Local Consensus from %s for cluster %s Global ID:%d\n", m.NodeID, m.GlobalShareMsg.Cluster, m.GlobalShareMsg.ViewID)
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

	//case *consensus.RequestMsg:
	//	//一开始没有进行共识的时候，此时 currentstate 为nil
	//	node.MsgBufferLock.ReqMsgsLock.Lock()
	//	node.MsgBuffer.ReqMsgs = append(node.MsgBuffer.ReqMsgs, msg.(*consensus.RequestMsg))
	//	node.MsgBufferLock.ReqMsgsLock.Unlock()
	//	fmt.Printf("                    Msgbuffer %d %d %d %d\n", len(node.MsgBuffer.ReqMsgs), len(node.MsgBuffer.PrePrepareMsgs), len(node.MsgBuffer.PrepareMsgs), len(node.MsgBuffer.CommitMsgs))

	case *consensus.PrePrepareMsg:

		node.MsgBufferLock.PrePrepareMsgsLock.Lock()
		node.MsgBuffer.PrePrepareMsgs = append(node.MsgBuffer.PrePrepareMsgs, msg.(*consensus.PrePrepareMsg))
		node.MsgBufferLock.PrePrepareMsgsLock.Unlock()
		//fmt.Printf("                    Msgbuffer %d %d %d %d\n", len(node.MsgBuffer.ReqMsgs), len(node.MsgBuffer.PrePrepareMsgs), len(node.MsgBuffer.PrepareMsgs), len(node.MsgBuffer.CommitMsgs))

	case *consensus.VoteMsg:
		if msg.(*consensus.VoteMsg).MsgType == consensus.PrepareMsg {
			// if node.CurrentState == nil || node.CurrentState.CurrentStage != consensus.PrePrepared
			// 这样的写法会导致当当前节点已经收到2f个节点进入committed阶段时，就会把后来收到的Preprepare消息放到缓冲区中，
			// 这样在下次共识又到prePrepare阶段时就会先去处理上一轮共识的prePrepare协议！
			node.MsgBufferLock.PrepareMsgsLock.Lock()
			node.MsgBuffer.PrepareMsgs = append(node.MsgBuffer.PrepareMsgs, msg.(*consensus.VoteMsg))
			node.MsgBufferLock.PrepareMsgsLock.Unlock()
		} else if msg.(*consensus.VoteMsg).MsgType == consensus.CommitMsg {
			node.MsgBufferLock.CommitMsgsLock.Lock()
			node.MsgBuffer.CommitMsgs = append(node.MsgBuffer.CommitMsgs, msg.(*consensus.VoteMsg))
			node.MsgBufferLock.CommitMsgsLock.Unlock()
		}

		//fmt.Printf("                    Msgbuffer %d %d %d %d\n", len(node.MsgBuffer.ReqMsgs), len(node.MsgBuffer.PrePrepareMsgs), len(node.MsgBuffer.PrepareMsgs), len(node.MsgBuffer.CommitMsgs))
	}

	return nil
}

var lastViewId int64
var lastGlobalId int64

func (node *Node) routeMsgWhenAlarmed() []error {
	if node.View.ID != lastViewId || node.GlobalViewID != lastGlobalId {
		fmt.Printf("                                                                View ID %d,Global ID %d,reqbuf %d\n", node.View.ID, node.GlobalViewID, len(node.MsgBuffer.ReqMsgs))
		lastViewId = node.View.ID
		lastGlobalId = node.GlobalViewID
	}

	//if node.CurrentState == nil || node.CurrentState.CurrentStage == consensus.Committed {
	//	// Check ReqMsgs, send them.
	//	if len(node.MsgBuffer.ReqMsgs) != 0 {
	//		msgs := make([]*consensus.RequestMsg, len(node.MsgBuffer.ReqMsgs))
	//		copy(msgs, node.MsgBuffer.ReqMsgs)
	//
	//		node.MsgDelivery <- msgs
	//	}
	//
	//	// Check PrePrepareMsgs, send them.
	//	if len(node.MsgBuffer.PrePrepareMsgs) != 0 {
	//		msgs := make([]*consensus.PrePrepareMsg, len(node.MsgBuffer.PrePrepareMsgs))
	//		copy(msgs, node.MsgBuffer.PrePrepareMsgs)
	//
	//		node.MsgDelivery <- msgs
	//	}
	//
	//} else {
	//	switch node.CurrentState.CurrentStage {
	//	case consensus.PrePrepared:
	//		// Check PrepareMsgs, send them.
	//		if len(node.MsgBuffer.PrepareMsgs) != 0 {
	//			msgs := make([]*consensus.VoteMsg, len(node.MsgBuffer.PrepareMsgs))
	//			copy(msgs, node.MsgBuffer.PrepareMsgs)
	//
	//			node.MsgDelivery <- msgs
	//		}
	//	case consensus.Prepared:
	//		// Check CommitMsgs, send them.
	//		if len(node.MsgBuffer.CommitMsgs) != 0 {
	//			msgs := make([]*consensus.VoteMsg, len(node.MsgBuffer.CommitMsgs))
	//			copy(msgs, node.MsgBuffer.CommitMsgs)
	//
	//			node.MsgDelivery <- msgs
	//		}
	//	}
	//}

	return nil
}

// 出队
// Dequeue for Request messages
func (mb *MsgBuffer) DequeueReqMsg() *consensus.RequestMsg {
	if len(mb.ReqMsgs) == 0 {
		return nil
	}
	msg := mb.ReqMsgs[0]        // 获取第一个元素
	mb.ReqMsgs = mb.ReqMsgs[1:] // 移除第一个元素
	return msg
}

// Dequeue for PrePrepare messages
func (mb *MsgBuffer) DequeuePrePrepareMsg() *consensus.PrePrepareMsg {
	if len(mb.PrePrepareMsgs) == 0 {
		return nil
	}
	msg := mb.PrePrepareMsgs[0]
	mb.PrePrepareMsgs = mb.PrePrepareMsgs[1:]
	return msg
}

func (node *Node) resolveGlobalMsg() {
	for {
		msg := <-node.MsgGlobalDelivery
		// time.Sleep(50 * time.Millisecond)
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
		switch {
		case len(node.MsgBuffer.ReqMsgs) > 0 && (node.CurrentState.LastSequenceID == -2 || node.CurrentState.CurrentStage == consensus.Committed):
			node.MsgBufferLock.ReqMsgsLock.Lock()
			errs := node.resolveRequestMsg(node.MsgBuffer.ReqMsgs[0])
			if errs != nil {
				fmt.Println(errs)
				// TODO: send err to ErrorChannel
			}
			node.MsgBuffer.DequeueReqMsg()
			node.MsgBufferLock.ReqMsgsLock.Unlock()
		case len(node.MsgBuffer.PrePrepareMsgs) > 0 && (node.CurrentState.LastSequenceID == -2 || node.CurrentState.CurrentStage == consensus.Committed):
			node.MsgBufferLock.PrePrepareMsgsLock.Lock()
			errs := node.resolvePrePrepareMsg(node.MsgBuffer.PrePrepareMsgs[0])
			if errs != nil {
				fmt.Println(errs)
				// TODO: send err to ErrorChannel
			}
			node.MsgBuffer.DequeuePrePrepareMsg()
			node.MsgBufferLock.PrePrepareMsgsLock.Unlock()
		case len(node.MsgBuffer.PrepareMsgs) > 0 && node.CurrentState.CurrentStage == consensus.PrePrepared:
			node.MsgBufferLock.PrepareMsgsLock.Lock()
			var keepIndexes []int     // 用于存储需要保留的元素的索引
			var processIndex int = -1 // 用于存储第一个符合条件的元素的索引，初始化为-1表示未找到
			// 首先遍历PrepareMsgs，确定哪些元素需要保留，哪个元素需要处理
			for index, value := range node.MsgBuffer.PrepareMsgs {
				if value.ViewID < node.View.ID {
					// 不需要做任何事，因为这个元素将被删除
				} else if value.ViewID > node.View.ID {
					keepIndexes = append(keepIndexes, index) // 保留这个元素
				} else if processIndex == -1 { // 只记录第一个符合条件的元素
					processIndex = index
				} else {
					keepIndexes = append(keepIndexes, index)
				}
			}
			// 如果找到了符合条件的元素，则处理它
			if processIndex != -1 {
				errs := node.resolvePrepareMsg(node.MsgBuffer.PrepareMsgs[processIndex])
				// 将这个元素标记为已处理，不再保留
				if errs != nil {
					fmt.Println(errs)
					// TODO: send err to ErrorChannel
				}
			}
			// 创建一个新的切片来存储保留的元素
			var newPrepareMsgs []*consensus.VoteMsg // 假设YourMsgType是PrepareMsgs中元素的类型
			for _, index := range keepIndexes {
				newPrepareMsgs = append(newPrepareMsgs, node.MsgBuffer.PrepareMsgs[index])
			}

			// 更新原来的PrepareMsgs为只包含保留元素的新切片
			node.MsgBuffer.PrepareMsgs = newPrepareMsgs

			node.MsgBufferLock.PrepareMsgsLock.Unlock()

			//errs := node.resolvePrepareMsg(node.MsgBuffer.PrepareMsgs[0])
			//if errs != nil {
			//
			//	fmt.Println(errs)
			//
			//	// TODO: send err to ErrorChannel
			//}
			//node.MsgBufferLock.PrepareMsgsLock.Lock()
			//node.MsgBuffer.DequeuePrepareMsg()
			//node.MsgBufferLock.PrepareMsgsLock.Unlock()
		case len(node.MsgBuffer.CommitMsgs) > 0 && (node.CurrentState.CurrentStage == consensus.Prepared):
			node.MsgBufferLock.CommitMsgsLock.Lock()
			var keepIndexes []int // 用于存储需要保留的元素的索引
			var processIndex = -1 // 用于存储第一个符合条件的元素的索引，初始化为-1表示未找到
			// 首先遍历PrepareMsgs，确定哪些元素需要保留，哪个元素需要处理
			for index, value := range node.MsgBuffer.CommitMsgs {
				if value.ViewID < node.View.ID {
					// 不需要做任何事，因为这个元素将被删除
				} else if value.ViewID > node.View.ID {
					keepIndexes = append(keepIndexes, index) // 保留这个元素
				} else if processIndex == -1 { // 只记录第一个符合条件的元素
					processIndex = index
				} else {
					keepIndexes = append(keepIndexes, index)
				}
			}
			// 如果找到了符合条件的元素，则处理它
			if processIndex != -1 {
				errs := node.resolveCommitMsg(node.MsgBuffer.CommitMsgs[processIndex])
				// 将这个元素标记为已处理，不再保留
				if errs != nil {
					fmt.Println(errs)
					// TODO: send err to ErrorChannel
				}
			}
			// 创建一个新的切片来存储保留的元素
			var newCommitMsgs []*consensus.VoteMsg // 假设YourMsgType是PrepareMsgs中元素的类型
			for _, index := range keepIndexes {
				newCommitMsgs = append(newCommitMsgs, node.MsgBuffer.CommitMsgs[index])
			}

			// 更新原来的PrepareMsgs为只包含保留元素的新切片
			node.MsgBuffer.CommitMsgs = newCommitMsgs

			node.MsgBufferLock.CommitMsgsLock.Unlock()

		default:

		}

	}
}

func (node *Node) alarmToDispatcher() {
	for {
		time.Sleep(ResolvingTimeDuration)
		node.Alarm <- true
	}
}

func (node *Node) resolveRequestMsg(msg *consensus.RequestMsg) error {

	err := node.GetReq(msg, false)
	if err != nil {
		return err
	}

	return nil
}
func (node *Node) resolveGlobalShareMsg(msgs []*consensus.GlobalShareMsg) []error {
	errs := make([]error, 0)

	// Resolve messages
	//fmt.Printf("len GlobalShareMsg msg %d\n", len(msgs))

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
	//fmt.Printf("len LocalGlobalShareMsg msg %d\n", len(msgs))

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
	//fmt.Printf("-----Global-Commit-Execute For %s----\n", msg.GlobalShareMsg.Cluster)

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
	// 首先判断当前消息不是来自本地集群的全局共识消息（因为本地的信誉值已经更新过）
	if reqMsg.GlobalShareMsg.Cluster != node.ClusterName {
		// 更新其他集群节点的信誉值
		for key, value := range reqMsg.GlobalShareMsg.Score {
			node.ReScore[reqMsg.GlobalShareMsg.Cluster][key] = value
		}
	}
	node.Reply(node.GlobalViewID, reqMsg.GlobalShareMsg.RequestMsg, replyMsg.ViewID)

	node.GlobalViewIDLock.Unlock()
	// LogStage("Reply\n", true)

	return nil
}

// 收到其他集群主节点发来的共识消息
func (node *Node) ShareGlobalMsgToLocal(reqMsg *consensus.GlobalShareMsg) error {
	// LogMsg(reqMsg)
	// 如果是本集群发送的消息不需要接受，如果已经收到过这个消息了也不用接收
	if reqMsg.Cluster == node.ClusterName || reqMsg.ViewID < node.GlobalViewID {
		return nil
	}

	if Allcluster[node.GlobalViewID%int64(len(Allcluster))] != reqMsg.Cluster {
		fmt.Printf("收到 %s %d 主节点的共识消息，但此时的代理节点为 %s ，需要等待······\n", reqMsg.Cluster, reqMsg.ViewID, Allcluster[node.GlobalViewID%int64(len(Allcluster))])
		node.GlobalBuffer.ReqMsg = append(node.GlobalBuffer.ReqMsg, reqMsg)
		if node.NodeID == node.View.Primary && node.CurrentState.CurrentStage == consensus.Committed {
			// 检查是否需要执行共识消息的代理节点为本节点
			node.PrimaryNodeExeLock.Lock()
			node.PrimaryNodeShareMsg()
			node.PrimaryNodeExeLock.Unlock()
		}
		// 在后面增加执行代码
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
	// 执行全局共识消息
	node.GlobalViewIDLock.Lock()
	// 更新其他节点的信誉值
	for key, value := range reqMsg.Score {
		node.ReScore[reqMsg.Cluster][key] = value
	}
	node.Reply(node.GlobalViewID, reqMsg.RequestMsg, reqMsg.ViewID)
	node.GlobalViewIDLock.Unlock()
	node.PrimaryNodeExeLock.Lock()
	node.PrimaryNodeShareMsg()
	node.PrimaryNodeExeLock.Unlock()

	return nil
}

func (node *Node) resolvePrePrepareMsg(msg *consensus.PrePrepareMsg) error {

	// Resolve messages
	// 从下标num_of_event_to_resolve开始执行，之前执行过的PrePrepareMsg不需要再执行
	///fmt.Printf("len PrePrepareMsg msg %d\n", len(msgs))
	err := node.GetPrePrepare(msg, false)

	if err != nil {
		return err
	}

	return nil
}

func (node *Node) resolvePrepareMsg(msg *consensus.VoteMsg) error {
	// Resolve messages
	///fmt.Printf("len PrepareMsg msg %d\n", len(msgs))
	if msg.ViewID < node.View.ID {
		return nil
	}
	err := node.GetPrepare(msg)

	if err != nil {
		return err
	}

	return nil
}

func (node *Node) resolveCommitMsg(msg *consensus.VoteMsg) error {
	if msg.ViewID < node.View.ID {
		return nil
	}

	err := node.GetCommit(msg)
	if err != nil {
		return err
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
