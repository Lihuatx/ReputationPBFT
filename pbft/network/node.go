package network

import (
	"My_PBFT/pbft/consensus"
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
	"strconv"
	"strings"
	"sync"
	"time"
)

type Node struct {
	NodeID    string
	NodeTable map[string]map[string]string // key=nodeID, value=url
	ReScore   map[string]map[string]uint16 // cluster , nodeID , score

	AllClusterNewViewBuffer map[string]map[int64]string

	ActiveCommitteeNode map[string]NodeType
	ReElement           *ReElement
	AcceptRequestTime   map[int64]time.Time // req SequenceID -> Start time

	View              *View
	CurrentState      *consensus.State
	CommittedMsgs     []*consensus.RequestMsg // kinda block.
	MsgBuffer         *MsgBuffer
	ViewChangeMsgVote map[string]*consensus.ViewChangeMsg
	MsgEntrance       chan interface{}
	ScoreEntrance     chan interface{}
	MsgDelivery       chan interface{}
	MsgRequsetchan    chan interface{}
	Alarm             chan bool
	// 全局消息日志和临时消息缓冲区
	//GlobalLog    *consensus.GlobalLog
	GlobalBuffer *GlobalBuffer
	GlobalViewID int64
	// 请求消息的锁
	// 请求消息的锁
	MsgBufferLock *MsgBufferLock

	GlobalBufferReqMsgs sync.Mutex
	PendingMsgsLock     sync.Mutex
	PrimaryNodeExeLock  sync.Mutex
	CheckViewChangeLock sync.Mutex

	GlobalViewIDLock sync.Mutex
	NewViewLock      sync.Mutex

	NodeType      NodeType
	MaliciousNode MaliciousNode

	PrimaryViewID    int64
	NewPrimaryNodeID string

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
	NonCommittedNode NodeType = iota // Node is created successfully, but the consensus process is not started yet.
	CommitteeNode                    // The ReqMsgs is processed successfully. The node is ready to head to the Prepare stage.
)

type MaliciousNode int

const (
	isMaliciousNode  MaliciousNode = iota // Node is created successfully, but the consensus process is not started yet.
	NonMaliciousNode                      // The ReqMsgs is processed successfully. The node is ready to head to the Prepare stage.
)

type SignInfo struct {
	Sign map[string][]byte
}

type ReElement struct {
	Active       map[string]int   //活跃度
	HistoryScore map[string][]int //历史分数
}

type MsgBufferLock struct {
	ReqMsgsLock        sync.Mutex
	PrePrepareMsgsLock sync.Mutex
	PrepareMsgsLock    sync.Mutex
	CommitMsgsLock     sync.Mutex
}

type GlobalBuffer struct {
	ReqMsg       []*consensus.GlobalShareMsg //其他集群的请求消息缓存
	consensusMsg []*consensus.LocalMsg       //本地节点的全局共识消息缓存
	NewViewMsg   []*consensus.NewView
}

type MsgBuffer struct {
	ReqMsgs        []*consensus.RequestMsg
	PrePrepareMsgs []*consensus.PrePrepareMsg
	PrepareMsgs    []*consensus.VoteMsg
	CommitMsgs     []*consensus.VoteMsg
	PendingMsgs    []*consensus.BatchRequestMsg
	BatchReqMsgs   []*consensus.BatchRequestMsg
	ViewChangeMsgs []*consensus.ViewChangeMsg
	NewViewMsgs    []*consensus.NewView
	SignInfo       []SignInfo
}

type View struct {
	ID      int64
	Number  int64
	Primary string
}

var PrimaryNode = map[string]string{
	"N": "N0",
	"M": "M0",
	"P": "P0",
	"J": "J0",
	"K": "K0",
}

var Allcluster = []string{"N", "M", "P", "J", "K"}

const ResolvingTimeDuration = time.Millisecond * 1000 // 1 second.
var CommitteeNodeNumber = 4                           // 定义委员会节点变量
var BaseReScore uint16 = 400
var ReScoretThreshold uint16 = 200
var ClusterNumber = 5
var PrimaryNodeChangeFreq = 9

func NewNode(nodeID string, clusterName string, ismaliciousNode string) *Node {
	const viewID = 10000000000 // temporary.
	consensus.F = 1
	node := &Node{
		// Hard-coded for test.
		NodeID:           nodeID,
		NewPrimaryNodeID: "",
		PrimaryViewID:    viewID,
		View: &View{
			ID:      viewID,
			Number:  0,
			Primary: PrimaryNode[clusterName],
		},

		// Consensus-related struct
		CurrentState:            nil,
		AllClusterNewViewBuffer: make(map[string]map[int64]string),
		CommittedMsgs:           make([]*consensus.RequestMsg, 0),
		MsgBuffer: &MsgBuffer{
			ReqMsgs:        make([]*consensus.RequestMsg, 0),
			PrePrepareMsgs: make([]*consensus.PrePrepareMsg, 0),
			PrepareMsgs:    make([]*consensus.VoteMsg, 0),
			CommitMsgs:     make([]*consensus.VoteMsg, 0),
			PendingMsgs:    make([]*consensus.BatchRequestMsg, 0),
			BatchReqMsgs:   make([]*consensus.BatchRequestMsg, 0),
			ViewChangeMsgs: make([]*consensus.ViewChangeMsg, 0),
			NewViewMsgs:    make([]*consensus.NewView, 0),
			SignInfo:       make([]SignInfo, 0),
		},
		MsgBufferLock:     &MsgBufferLock{},
		ReScore:           make(map[string]map[string]uint16),
		ViewChangeMsgVote: make(map[string]*consensus.ViewChangeMsg),

		GlobalBuffer: &GlobalBuffer{
			ReqMsg:       make([]*consensus.GlobalShareMsg, 0),
			consensusMsg: make([]*consensus.LocalMsg, 0),
			NewViewMsg:   make([]*consensus.NewView, 0),
		},

		// Channels
		MsgEntrance:       make(chan interface{}, 100),
		MsgDelivery:       make(chan interface{}, 100),
		MsgGlobal:         make(chan interface{}, 100),
		MsgGlobalDelivery: make(chan interface{}, 100),
		MsgRequsetchan:    make(chan interface{}, 100),
		ScoreEntrance:     make(chan interface{}, 10),
		Alarm:             make(chan bool),

		ReElement: &ReElement{
			Active:       make(map[string]int),
			HistoryScore: make(map[string][]int),
		},
		ActiveCommitteeNode: make(map[string]NodeType),
		AcceptRequestTime:   make(map[int64]time.Time),
		// 所属集群
		ClusterName:  clusterName,
		GlobalViewID: viewID,
	}

	node.NodeTable = LoadNodeTable("nodetable.txt")
	if ismaliciousNode == "Y" {
		node.MaliciousNode = isMaliciousNode
	} else {
		node.MaliciousNode = NonMaliciousNode
	}

	//初始化每个节点的分数为400分
	for cluster, nodes := range node.NodeTable {
		node.ReScore[cluster] = make(map[string]uint16) // 为每个集群初始化内部 map
		for key, _ := range nodes {
			node.ReScore[cluster][key] = BaseReScore
		}
	}

	for cluster, _ := range PrimaryNode {
		node.AllClusterNewViewBuffer[cluster] = make(map[int64]string) // 为每个集群初始化New View buffer
	}

	node.rsaPubKey = node.getPubKey(clusterName, nodeID)
	node.rsaPrivKey = node.getPivKey(clusterName, nodeID)

	// 初始化
	node.CurrentState = consensus.CreateState(node.View.ID, -2)

	lastViewId = 0
	lastGlobalId = 0

	numberStr := strings.TrimPrefix(nodeID, clusterName)
	// 将剩余的字符串转换为数字
	number, err := strconv.Atoi(numberStr)

	if err != nil {
		fmt.Println("Conversion error:", err)
	}

	// 设置默认委员会节点
	if number < CommitteeNodeNumber {
		node.NodeType = CommitteeNode
		fmt.Printf("节点 %s 是委员会节点!\n", node.NodeID)
	} else {
		node.NodeType = NonCommittedNode
		fmt.Printf("节点 %s 是非委员会节点！\n", node.NodeID)
	}
	// 初始化委员会节点
	for i := 0; i < CommitteeNodeNumber; i++ {
		nodeId := node.ClusterName + strconv.Itoa(i)
		node.ActiveCommitteeNode[nodeId] = CommitteeNode
	}
	consensus.F = (CommitteeNodeNumber - 1) / 3

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

	// go node.check()

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
		//fmt.Printf("Send to %s Size of JSON message: %d bytes\n", url+path, len(jsonMsg))
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

	for i := 0; i < ClusterNumber; i++ {
		cluster := Allcluster[i]
		if cluster == node.ClusterName {
			continue
		}
		primaryNodeID := PrimaryNode[cluster]
		url := node.NodeTable[cluster][primaryNodeID]
		jsonMsg, err := json.Marshal(msg)
		if err != nil {
			errorMap[cluster][PrimaryNode[cluster]] = err
			continue
		}
		fmt.Printf("GloablMsg Send to %s Size of JSON message: %d bytes\n", url+path, len(jsonMsg))
		send(url+path, jsonMsg)
	}

	return nil
}

// 将主节点更换信息共享给其他集群
func (node *Node) ShareViewChangeMsgToGlobal(msg *consensus.NewView, path string) error {
	errorMap := make(map[string]map[string]error)

	for i := 0; i < ClusterNumber; i++ {
		cluster := Allcluster[i]
		if cluster == node.ClusterName {
			continue
		}
		primaryNodeID := PrimaryNode[cluster]
		url := node.NodeTable[cluster][primaryNodeID]
		jsonMsg, err := json.Marshal(msg)
		if err != nil {
			errorMap[cluster][PrimaryNode[cluster]] = err
			continue
		}
		fmt.Printf("GloablMsg Send to %s Size of JSON message: %d bytes\n", url+path, len(jsonMsg))
		send(url+path, jsonMsg)
	}

	return nil
}

var start time.Time
var duration time.Duration

func (node *Node) Reply(ViewID int64, ReplyMsg *consensus.BatchRequestMsg, GloID int64) (bool, int64) {
	fmt.Printf("Global View ID : %d == %d 达成全局共识\n", node.GlobalViewID, GloID)
	//re := regexp.MustCompile(`\d+`)

	for i := 0; i < consensus.BatchSize; i++ {
		node.CommittedMsgs = append(node.CommittedMsgs, ReplyMsg.Requests[i])

	}
	//for _, value := range node.CommittedMsgs {
	//	fmt.Printf("Commited Msg:%v  ", value.Operation)
	//}
	fmt.Print("\n\n")

	for value, _type := range node.ActiveCommitteeNode {
		if _type == CommitteeNode {
			fmt.Printf("node %s score %d \n", value, node.ReScore[node.ClusterName][value])
		}
	}

	if len(node.CommittedMsgs) == 1 {
		//start = time.Now()
	} else if len(node.CommittedMsgs) == 3000 && node.NodeID == "N0" {
		duration = time.Since(start)
		// 打开文件，如果文件不存在则创建，如果文件存在则追加内容
		fmt.Printf("  Function took %s\n", duration)

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

	} else if len(node.CommittedMsgs) > 3000 && node.NodeID == "N0" {
		fmt.Printf("  Function took %s\n", duration)
	}

	if node.NodeID == node.View.Primary && Allcluster[node.GlobalViewID%int64(ClusterNumber)] == node.ClusterName { //主节点返回reply消息给客户端
		go func() {
			for i := 0; i < consensus.BatchSize; i++ {
				jsonMsg, _ := json.Marshal(ReplyMsg.Requests[i])
				// 系统中没有设置用户，reply消息直接发送给主节点
				url := ClientURL[node.ClusterName] + "/reply"
				send(url, jsonMsg)
				//fmt.Printf("\n\nReply to Client!\n\n\n")
			}
		}()
	}

	node.GlobalViewID++
	node.CheckViewChangeLock.Lock()
	node.CheckViewChange()
	node.CheckViewChangeLock.Unlock()
	return true, ViewID + 1
}

func (node *Node) check() {
	for {
		err := node.CheckViewChange()
		if err != nil {
			fmt.Println(err)
		}
		time.Sleep(time.Millisecond)
	}
}

func (node *Node) CheckViewChange() error {
	if (node.GlobalViewID - viewID) == 0 {
		return nil
	}
	if node.PrimaryViewID != node.View.ID {
		return nil
	}
	if (node.GlobalViewID-viewID)%int64(PrimaryNodeChangeFreq) == 0 && node.CurrentState.CurrentStage != consensus.ViewChange && (node.GlobalViewID-viewID) > node.View.Number*int64(PrimaryNodeChangeFreq) {
		fmt.Println("准备开始View Change--->")
		for node.CurrentState.CurrentStage != consensus.Committed && node.CurrentState.CurrentStage != consensus.Idle { // 等待此次本地共识的结束
			if node.PrimaryViewID != node.View.ID {
				return nil
			}
		}
		node.CurrentState.CurrentStage = consensus.ViewChange
		fmt.Println("--->开始View Change")
		var NewPrimaryNode = ""
		var NewPrimaryNodeScore uint16 = 0
		for i := 0; i < len(node.NodeTable[node.ClusterName]); i++ {
			nodeID := node.ClusterName + strconv.Itoa(i)
			score := node.ReScore[node.ClusterName][nodeID]
			if nodeID == node.View.Primary || node.ActiveCommitteeNode[nodeID] == NonCommittedNode {
				continue
			}
			if score > NewPrimaryNodeScore {
				NewPrimaryNode = nodeID
				NewPrimaryNodeScore = score
			}
		}
		digest, err := consensus.GetDigest(node.MsgBuffer.PendingMsgs[len(node.MsgBuffer.PendingMsgs)-1])
		fmt.Printf("Digest Msg op:%v, sq:%v\n", node.MsgBuffer.PendingMsgs[len(node.MsgBuffer.PendingMsgs)-1].Requests[0].Operation, node.MsgBuffer.PendingMsgs[len(node.MsgBuffer.PendingMsgs)-1].Requests[0].SequenceID)
		if err != nil {
			fmt.Println(err)
			return err
		}
		digestInfo, _ := hex.DecodeString(digest)
		signInfo := node.RsaSignWithSha256(digestInfo, node.rsaPrivKey)

		node.NewPrimaryNodeID = NewPrimaryNode

		msg := consensus.ViewChangeMsg{
			NodeID:           node.NodeID,
			NewPrimaryNodeID: NewPrimaryNode,
			NewViewNumber:    node.View.Number + 1,
			LastViewID:       node.View.ID,
			LastPendingMsg:   digest,
			Sign:             signInfo,
		}
		node.Broadcast(node.ClusterName, msg, "/ViewChange")
		fmt.Printf("\n\nStart View Change!!! Primary Node Should be %s\n\n", NewPrimaryNode)
	}
	return nil
}

func (node *Node) GetNewView(Msg *consensus.NewView) error {
	//node.NewViewLock.Lock()
	if node.AllClusterNewViewBuffer[Msg.Cluster][Msg.NewViewNumber] != "" { //保证每个消息只接收和转发一次
		//node.NewViewLock.Unlock()
		return nil
	} else {
		node.AllClusterNewViewBuffer[Msg.Cluster][Msg.NewViewNumber] = Msg.NodeID
	}
	fmt.Printf("Get New View Msg form %s\n", Msg.NodeID)
	// Verify Msg
	digestInfo, err := hex.DecodeString(Msg.Digest)
	if err != nil {
		println(err)
		//node.NewViewLock.Unlock()
		return err
	}
	for nodeId, sign := range Msg.VoteNodeMsg {
		if !node.RsaVerySignWithSha256(digestInfo, sign, node.getPubKey(Msg.Cluster, nodeId)) {
			fmt.Printf("%v 投票节点签名信息验证失败！,拒绝New View!!!\n", nodeId)
		}
	}
	if !node.RsaVerySignWithSha256(digestInfo, Msg.Sign, node.getPubKey(Msg.Cluster, Msg.NodeID)) {
		fmt.Println("节点签名验证失败！,拒绝New View!!!")
	}

	PrimaryNode[Msg.Cluster] = Msg.NodeID
	fmt.Printf("\n\nCluster %s Change Primary Node %s\n\n\n", Msg.Cluster, Msg.NodeID)
	node.NewPrimaryNodeID = ""
	if Msg.Cluster == node.ClusterName {
		node.CurrentState.CurrentStage = consensus.Committed
		node.View.Primary = Msg.NodeID
		node.View.Number = Msg.NewViewNumber
		node.View.ID = Msg.ViewID
		fmt.Printf("Msg wait to send waitToSendPendingMsgsIndex:  %v\n", waitToSendPendingMsgsIndex)
	} else { //其他集群的节点收到了当前的消息要共享给本地节点
		node.Broadcast(node.ClusterName, Msg, "/ShareGlobalNewViewMsgToLocalNode")
	}
	//node.NewViewLock.Unlock()

	return nil

}

func (node *Node) GetViewChange(Msg *consensus.ViewChangeMsg) error {
	if node.NewPrimaryNodeID == "" {
		var NewPrimaryNode = ""
		var NewPrimaryNodeScore uint16 = 0
		for i := 0; i < len(node.NodeTable[node.ClusterName]); i++ {
			nodeID := node.ClusterName + strconv.Itoa(i)
			score := node.ReScore[node.ClusterName][nodeID]
			if nodeID == node.View.Primary || node.ActiveCommitteeNode[nodeID] == NonCommittedNode {
				continue
			}
			if score > NewPrimaryNodeScore {
				NewPrimaryNode = nodeID
				NewPrimaryNodeScore = score
			}
		}
		node.NewPrimaryNodeID = NewPrimaryNode
	}

	if node.NewPrimaryNodeID != node.NodeID {
		return nil
	}
	fmt.Printf("Get View Change Msg from %s\n", Msg.NodeID)
	//Verify Msg
	digest, err := consensus.GetDigest(node.MsgBuffer.PendingMsgs[len(node.MsgBuffer.PendingMsgs)-1])
	if err != nil {
		fmt.Println(err)
		return err
	}
	if digest != Msg.LastPendingMsg {
		fmt.Println("View Change Msg Digest Verify Error!!!")
	}
	if node.NewPrimaryNodeID != Msg.NewPrimaryNodeID {
		fmt.Printf("节点 %s 投票失败\n", Msg.NodeID)
		return nil
	}
	digestInfo, _ := hex.DecodeString(digest)
	if !node.RsaVerySignWithSha256(digestInfo, Msg.Sign, node.getPubKey(node.ClusterName, Msg.NodeID)) {
		fmt.Println("节点签名验证失败！,拒绝执行ViewChange!!!")
	}
	if Msg.NewViewNumber != node.View.Number+1 {
		fmt.Println("New View Number Error!!!")
	}
	// 记录 ViewChange
	node.ViewChangeMsgVote[Msg.NodeID] = Msg
	fmt.Printf("%v 个节点为ViewChange消息投票！\n", len(node.ViewChangeMsgVote))
	if len(node.ViewChangeMsgVote) >= 2*consensus.F { // Start New View
		node.View.Primary = node.NodeID
		PrimaryNode[node.ClusterName] = node.NodeID

		node.View.Number += 1

		signInfo := node.RsaSignWithSha256(digestInfo, node.rsaPrivKey)
		NewViewMsg := &consensus.NewView{
			VoteNodeMsg:   make(map[string][]byte),
			NewViewNumber: node.View.Number,
			Digest:        digest,
			NodeID:        node.NodeID,
			Cluster:       node.ClusterName,
			Sign:          signInfo,
			ViewID:        node.View.ID,
		}
		for nodeId, msg := range node.ViewChangeMsgVote {
			NewViewMsg.VoteNodeMsg[nodeId] = msg.Sign
		}
		node.ViewChangeMsgVote = make(map[string]*consensus.ViewChangeMsg) //清空缓存
		node.Broadcast(node.ClusterName, NewViewMsg, "/NewView")
		node.ShareViewChangeMsgToGlobal(NewViewMsg, "/NewViewToGlobal")
		node.CurrentState.CurrentStage = consensus.Committed
		node.NewPrimaryNodeID = ""
		sendRequestNumber = len(node.MsgBuffer.BatchReqMsgs)
		//fmt.Printf("sendRequestNumber: %v,")
		fmt.Printf("New View !!! This Node is New Primary Node\n\n\n\n")
		for _, value := range node.MsgBuffer.PendingMsgs {
			fmt.Printf("Pending Msg: %v  ", value.Requests[0].Operation)
		}
		fmt.Printf("\nwait to send %v", waitToSendPendingMsgsIndex)
		// 检查是否有要全局共识的消息
		node.PrimaryNodeShareMsg()
	}
	return nil
}

// GetReq can be called when the node's CurrentState is nil.
// Consensus start procedure for the Primary.
func (node *Node) GetReq(reqMsg *consensus.BatchRequestMsg, goOn bool) error {
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

	//LogStage(fmt.Sprintf("Consensus Process (ViewID:%d)", node.CurrentState.ViewID), false)

	// Send getPrePrepare message
	if prePrepareMsg != nil {
		// 附加主节点ID,用于数字签名验证
		prePrepareMsg.NodeID = node.NodeID
		fmt.Printf("prePreMsg SeqID:　%d\n", prePrepareMsg.SequenceID)
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
	if prePrepareMsg.NodeID != node.View.Primary {
		return nil
	}

	LogMsg(prePrepareMsg)

	node.AcceptRequestTime[prePrepareMsg.RequestMsg.SequenceID] = time.Now()

	// Create a new state for the new consensus.
	err := node.createStateForNewConsensus(goOn)
	if err != nil {
		return err
	}
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
		if node.NodeType == CommitteeNode && node.MaliciousNode != isMaliciousNode {
			node.Broadcast(node.ClusterName, prePareMsg, "/prepare")
		}
		LogStage("Prepare", false)
	}

	return nil
}

func (node *Node) GetPrepare(prepareMsg *consensus.VoteMsg) error {
	if node.CurrentState.CurrentStage == consensus.Prepared && node.NodeID != node.View.Primary {
		return nil
	}
	if node.ActiveCommitteeNode[prepareMsg.NodeID] == NonCommittedNode {
		return nil
	}

	LogMsg(prepareMsg)

	digest, _ := hex.DecodeString(prepareMsg.Digest)
	if !node.RsaVerySignWithSha256(digest, prepareMsg.Sign, node.getPubKey(node.ClusterName, prepareMsg.NodeID)) {
		fmt.Println("节点签名验证失败！,拒绝执行prepare")
	}
	//主节点是不广播prepare的，所以为自己投一票
	if node.CurrentState.MsgLogs.PrepareMsgs[node.NodeID] == nil && node.NodeID != node.View.Primary {
		node.CurrentState.MsgLogs.PrepareMsgs[node.NodeID] = prepareMsg
	}
	commitMsg, err := node.CurrentState.Prepare(prepareMsg)
	if err != nil {
		ErrMessage(prepareMsg)
		return err
	}

	// 更新本节点收到的活跃节点
	node.ReElement.Active[prepareMsg.NodeID] = 1

	if commitMsg != nil {
		node.ReElement.Active[node.NodeID] = 1
		// Attach node ID to the message 同时对摘要签名
		commitMsg.NodeID = node.NodeID
		signInfo := node.RsaSignWithSha256(digest, node.rsaPrivKey)
		commitMsg.Sign = signInfo

		// 将投票状态共享给其他节点
		for _, value := range node.CurrentState.MsgLogs.PrepareMsgs {
			commitMsg.Score[value.NodeID] = true
		}

		LogStage("Prepare", true)
		if node.NodeType == CommitteeNode && node.MaliciousNode != isMaliciousNode && len(node.CurrentState.MsgLogs.PrepareMsgs) == 2*consensus.F {
			node.Broadcast(node.ClusterName, commitMsg, "/commit")

		}
		LogStage("Commit", false)
	}

	return nil
}

// 保存委员会节点信用分数
func (node *Node) appendScoresToFile(filename string) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// 写入每轮的数据
	_, err = file.WriteString(fmt.Sprintf("Round %d:\n", node.View.ID-10000000000))
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}
	for i := 0; i < 4; i++ {
		nodeId := "N" + strconv.Itoa(i)
		line := fmt.Sprintf("%s: %d\n", nodeId, node.ReScore[node.ClusterName][nodeId])
		_, err := file.WriteString(line)
		if err != nil {
			fmt.Println("Error writing to file:", err)
			return
		}
	}
	//for value, _type := range node.ActiveCommitteeNode {
	//	if _type == CommitteeNode {
	//		line := fmt.Sprintf("%s: %d\n", value, node.ReScore[node.ClusterName][value])
	//		_, err := file.WriteString(line)
	//		if err != nil {
	//			fmt.Println("Error writing to file:", err)
	//			return
	//		}
	//	}
	//}
}

func (node *Node) GetCommit(commitMsg *consensus.VoteMsg) error {
	// 当节点已经完成Committed阶段后就停止接收其他节点的Committed消息，同时不接受非委员会节点的消息
	if node.CurrentState.CurrentStage == consensus.Committed || node.CurrentState.CurrentStage == consensus.ViewChange {
		return nil
	}
	if node.ActiveCommitteeNode[commitMsg.NodeID] == NonCommittedNode {
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
	node.PrimaryNodeExeLock.Lock()
	if replyMsg != nil {
		oldViewID := node.View.ID
		if committedMsg == nil {
			return errors.New("committed message is nil, even though the reply message is not nil")
		}

		// Attach node ID to the message
		replyMsg.NodeID = node.NodeID
		if node.NodeID == node.View.Primary {

			// 更新每个节点的信誉值，首先初始化每个节点的都还没更新分数
			// 先更新每个节点的活跃度
			for _, value := range node.CurrentState.MsgLogs.CommitMsgs {
				node.ReElement.Active[value.NodeID] += consensus.F
				for nodeId, vote := range value.Score {
					if vote == true {
						node.ReElement.Active[nodeId]++
					}
				}
			}

			// 判断节点是否共识成功
			// 记得CommitteeNodeNumber后面换成活跃的委员会节点集合
			sumOfAddScore := 0 // 增加的总分数
			AddNodeNum := 0    // 本轮共识中非恶意节点数
			for nodeID, isActive := range node.ActiveCommitteeNode {
				if isActive != CommitteeNode { //如果不是委员会节点就跳过
					continue
				}
				// 主节点在prepare阶段时不会发送消息的所以不会计算active，直接增加信用值
				if nodeID == node.View.Primary {
					//node.ReScore[node.ClusterName][nodeID] = uint16(min(1000, node.ReScore[node.ClusterName][nodeID]+30))
					continue
				}

				active := node.ReElement.Active[nodeID]
				var historySuccessRate float32 = 0
				success := 0
				var sum float32 = 0
				// 通过nodeID获取对应的整数切片
				scores, exists := node.ReElement.HistoryScore[nodeID]
				if exists {
					// 迭代整数切片并计算总和
					for _, value := range scores {
						sum += float32(value)
					}
					historySuccessRate = sum / float32(len(node.ReElement.HistoryScore[nodeID]))
					fmt.Printf("historySuccessRate %v\n", historySuccessRate)
				}
				historyScore := int(historySuccessRate * 4)
				// 如果活跃度为 0 ，当前节点的此次共识结果为失败，信用值减少
				if active == 0 {
					fmt.Printf("节点 %s 不活跃！\n", nodeID)
					node.ReElement.HistoryScore[nodeID] = append(node.ReElement.HistoryScore[nodeID], -1)
					success = -5 + int(-(50.0 * (1.0 - historySuccessRate)))
					active = -5
				} else {
					AddNodeNum++

					node.ReElement.HistoryScore[nodeID] = append(node.ReElement.HistoryScore[nodeID], 1)
					success = 4
					// 假设 active 和 CommitteeNodeNumber 都是 int 类型
					activeFloat := float64(active)                     // 将 active 转换为浮点数
					committeeFloat := float64(CommitteeNodeNumber - 1) // 将 CommitteeNodeNumber - 1 转换为浮点数

					// 使用浮点数进行计算以保留小数部分
					active = int((activeFloat / committeeFloat) * 4) // 最后将结果转换回 int 类型
					sumOfAddScore += active + success + historyScore
				}
				CurrentScore := int(node.ReScore[node.ClusterName][nodeID])
				fmt.Printf("historyScore %v\n", historyScore)
				//fmt.Printf("Node %s Acitve %d Success %d historySuccessRate %d\n", nodeID, active, success, historySuccessRate)
				node.ReScore[node.ClusterName][nodeID] = uint16(min(1000, max(0, CurrentScore+active+success+historyScore)))
			}
			//主节点的更新分数为所有更增加分数的平均值+10
			primaryAddScore := uint16(sumOfAddScore/AddNodeNum + 2)
			node.ReScore[node.ClusterName][node.View.Primary] = min(1000, node.ReScore[node.ClusterName][node.View.Primary]+primaryAddScore)
			// 每一轮的活跃值要清空
			node.ReElement.Active = make(map[string]int)

			//// 记录信用分值的记得删除
			if node.ClusterName == "N" && node.NodeID == node.View.Primary {
				node.appendScoresToFile("scores.txt")
			}
		}

		LogStage("Commit", true)
		fmt.Printf("ViewID :%d 达成本地共识，存入待执行缓存池\n", node.View.ID)

		// Append msg to its logs
		//node.PendingMsgsLock.Lock()
		//committedMsg.Send = false
		// 集群内同意当前消息的节点和签名
		NodeSign := SignInfo{
			Sign: make(map[string][]byte),
		}
		for nodeID, voteMsg := range node.CurrentState.MsgLogs.CommitMsgs {
			NodeSign.Sign[nodeID] = voteMsg.Sign
		}
		node.MsgBuffer.SignInfo = append(node.MsgBuffer.SignInfo, NodeSign)

		node.MsgBuffer.PendingMsgs = append(node.MsgBuffer.PendingMsgs, committedMsg)

		//for _, value := range node.MsgBuffer.PendingMsgs {
		//	fmt.Printf("Get Commit %v ", value.Requests[0].Operation)
		//}
		//fmt.Printf("\n")

		//go func() {
		if node.NodeID == node.View.Primary { // 本地共识结束后，主节点将本地达成共识的请求发送至其他集群的主节点
			msg := consensus.SyncReScore{
				Score:  make(map[string]uint16),
				NodeID: node.NodeID,
				ViewID: node.View.ID + 1,
			}
			for nodeID, isActive := range node.ActiveCommitteeNode {
				if isActive != CommitteeNode { //如果不是委员会节点就跳过
					continue
				}
				msg.Score[nodeID] = node.ReScore[node.ClusterName][nodeID]
			}
			jsonMsg, err1 := json.Marshal(msg.Score) // 对分值签名
			if err1 != nil {
				fmt.Println(err1)
				return err1
			}
			signInfo := node.RsaSignWithSha256(jsonMsg, node.rsaPrivKey)
			msg.Sign = signInfo
			node.Broadcast(node.ClusterName, msg, "/SyncScore")
			node.View.ID++
			node.PrimaryViewID = node.View.ID
			node.CurrentState.CurrentStage = consensus.Committed
			node.CheckViewChangeLock.Lock()
			node.CheckViewChange()
			node.CheckViewChangeLock.Unlock()
			node.PrimaryNodeShareMsg()

		} else {
			//CompleteTime := time.Since(node.AcceptRequestTime[commitMsg.SequenceID])

			node.View.ID = oldViewID + 1
			node.CurrentState.CurrentStage = consensus.Committed
			node.CheckViewChangeLock.Lock()
			node.CheckViewChange()
			node.CheckViewChangeLock.Unlock()
		}
		//}()

	}
	node.PrimaryNodeExeLock.Unlock()
	return nil
}

var waitToSendPendingMsgsIndex = -1

func (node *Node) PrimaryNodeShareMsg() error {
	// 判断当前节点是代理执行节点，且有要共享的本地共识
	fmt.Printf("\n\n本次全局共识的主节点是%s\n\n\n\n", Allcluster[node.GlobalViewID%int64(ClusterNumber)])
	if Allcluster[node.GlobalViewID%int64(ClusterNumber)] == node.ClusterName && len(node.MsgBuffer.PendingMsgs) > waitToSendPendingMsgsIndex+1 && node.CurrentState.CurrentStage != consensus.ViewChange && node.NodeID == node.View.Primary { // 如果轮询到本地主节点作为代理人，发送消息给全局和本地

		index := waitToSendPendingMsgsIndex
		waitToSendPendingMsgsIndex++
		committedMsg := node.MsgBuffer.PendingMsgs[index+1]
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
		GlobalShareMsg.AddNewCommitteeNodeID = nil

		GlobalShareMsg.SignInfo = make(map[string][]byte)
		for nodeId, sign := range node.MsgBuffer.SignInfo[index+1].Sign {
			GlobalShareMsg.SignInfo[nodeId] = sign
		}

		if GlobalShareMsg.Score == nil {
			GlobalShareMsg.Score = make(map[string]uint16)
		}
		for key, value := range node.ReScore[node.ClusterName] {
			GlobalShareMsg.Score[key] = value
		}

		// 附加节点ID,用于数字签名验证
		sendMsg := &consensus.LocalMsg{
			Sign:                       signInfo,
			NodeID:                     node.NodeID,
			GlobalShareMsg:             GlobalShareMsg,
			WaitToSendPendingMsgsIndex: waitToSendPendingMsgsIndex,
		}

		// 同时调整参与委员会共识的活跃节点列表，删除分数不够的委员会节点
		//initActiveCommitteeNode := 0
		//numOfActiveCommitteeNode := 0
		//for nodeID, isActive := range node.ActiveCommitteeNode {
		//	if isActive != CommitteeNode {
		//		continue
		//	}
		//	initActiveCommitteeNode++
		//	if node.ReScore[node.ClusterName][nodeID] < ReScoretThreshold && false {
		//		if nodeID == node.NodeID {
		//			node.NodeType = NonCommittedNode
		//
		//		}
		//		node.ActiveCommitteeNode[nodeID] = NonCommittedNode
		//		fmt.Printf("修改节点 %s 为非委员会节点！\n", nodeID)
		//	} else {
		//		numOfActiveCommitteeNode++
		//	}
		//}
		//// 从非委员会节点中挑选信用值最高的节点作为委员会节点
		//newAddCommitteeNodeNum := initActiveCommitteeNode - numOfActiveCommitteeNode
		//for i := 0; newAddCommitteeNodeNum > 0; i++ {
		//	var MaxReScore uint16 = 0
		//	var changeID = 0
		//	for j := 0; j < len(node.ReScore[node.ClusterName]); j++ {
		//		if j > 6 { // 测试只有7个节点，后期删除
		//			break
		//		}
		//		nodeID := node.ClusterName + strconv.Itoa(j)
		//		if node.ActiveCommitteeNode[nodeID] == NonCommittedNode {
		//			if node.ReScore[node.ClusterName][nodeID] > MaxReScore && node.ReScore[node.ClusterName][nodeID] >= ReScoretThreshold {
		//				MaxReScore = node.ReScore[node.ClusterName][nodeID]
		//				changeID = j
		//			}
		//		}
		//	}
		//
		//	ChangeNodeID := node.ClusterName + strconv.Itoa(changeID)
		//	node.ActiveCommitteeNode[ChangeNodeID] = CommitteeNode
		//
		//	GlobalShareMsg.AddNewCommitteeNodeID = append(GlobalShareMsg.AddNewCommitteeNodeID, ChangeNodeID)
		//	fmt.Printf("增加节点 %s 为委员会节点!\n", ChangeNodeID)
		//	newAddCommitteeNodeNum--
		//
		//}

		// 根据参与共识的委员会节点总数更新f值
		//consensus.F = numOfActiveCommitteeNode / 3

		node.Broadcast(node.ClusterName, sendMsg, "/GlobalToLocal")
		node.ShareLocalConsensus(GlobalShareMsg, "/global")

		// 主节点达成本地共识，进行全局共识的排序和执行
		node.GlobalViewIDLock.Lock()
		node.Reply(node.GlobalViewID, committedMsg, node.GlobalViewID)
		node.GlobalViewIDLock.Unlock()

		//node.GlobalBufferReqMsgs.Lock()
		////最后检查缓存中有没有收到其他代理主节点的消息，执行
		//if len(node.GlobalBuffer.ReqMsg) != 0 {
		//	tempmsg := node.GlobalBuffer.ReqMsg[0]
		//	if Allcluster[node.GlobalViewID%int64(ClusterNumber)] == tempmsg.Cluster && tempmsg.ViewID == node.GlobalViewID {
		//		node.GlobalBuffer.ReqMsg = node.GlobalBuffer.ReqMsg[1:]
		//		node.ShareGlobalMsgToLocal(tempmsg)
		//	}
		//}
		//node.GlobalBufferReqMsgs.Unlock()
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
		time.Sleep(10 * time.Microsecond)
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
		case msg := <-node.ScoreEntrance:
			err := node.routeScoreMsg(msg)
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
	case *consensus.BatchRequestMsg:
		//备份请求
		node.MsgBuffer.BatchReqMsgs = append(node.MsgBuffer.BatchReqMsgs, msg.(*consensus.BatchRequestMsg))
		fmt.Printf("缓存中收到主节点共享的 %d 条客户端请求\n", len(node.MsgBuffer.BatchReqMsgs))
	}
}

func (node *Node) resolveClientRequest() {
	for {
		select {
		case msg := <-node.MsgRequsetchan:
			node.SaveClientRequest(msg)
			time.Sleep(10 * time.Microsecond)
		}
	}
}

func (node *Node) resolveSyncReScore(msg *consensus.SyncReScore) error {
	fmt.Println("收到主节点的信用值同步消息")
	jsonMsg, _ := json.Marshal(msg.Score)
	if !node.RsaVerySignWithSha256(jsonMsg, msg.Sign, node.getPubKey(node.ClusterName, msg.NodeID)) {
		fmt.Println("信用值同步消息 --- 签名验证失败！！！")
		return nil
	}
	if msg.NodeID != node.View.Primary {
		return nil
	}
	node.PrimaryViewID = msg.ViewID
	for nodeID, score := range msg.Score {
		node.ReScore[node.ClusterName][nodeID] = score
		fmt.Printf("信用值 %s %d ", nodeID, score)
	}
	fmt.Printf("\n")
	fmt.Println("同步信用值成功！")
	node.CheckViewChange()
	return nil
}

func (node *Node) routeScoreMsg(msg interface{}) error {
	switch m := msg.(type) {
	case *consensus.SyncReScore:

		node.resolveSyncReScore(m)
	}
	return nil
}

func (node *Node) routeGlobalMsg(msg interface{}) []error {
	switch m := msg.(type) {
	case *consensus.NewView:
		msgs := make([]*consensus.NewView, 0)
		//copy(msgs, node.GlobalBuffer.NewViewMsg)

		// Append a newly arrived message.
		msgs = append(msgs, msg.(*consensus.NewView))
		//node.GlobalBuffer.NewViewMsg = append(node.GlobalBuffer.NewViewMsg, msg.(*consensus.NewView))
		// Empty the buffer.
		//node.GlobalBuffer.NewViewMsg = make([]*consensus.NewView, 0)

		// Send messages.
		node.MsgGlobalDelivery <- msgs

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
	case *consensus.ViewChangeMsg:
		node.MsgBuffer.ViewChangeMsgs = append(node.MsgBuffer.ViewChangeMsgs, msg.(*consensus.ViewChangeMsg))
	case *consensus.NewView:
		node.MsgBuffer.NewViewMsgs = append(node.MsgBuffer.NewViewMsgs, msg.(*consensus.NewView))
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
		fmt.Printf("                                                                View ID %d,Global ID %d,View Number %d\n", node.View.ID, node.GlobalViewID, node.View.Number)
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
	msg := mb.ReqMsgs[0]                          // 获取第一个元素
	mb.ReqMsgs = mb.ReqMsgs[consensus.BatchSize:] // 移除第一个元素
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

func (mb *MsgBuffer) DequeueNewViewMsg() *consensus.NewView {
	if len(mb.NewViewMsgs) == 0 {
		return nil
	}
	msg := mb.NewViewMsgs[0]
	mb.NewViewMsgs = mb.NewViewMsgs[1:]
	return msg
}

func (mb *MsgBuffer) DequeueViewChangeMsg() *consensus.ViewChangeMsg {
	if len(mb.ViewChangeMsgs) == 0 {
		return nil
	}
	msg := mb.ViewChangeMsgs[0]
	mb.ViewChangeMsgs = mb.ViewChangeMsgs[1:]
	return msg
}

func (node *Node) resolveGlobalMsg() {
	for {
		time.Sleep(10 * time.Microsecond)
		msg := <-node.MsgGlobalDelivery
		// time.Sleep(50 * time.Millisecond)
		switch msg.(type) {
		case []*consensus.NewView:
			errs := node.resolveGlobalNewViewMsg(msg.([]*consensus.NewView))
			if len(errs) != 0 {
				for _, err := range errs {
					fmt.Println(err)
				}
				// TODO: send err to ErrorChannel
			}
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

var sendRequestNumber = 0

const viewID = 10000000000

func (node *Node) resolveMsg() {
	for {
		time.Sleep(10 * time.Microsecond)
		// Get buffered messages from the dispatcher.
		switch {
		case len(node.MsgBuffer.ViewChangeMsgs) > 0 || len(node.MsgBuffer.NewViewMsgs) > 0 || node.CurrentState.CurrentStage == consensus.ViewChange:

			if len(node.MsgBuffer.ViewChangeMsgs) > 0 {
				errs := node.resolveViewChangeMsg(node.MsgBuffer.ViewChangeMsgs[0])
				if errs != nil {
					fmt.Println(errs)
					// TODO: send err to ErrorChannel
				}
				node.MsgBuffer.DequeueViewChangeMsg()
			} else if len(node.MsgBuffer.NewViewMsgs) > 0 {
				errs := node.resolveNewViewMsg(node.MsgBuffer.NewViewMsgs[0])
				if errs != nil {
					fmt.Println(errs)
					// TODO: send err to ErrorChannel
				}
				node.MsgBuffer.DequeueNewViewMsg()
			}
		case len(node.MsgBuffer.ReqMsgs) > 0 && node.NodeID != node.View.Primary: //非主节点收到客户端请求后转发给主节点
			url := node.NodeTable[node.ClusterName][node.View.Primary]
			msg := node.MsgBuffer.ReqMsgs[0]
			node.MsgBuffer.DequeueReqMsg()
			jsonMsg, err := json.Marshal(msg)
			if err != nil {
				fmt.Println(err)
			}
			go func() {
				fmt.Println("收到客户端发来的消息，已经转发给主节点")
				send(url+"/req", jsonMsg)
			}()

		case node.NodeID == node.View.Primary && (len(node.MsgBuffer.ReqMsgs) > consensus.BatchSize-1 || int64(len(node.MsgBuffer.BatchReqMsgs)) > node.View.ID-viewID) && (node.CurrentState.LastSequenceID == -2 || node.CurrentState.CurrentStage == consensus.Committed):
			if len(node.MsgBuffer.ReqMsgs) > consensus.BatchSize-1 {
				// 逐个赋值到数组中
				lengthOfReqMsg := len(node.MsgBuffer.ReqMsgs)
				for i := 0; i < lengthOfReqMsg/consensus.BatchSize; i++ {
					// 初始化batch并确保它是非nil
					var batch consensus.BatchRequestMsg
					// 为每个request获取唯一的SequenceID
					//sequenceID := time.Now().UnixNano()
					// Find the unique and largest number for the sequence ID
					//if node.CurrentState.LastSequenceID > 0 {
					//	for node.CurrentState.LastSequenceID >= sequenceID {
					//		sequenceID += 1
					//	}
					//}

					for j := 0; j < consensus.BatchSize; j++ {
						//	node.MsgBuffer.ReqMsgs[j].SequenceID = sequenceID
						batch.Requests[j] = node.MsgBuffer.ReqMsgs[j]

					}

					batch.Timestamp = node.MsgBuffer.ReqMsgs[0].Timestamp
					batch.ClientID = node.MsgBuffer.ReqMsgs[0].ClientID
					batch.SequenceID = node.MsgBuffer.ReqMsgs[0].SequenceID

					// 添加新的批次到批次消息缓存
					node.MsgBuffer.BatchReqMsgs = append(node.MsgBuffer.BatchReqMsgs, &batch)
					node.MsgBufferLock.ReqMsgsLock.Lock()
					node.MsgBuffer.DequeueReqMsg()
					node.MsgBufferLock.ReqMsgsLock.Unlock()
				}
			}
			errs := node.resolveRequestMsg(node.MsgBuffer.BatchReqMsgs[node.View.ID-viewID])
			if errs != nil {
				fmt.Println(errs)
				// TODO: send err to ErrorChannel
			}

			go func() {
				for sendRequestNumber < len(node.MsgBuffer.BatchReqMsgs) {
					node.Broadcast(node.ClusterName, node.MsgBuffer.BatchReqMsgs[sendRequestNumber], "/reqToLocal")
					sendRequestNumber++
				}
			}()

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
			var keepIndexes []int    // 用于存储需要保留的元素的索引
			var processIndexes []int //存储需要处理的元素索引
			// 首先遍历PrepareMsgs，确定哪些元素需要保留，哪个元素需要处理
			for index, value := range node.MsgBuffer.PrepareMsgs {
				if value.ViewID < node.View.ID {
					// 不需要做任何事，因为这个元素将被删除
				} else if value.ViewID > node.View.ID {
					keepIndexes = append(keepIndexes, index) // 保留这个元素
				} else {
					processIndexes = append(processIndexes, index)
				}
			}
			// 如果找到了符合条件的元素，则处理它
			if len(processIndexes) != 0 {
				var ProcessPrepareMsgs []*consensus.VoteMsg // 假设YourMsgType是PrepareMsgs中元素的类型
				for _, index := range processIndexes {
					ProcessPrepareMsgs = append(ProcessPrepareMsgs, node.MsgBuffer.PrepareMsgs[index])
				}
				errs := node.resolvePrepareMsg(ProcessPrepareMsgs)
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
		case len(node.MsgBuffer.CommitMsgs) > 0 && (node.CurrentState.CurrentStage == consensus.Prepared):
			if len(node.MsgBuffer.PrepareMsgs) > 0 && node.NodeID == node.View.Primary && node.MsgBuffer.PrepareMsgs[0].ViewID == node.View.ID {
				node.MsgBufferLock.PrepareMsgsLock.Lock()
				var keepIndexes []int    // 用于存储需要保留的元素的索引
				var processIndexes []int //存储需要处理的元素索引
				// 首先遍历PrepareMsgs，确定哪些元素需要保留，哪个元素需要处理
				for index, value := range node.MsgBuffer.PrepareMsgs {
					if value.ViewID < node.View.ID {
						// 不需要做任何事，因为这个元素将被删除
					} else if value.ViewID > node.View.ID {
						keepIndexes = append(keepIndexes, index) // 保留这个元素
					} else {
						processIndexes = append(processIndexes, index)
					}
				}
				// 如果找到了符合条件的元素，则处理它
				if len(processIndexes) != 0 {
					var ProcessPrepareMsgs []*consensus.VoteMsg // 假设YourMsgType是PrepareMsgs中元素的类型
					for _, index := range processIndexes {
						ProcessPrepareMsgs = append(ProcessPrepareMsgs, node.MsgBuffer.PrepareMsgs[index])
					}
					errs := node.resolvePrepareMsg(ProcessPrepareMsgs)
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
			}
			node.MsgBufferLock.CommitMsgsLock.Lock()
			var keepIndexes []int    // 用于存储需要保留的元素的索引
			var processIndexes []int //存储需要处理的元素索引
			// 首先遍历PrepareMsgs，确定哪些元素需要保留，哪个元素需要处理
			for index, value := range node.MsgBuffer.CommitMsgs {
				if value.ViewID < node.View.ID {
					// 不需要做任何事，因为这个元素将被删除
				} else if value.ViewID > node.View.ID {
					keepIndexes = append(keepIndexes, index) // 保留这个元素
				} else {
					processIndexes = append(processIndexes, index)
				}
			}
			// 如果找到了符合条件的元素，则处理它
			if len(processIndexes) != 0 {
				var ProcessPrepareMsgs []*consensus.VoteMsg // 假设YourMsgType是PrepareMsgs中元素的类型
				for _, index := range processIndexes {
					ProcessPrepareMsgs = append(ProcessPrepareMsgs, node.MsgBuffer.CommitMsgs[index])
				}
				errs := node.resolveCommitMsg(ProcessPrepareMsgs)
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

func (node *Node) resolveNewViewMsg(msg *consensus.NewView) error {
	node.CheckViewChangeLock.Lock()
	err := node.GetNewView(msg)
	node.CheckViewChangeLock.Unlock()
	if err != nil {
		return err
	}

	return nil
}

func (node *Node) resolveViewChangeMsg(msg *consensus.ViewChangeMsg) error {

	err := node.GetViewChange(msg)
	if err != nil {
		return err
	}

	return nil
}

func (node *Node) resolveRequestMsg(msg *consensus.BatchRequestMsg) error {

	err := node.GetReq(msg, false)
	if err != nil {
		return err
	}

	return nil
}

func (node *Node) resolveGlobalNewViewMsg(msgs []*consensus.NewView) []error {
	errs := make([]error, 0)

	// Resolve messages
	//fmt.Printf("获得其他集群的NewView消息 %d\n", len(msgs))

	for _, reqMsg := range msgs {
		// 收到其他组的消息，转发给其他主节点节点
		err := node.GetNewView(reqMsg)
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
	//fmt.Printf("获得其他节点的全局共识消息 %d\n", len(msgs))

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

func (node *Node) GlobalConsensus(msg *consensus.LocalMsg) (*consensus.ReplyMsg, *consensus.BatchRequestMsg, error) {
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
	waitToSendPendingMsgsIndex = reqMsg.WaitToSendPendingMsgsIndex
	// Attach node ID to the message
	replyMsg.NodeID = node.NodeID
	// Save the last version of committed messages to node.
	// node.CommittedMsgs = append(node.CommittedMsgs, committedMsg)
	fmt.Printf("Global stage ID %s %d\n", reqMsg.GlobalShareMsg.Cluster, reqMsg.GlobalShareMsg.ViewID)
	node.GlobalViewIDLock.Lock()
	// 首先判断当前消息不是来自本地集群的全局共识消息（因为本地的信誉值已经更新过）
	//if reqMsg.GlobalShareMsg.Cluster != node.ClusterName {
	// 更新其他集群节点的信誉值
	for key, value := range reqMsg.GlobalShareMsg.Score {
		node.ReScore[reqMsg.GlobalShareMsg.Cluster][key] = value
	}
	//}

	// 收到的是本地主节点发来的本地共识的全局共识，要更改参与委员会的节点数
	if reqMsg.GlobalShareMsg.Cluster == node.ClusterName {
		numOfActiveCommitteeNode := 0
		for nodeID, isActive := range node.ActiveCommitteeNode {
			if isActive != CommitteeNode {
				continue
			}
			if node.ReScore[node.ClusterName][nodeID] < BaseReScore && false {
				if nodeID == node.NodeID {
					node.NodeType = NonCommittedNode
				}
				node.ActiveCommitteeNode[nodeID] = NonCommittedNode
				fmt.Printf("修改节点 %s 为非委员会节点！\n", nodeID)

			} else {
				numOfActiveCommitteeNode++
			}
		}
		// 如果有要新加入的委员会节点
		if reqMsg.GlobalShareMsg.AddNewCommitteeNodeID != nil {
			for _, value := range reqMsg.GlobalShareMsg.AddNewCommitteeNodeID {
				node.ActiveCommitteeNode[value] = CommitteeNode
				if value == node.NodeID { // 如果就是当前节点则更改本节点的type
					node.NodeType = CommitteeNode
					fmt.Printf("本节点 %s 设置为委员会节点！\n", value)
				}
				fmt.Printf("节点 %s 设置为委员会节点！\n", value)
			}
		}
		//consensus.F = numOfActiveCommitteeNode / 3
	}

	node.Reply(node.GlobalViewID, reqMsg.GlobalShareMsg.RequestMsg, replyMsg.ViewID)

	node.GlobalViewIDLock.Unlock()
	// LogStage("Reply\n", true)
	//if node.CurrentState.CurrentStage != consensus.Prepared {

	return nil
}

// 收到其他集群主节点发来的共识消息
func (node *Node) ShareGlobalMsgToLocal(reqMsg *consensus.GlobalShareMsg) error {
	// 如果是本集群发送的消息不需要接受，如果已经收到过这个消息了也不用接收
	if reqMsg.Cluster == node.ClusterName || reqMsg.ViewID < node.GlobalViewID {
		return nil
	}
	fmt.Printf("\n接收到来自节点%s的 GlobalID = %d op=%s全局共识消息\n", reqMsg.NodeID, reqMsg.ViewID, reqMsg.RequestMsg.Requests[0].Operation)
	// 检查主节点签名信息
	for node.CurrentState.CurrentStage == consensus.ViewChange {

	}
	if node.NodeID != node.View.Primary {
		fmt.Printf("该节点非主节点，已转发给集群主节点: %v", node.View.Primary)
		url := node.NodeTable[node.ClusterName][node.View.Primary] + "/global"
		jsonMsg, err := json.Marshal(reqMsg)
		if err != nil {
			return err
		}
		send(url, jsonMsg)
		return nil
	}

	if Allcluster[node.GlobalViewID%int64(ClusterNumber)] != reqMsg.Cluster {
		fmt.Printf("收到 %s %d 主节点的共识消息，但此时的代理节点为 %s ，需要等待······\n", reqMsg.Cluster, reqMsg.ViewID, Allcluster[node.GlobalViewID%int64(ClusterNumber)])
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
		fmt.Println("主节点签名验证失败！,拒绝执行全局共识")
	}
	// 检查集群内其他节点签名信息
	for nodeId, sign := range reqMsg.SignInfo {
		if !node.RsaVerySignWithSha256(digest, sign, node.getPubKey(reqMsg.Cluster, nodeId)) {
			fmt.Printf("\n%s子节点签名验证失败！,拒绝执行全局共识\n", nodeId)
		}
	}

	//if reqMsg.NodeID != PrimaryNode[reqMsg.Cluster] {
	//	fmt.Printf("非 %s 主节点发送的全局共识，拒绝接受\n", reqMsg.Cluster)
	//	return nil
	//}

	// 节点对消息摘要进行签名
	signInfo := node.RsaSignWithSha256(digest, node.rsaPrivKey)

	// 附加节点ID,用于数字签名验证
	sendMsg := &consensus.LocalMsg{
		Sign:                       signInfo,
		NodeID:                     node.NodeID,
		GlobalShareMsg:             reqMsg,
		WaitToSendPendingMsgsIndex: waitToSendPendingMsgsIndex,
	}

	// 发送给其他主节点和本地节点
	node.ShareLocalConsensus(reqMsg, "/global")
	node.Broadcast(node.ClusterName, sendMsg, "/GlobalToLocal")

	//fmt.Printf("----- 收到其他委员会节点发来的全局共识，已发送给本地节点和其他委员会节点 -----\n")
	// 执行全局共识消息
	node.GlobalViewIDLock.Lock()
	// 更新其他节点的信誉值
	for key, value := range reqMsg.Score {
		node.ReScore[reqMsg.Cluster][key] = value
	}
	node.Reply(node.GlobalViewID, reqMsg.RequestMsg, reqMsg.ViewID)
	node.GlobalViewIDLock.Unlock()

	//if node.CurrentState.CurrentStage != consensus.Prepared {
	if node.CurrentState.CurrentStage == consensus.Committed {

		node.PrimaryNodeExeLock.Lock()
		node.PrimaryNodeShareMsg()
		node.PrimaryNodeExeLock.Unlock()
	}

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

func (node *Node) resolvePrepareMsg(msgs []*consensus.VoteMsg) error {
	// Resolve messages
	///fmt.Printf("len PrepareMsg msg %d\n", len(msgs))
	for _, msg := range msgs {
		err := node.GetPrepare(msg)
		if err != nil {
			return err
		}

	}

	return nil
}

func (node *Node) resolveCommitMsg(msgs []*consensus.VoteMsg) error {

	for _, msg := range msgs {
		err := node.GetCommit(msg)
		if err != nil {
			return err
		}
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
		return false
	}
	return true
}
