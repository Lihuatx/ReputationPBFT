package main

import (
	"My_PBFT/pbft/network"
	"fmt"
	"os"
	"strconv"
)

func saveScoresToText(filename string) {
	// 初始化文件
	file, err := os.Create(filename)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()
}

func main() {
	genRsaKeys("N")
	genRsaKeys("M")
	genRsaKeys("P")
	genRsaKeys("J")
	genRsaKeys("K")
	saveScoresToText("scores.txt") // 初始化文件并准备写入

	// 检查是否提供了第三个参数
	if len(os.Args) < 3 { // 判断节点是正常节点还是恶意节点
		fmt.Println("参数不足！")
	}

	nodeID := os.Args[1]
	clusterName := os.Args[2]
	sendMsgNumber := 1
	if nodeID == "client" {
		client := network.ClientStart(clusterName)

		go client.SendMsg(sendMsgNumber)

		client.Start()
	} else {
		network.ClusterNumber, _ = strconv.Atoi(os.Args[3])
		clusterNodeNumbers, _ := strconv.Atoi(os.Args[4])

		// 设置默认值
		isMaliciousNode := "no" // 假设默认情况下节点不是恶意的

		// 检查是否提供了第三个参数
		if len(os.Args) > 5 { // 判断节点是正常节点还是恶意节点
			isMaliciousNode = os.Args[5] // 使用提供的第三个参数
		}
		network.PrimaryNodeChangeFreq = 10000
		network.CommitteeNodeNumber = clusterNodeNumbers / 2
		if network.CommitteeNodeNumber < 4 {
			network.CommitteeNodeNumber = 4
		}

		server := network.NewServer(nodeID, clusterName, isMaliciousNode)

		server.Start()
	}
}
