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
	server := network.NewServer(nodeID, clusterName)

	server.Start()

}
