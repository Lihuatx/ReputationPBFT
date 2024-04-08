package consensus

type RequestMsg struct {
	Timestamp  int64  `json:"timestamp"`
	ClientID   string `json:"clientID"`
	Operation  string `json:"operation"`
	SequenceID int64  `json:"sequenceID"`
	URL        string `json:"url"` // 新增URL字段
	Send       bool   `json:"Send"`
}

type ReplyMsg struct {
	ViewID    int64  `json:"viewID"`
	Timestamp int64  `json:"timestamp"`
	ClientID  string `json:"clientID"`
	NodeID    string `json:"nodeID"`
	Result    string `json:"result"`
}

type PrePrepareMsg struct {
	ViewID     int64       `json:"viewID"`
	SequenceID int64       `json:"sequenceID"`
	Digest     string      `json:"digest"`
	NodeID     string      `json:"nodeID"` //添加nodeID
	RequestMsg *RequestMsg `json:"requestMsg"`
	Sign       []byte      `json:"sign"` // 如果你想在 JSON 中包含 Sign 字段
}

type VoteMsg struct {
	ViewID     int64  `json:"viewID"`
	SequenceID int64  `json:"sequenceID"`
	Digest     string `json:"digest"`
	NodeID     string `json:"nodeID"`
	MsgType    `json:"msgType"`
	Sign       []byte          `json:"sign"` // 如果你想在 JSON 中包含 Sign 字段
	Score      map[string]bool `json:"score"`
}

type GlobalShareMsg struct {
	Cluster               string            `json:"ClusterName"`
	NodeID                string            `json:"nodeID"`
	RequestMsg            *RequestMsg       `json:"requestMsg"`
	Digest                string            `json:"digest"`
	Sign                  []byte            `json:"sign"` // 如果你想在 JSON 中包含 Sign 字段
	ViewID                int64             `json:"viewID"`
	Score                 map[string]uint16 `json:"score"`
	AddNewCommitteeNodeID []string          `json:"AddNewCommitteeNodeID"` // 用于替换信用值不够的节点
}

// 在这里LocalMsg是上层主节点委员会中的消息
type LocalMsg struct {
	GlobalShareMsg *GlobalShareMsg `json:"globalShareMsg"`
	NodeID         string          `json:"nodeID"`
	Sign           []byte          `json:"sign"` // 如果你想在 JSON 中包含 Sign 字段
}

type MsgType int

const (
	PrepareMsg MsgType = iota
	CommitMsg
)

var GlobalViewID int
