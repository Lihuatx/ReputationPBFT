package main

import (
	"os"

	"simple_pbft/pbft/network"
)

func main() {
	genRsaKeys()
	nodeID := os.Args[1]
	server := network.NewServer(nodeID)

	server.Start()
}
