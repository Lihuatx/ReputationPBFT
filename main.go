package main

import (
	"os"

	"simple_pbft/pbft/network"
)

func main() {
	genRsaKeys("N")
	genRsaKeys("M")
	nodeID := os.Args[1]
	server := network.NewServer(nodeID)

	server.Start()

}
