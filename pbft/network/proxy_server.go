package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"simple_pbft/pbft/consensus"
)

type Server struct {
	url  string
	node *Node
}

func NewServer(nodeID string, clusterName string) *Server {
	node := NewNode(nodeID, clusterName)
	server := &Server{node.NodeTable[clusterName][nodeID], node}

	server.setRoute()

	return server
}

func (server *Server) Start() {
	fmt.Printf("Server will be started at %s...\n", server.url)
	if err := http.ListenAndServe(server.url, nil); err != nil {
		fmt.Println(err)
		return
	}
}

func (server *Server) setRoute() {
	http.HandleFunc("/req", server.getReq)
	http.HandleFunc("/preprepare", server.getPrePrepare)
	http.HandleFunc("/prepare", server.getPrepare)
	http.HandleFunc("/commit", server.getCommit)
	http.HandleFunc("/reply", server.getReply)
	//接受全局共识消息
	http.HandleFunc("/global", server.getGlobal)
	http.HandleFunc("/GlobalToLocal", server.getGlobalToLocal)

}

func (server *Server) getReq(writer http.ResponseWriter, request *http.Request) {
	var msg consensus.RequestMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	// for test
	if err != nil {
		fmt.Println(err)
		return
	}
	// 保存请求的URL到RequestMsg中
	msg.URL = request.URL.String()

	fmt.Printf("\nhttp get RequestMsg op : %s\n", msg.Operation)
	server.node.MsgEntrance <- &msg
}

func (server *Server) getPrePrepare(writer http.ResponseWriter, request *http.Request) {
	var msg consensus.PrePrepareMsg
	err := json.NewDecoder(request.Body).Decode(&msg)

	if err != nil {
		fmt.Println(err)
		return
	}

	server.node.MsgEntrance <- &msg
}

func (server *Server) getPrepare(writer http.ResponseWriter, request *http.Request) {
	var msg consensus.VoteMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		fmt.Println(err)
		return
	}

	server.node.MsgEntrance <- &msg
}

func (server *Server) getCommit(writer http.ResponseWriter, request *http.Request) {
	var msg consensus.VoteMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		fmt.Println(err)
		return
	}

	server.node.MsgEntrance <- &msg
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

func (server *Server) getGlobal(writer http.ResponseWriter, request *http.Request) {
	var msg consensus.GlobalShareMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	// fmt.Printf("http1 getGlobal receive %s\n", msg.NodeID)
	server.node.MsgGlobal <- &msg
}

func (server *Server) getGlobalToLocal(writer http.ResponseWriter, request *http.Request) {
	var msg consensus.LocalMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	// fmt.Printf("http2 getGlobalToLocal receive %s\n", msg.NodeID)
	server.node.MsgGlobal <- &msg
}

func send(url string, msg []byte) {
	buff := bytes.NewBuffer(msg)
	http.Post("http://"+url, "application/json", buff)
}
