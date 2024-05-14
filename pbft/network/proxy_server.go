package network

import (
	"My_PBFT/pbft/consensus"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

func init() {
	// 配置 http.DefaultTransport
	http.DefaultTransport.(*http.Transport).DialContext = (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext
	http.DefaultTransport.(*http.Transport).MaxIdleConns = 200
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 100
	http.DefaultTransport.(*http.Transport).IdleConnTimeout = 90 * time.Second
	http.DefaultTransport.(*http.Transport).TLSHandshakeTimeout = 10 * time.Second
}

type Server struct {
	url  string
	node *Node
}

var flag = false

func NewServer(nodeID string, clusterName string, isMaliciousNode string) *Server {
	node := NewNode(nodeID, clusterName, isMaliciousNode)
	server := &Server{node.NodeTable[clusterName][nodeID], node}

	server.setRoute()

	return server
}

func (server *Server) Start() {
	fmt.Printf("Server will be started at %s...\n", server.url)
	// 创建一个自定义的http.Server
	srv := &http.Server{
		Addr:         server.url,
		Handler:      http.DefaultServeMux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  90 * time.Second,
	}
	listener, err := net.Listen("tcp", server.url)
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}

	if err := srv.Serve(listener); err != nil {
		fmt.Println("Error running server:", err)
	}
}

func (server *Server) setRoute() {
	http.HandleFunc("/req", server.getReq)
	http.HandleFunc("/preprepare", server.getPrePrepare)
	http.HandleFunc("/prepare", server.getPrepare)
	http.HandleFunc("/commit", server.getCommit)
	http.HandleFunc("/ViewChange", server.getViewChange)
	http.HandleFunc("/NewView", server.getNewView)
	http.HandleFunc("/SyncScore", server.getScore)
	http.HandleFunc("/reply", server.getReply)
	//接受全局共识消息
	http.HandleFunc("/ShareGlobalNewViewMsgToLocalNode", server.getGlobalNewViewMsgFromPrimary)
	http.HandleFunc("/NewViewToGlobal", server.getGlobalNewView)
	http.HandleFunc("/global", server.getGlobal)
	http.HandleFunc("/GlobalToLocal", server.getGlobalToLocal)

	//飞线
	http.HandleFunc("/reqToLocal", server.getReqToLocal)

}

func (server *Server) getScore(writer http.ResponseWriter, request *http.Request) {
	var msg *consensus.SyncReScore
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		fmt.Println("Decode error:", err)

		// 记录原始请求体以便调试
		bodyBytes, _ := ioutil.ReadAll(request.Body) // 注意: 这应该在Decode之前做
		fmt.Println("Received body:", string(bodyBytes))

		return
	}

	server.node.ScoreEntrance <- msg
}

func (server *Server) getNewView(writer http.ResponseWriter, request *http.Request) {
	var msg *consensus.NewView
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		fmt.Println(err)
		return
	}

	server.node.MsgEntrance <- msg
}

func (server *Server) getViewChange(writer http.ResponseWriter, request *http.Request) {
	var msg *consensus.ViewChangeMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	// for test
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("\nhttp get ViewChange Msg from: %s\n", msg.NodeID)
	server.node.MsgEntrance <- msg
}

func (server *Server) getReq(writer http.ResponseWriter, request *http.Request) {
	var msg *consensus.RequestMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	// for test
	if err != nil {
		fmt.Println(err)
		return
	}
	// 保存请求的URL到RequestMsg中
	if !flag {
		start = time.Now()
		flag = true
	}

	fmt.Printf("\nhttp get RequestMsg op : %s\n", msg.Operation)
	server.node.MsgRequsetchan <- msg
}

func (server *Server) getPrePrepare(writer http.ResponseWriter, request *http.Request) {
	var msg *consensus.PrePrepareMsg
	err := json.NewDecoder(request.Body).Decode(&msg)

	if err != nil {
		fmt.Println(err)
		return
	}

	server.node.MsgEntrance <- msg
}

func (server *Server) getPrepare(writer http.ResponseWriter, request *http.Request) {
	var msg *consensus.VoteMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		fmt.Println(err)
		return
	}

	server.node.MsgEntrance <- msg
}

func (server *Server) getCommit(writer http.ResponseWriter, request *http.Request) {
	var msg *consensus.VoteMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		fmt.Println(err)
		return
	}

	server.node.MsgEntrance <- msg
}

func (server *Server) getReply(writer http.ResponseWriter, request *http.Request) {
	var msg consensus.ReplyMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		fmt.Println(err)
		return
	}

	server.node.GetReply(&msg)
}

func (server *Server) getGlobalNewView(writer http.ResponseWriter, request *http.Request) {
	var msg *consensus.NewView
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	server.node.MsgGlobal <- msg
}

func (server *Server) getGlobalNewViewMsgFromPrimary(writer http.ResponseWriter, request *http.Request) {
	var msg *consensus.NewView
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	server.node.MsgGlobal <- msg
}

func (server *Server) getGlobal(writer http.ResponseWriter, request *http.Request) {
	var msg *consensus.GlobalShareMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	// fmt.Printf("http1 getGlobal receive %s\n", msg.NodeID)
	server.node.MsgGlobal <- msg
}

func (server *Server) getGlobalToLocal(writer http.ResponseWriter, request *http.Request) {
	var msg *consensus.LocalMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	// fmt.Printf("http2 getGlobalToLocal receive %s\n", msg.NodeID)
	server.node.MsgGlobal <- msg
}

func (server *Server) getReqToLocal(writer http.ResponseWriter, request *http.Request) {
	var msg *consensus.BatchRequestMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	// for test
	if err != nil {
		fmt.Println(err)
		return
	}

	server.node.MsgRequsetchan <- msg
}

func send(url string, msg []byte) {
	buff := bytes.NewBuffer(msg)
	http.Post("http://"+url, "application/json", buff)
}
