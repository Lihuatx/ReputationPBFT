package network

import (
	"My_PBFT/pbft/consensus"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"
)

var lock sync.Mutex

type reply struct {
	msg       consensus.RequestMsg
	startTime time.Time
}

var ClientURL = map[string]string{
	"N": "127.0.0.1:1000",
	"M": "127.0.0.1:1001",
	"P": "127.0.0.1:1002",
	"J": "127.0.0.1:1003",
	"K": "127.0.0.1:1004",
}

type Client struct {
	ClientID      string
	url           string
	cluster       string
	NodeTable     map[string]map[string]string // key=nodeID, value=url
	msgTimeLog    map[int64]reply
	sendMsgNumber int
}

func NewClient(clusterName string) *Client {
	client := &Client{
		ClientID:   "Client-" + clusterName,
		url:        ClientURL[clusterName],
		cluster:    clusterName,
		msgTimeLog: make(map[int64]reply),
	}
	PreTime = time.Now()
	return client
}

func (client *Client) SendMsg(sendMsgNumber int) error {
	client.NodeTable = LoadNodeTable("nodetable.txt")
	client.sendMsgNumber = sendMsgNumber
	primary := client.cluster + "0"
	url := client.NodeTable[client.cluster][primary]
	msg := consensus.RequestMsg{
		ClientID:  client.ClientID,
		Timestamp: time.Now().UnixNano(),
	}
	for i := 0; i < sendMsgNumber; i++ {
		Timestamp := time.Now().UnixNano()
		if Timestamp <= msg.Timestamp {
			Timestamp++
		}
		msg.Timestamp = Timestamp
		msg.Operation = "msg: " + client.ClientID + strconv.Itoa(i)
		jsonMsg, err := json.Marshal(msg)
		if err != nil {
			return err
		}

		fmt.Printf("Client Send request Size of JSON message: %d bytes\n", len(jsonMsg))
		send(url+"/req", jsonMsg)
		lock.Lock()
		client.msgTimeLog[msg.Timestamp] = reply{
			msg:       msg,
			startTime: time.Now(),
		}
		lock.Unlock()
	}
	return nil
}

var PreTime time.Time

func (client *Client) GetReply(msg consensus.ReplyMsg) {
	lock.Lock()
	// 定义正则表达式，匹配字符串中的数字
	re := regexp.MustCompile(`\d+`)
	// 查找匹配的子字符串
	numbers := re.FindString(client.msgTimeLog[msg.Timestamp].msg.Operation)
	number, _ := strconv.Atoi(numbers)
	duration := time.Since(client.msgTimeLog[msg.Timestamp].startTime)
	if number%50 == 0 {
		cha := time.Since(PreTime)
		fmt.Printf("msg %s took %s   时间差: %s \n", client.msgTimeLog[msg.Timestamp].msg.Operation, duration, cha)
		// 创建文件并写入 duration
		file, err := os.Create("eachConsensusCostTIme.txt")
		if err != nil {
			log.Fatal("Cannot create file", err)
		}
		defer file.Close()

		// 写入持续时间到文件
		_, err = file.WriteString(cha.String())
		if err != nil {
			log.Fatal("Cannot write to file", err)
		}
		PreTime = time.Now()
	}
	cmd := "msg: Client-" + client.cluster + strconv.Itoa(client.sendMsgNumber-1)
	if client.msgTimeLog[msg.Timestamp].msg.Operation == cmd {
		fmt.Println("save Time!!!")
		// 创建文件并写入 duration
		file, err := os.Create("costTime.txt")
		if err != nil {
			log.Fatal("Cannot create file", err)
		}
		defer file.Close()

		// 写入持续时间到文件
		_, err = file.WriteString(duration.String())
		if err != nil {
			log.Fatal("Cannot write to file", err)
		}

	}
	//fmt.Printf("msg %s took %s   时间差: %s \n", client.msgTimeLog[msg.Timestamp].msg.Operation, duration, cha)

	lock.Unlock()
}
