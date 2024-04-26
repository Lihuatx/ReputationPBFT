package network

import (
	"encoding/json"
	"fmt"
	"net/http"
	"simple_pbft/pbft/consensus"
)

func ClientStart(name string) *Client {
	client := NewClient(name)
	client.setRoute()

	return client
}
func (client *Client) setRoute() {
	http.HandleFunc("/reply", client.getReply)

}

func (client *Client) Start() {
	fmt.Printf("Client will be started at %s...\n", client.url)
	if err := http.ListenAndServe(client.url, nil); err != nil {
		fmt.Println(err)
		return
	}
}

func (client *Client) getReply(writer http.ResponseWriter, request *http.Request) {
	var msg consensus.ReplyMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		fmt.Println(err)
		return
	}

	client.GetReply(msg)
}
