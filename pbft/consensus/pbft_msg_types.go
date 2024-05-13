package consensus

type RequestMsg struct {
	Timestamp  int64  `json:"timestamp"`
	ClientID   string `json:"clientID"`
	Operation  string `json:"operation"`
	SequenceID int64  `json:"sequenceID"`
}

type BatchRequestMsg struct {
	Requests   [BatchSize]*RequestMsg `json:"Requests"`
	Timestamp  int64                  `json:"timestamp"`
	ClientID   string                 `json:"clientID"`
	SequenceID int64                  `json:"sequenceID"`
}

type ReplyMsg struct {
	ViewID    int64  `json:"viewID"`
	Timestamp int64  `json:"timestamp"`
	ClientID  string `json:"clientID"`
	NodeID    string `json:"nodeID"`
	Result    string `json:"result"`
}

type PrePrepareMsg struct {
	ViewID     int64            `json:"viewID"`
	SequenceID int64            `json:"sequenceID"`
	Digest     string           `json:"digest"`
	NodeID     string           `json:"nodeID"` //添加nodeID
	RequestMsg *BatchRequestMsg `json:"requestMsg"`
	Sign       []byte           `json:"sign"`
}

type VoteMsg struct {
	ViewID     int64  `json:"viewID"`
	SequenceID int64  `json:"sequenceID"`
	Digest     string `json:"digest"`
	NodeID     string `json:"nodeID"`
	MsgType    `json:"msgType"`
	Sign       []byte          `json:"sign"`
	Score      map[string]bool `json:"score"`
}

type ViewChangeMsg struct {
	NewViewNumber    int64  `json:"NewViewNumber"`
	NewPrimaryNodeID string `json:"NewPrimaryNodeID"`
	LastViewID       int64  `json:"LastViewID"`
	LastPendingMsg   string `json:"digest"`
	NodeID           string `json:"nodeID"`
	Sign             []byte `json:"sign"`
}

type NewView struct {
	VoteNodeMsg   map[string][]byte `json:"NodeSignInfo"`
	NewViewNumber int64             `json:"NewViewNumber"`
	NodeID        string            `json:"nodeID"`
	Cluster       string            `json:"ClusterName"`
	Digest        string            `json:"digest"`
	Sign          []byte            `json:"sign"`
	ViewID        int64             `json:"ViewID"`
}

type SyncReScore struct {
	Score  map[string]uint16 `json:"score"`
	NodeID string            `json:"nodeID"`
	Sign   []byte            `json:"sign"`
	ViewID int64             `json:"ViewID"` // 这个viewID用于判断同步和主节点之间的信用值分数值
}

type GlobalShareMsg struct {
	Cluster               string            `json:"ClusterName"`
	NodeID                string            `json:"nodeID"`
	RequestMsg            *BatchRequestMsg  `json:"requestMsg"`
	Digest                string            `json:"digest"`
	Sign                  []byte            `json:"sign"`
	ViewID                int64             `json:"viewID"`
	Score                 map[string]uint16 `json:"score"`
	SignInfo              map[string][]byte `json:"SignInfo"`              //集群内部节点对该消息的签名
	AddNewCommitteeNodeID []string          `json:"AddNewCommitteeNodeID"` // 用于替换信用值不够的节点
}

// 在这里LocalMsg是上层主节点委员会中的消息
type LocalMsg struct {
	GlobalShareMsg             *GlobalShareMsg `json:"globalShareMsg"`
	NodeID                     string          `json:"nodeID"`
	Sign                       []byte          `json:"sign"`
	WaitToSendPendingMsgsIndex int             `json:"waitToSendPendingMsgsIndex"`
}

type MsgType int

const (
	PrepareMsg MsgType = iota
	CommitMsg
)

const BatchSize = 5

var GlobalViewID int
