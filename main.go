package main

import (
	"os"
	"simple_pbft/pbft/network"
)

func main() {
	genRsaKeys("N")
	genRsaKeys("M")
	genRsaKeys("P")
	nodeID := os.Args[1]
	clusterName := os.Args[2]
	// 设置默认值
	isMaliciousNode := "no" // 假设默认情况下节点不是恶意的

	// 检查是否提供了第三个参数
	if len(os.Args) > 3 { // 判断节点是正常节点还是恶意节点
		isMaliciousNode = os.Args[3] // 使用提供的第三个参数
	}

	server := network.NewServer(nodeID, clusterName, isMaliciousNode)

	server.Start()

}
