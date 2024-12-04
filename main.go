package main

import (
	"My_PBFT/pbft/network"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
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

type PerformanceMetrics struct {
	Timestamp    string
	HeapAlloc    uint64
	HeapInuse    uint64
	StackInUse   uint64
	NumGoroutine int
	TotalSys     uint64 // 改用 Sys 来表示总内存
}

func monitorPerformance(nodeID string) {
	dirName := fmt.Sprintf("performance_data_%s", time.Now().Format("20060102_150405"))
	err := os.MkdirAll(dirName, 0755)
	if err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		return
	}

	filename := filepath.Join(dirName, fmt.Sprintf("%s_performance.csv", nodeID))
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"Timestamp", "NodeID", "HeapAlloc(MB)", "HeapInuse(MB)", "StackInUse(MB)", "NumGoroutine", "TotalAlloc(MB)"})

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var m runtime.MemStats
	startTime := time.Now()

	fmt.Printf("Started monitoring performance for node %s\n", nodeID)
	fmt.Println("Timestamp | NodeID | HeapAlloc(MB) | HeapInuse(MB) | StackInUse(MB) | NumGoroutine | TotalAlloc(MB)")
	fmt.Println("-----------------------------------------------------------------------------------------")

	for range ticker.C {
		if time.Since(startTime) >= 20*time.Second {
			fmt.Printf("Monitoring completed for node %s after 60 seconds\n", nodeID)
			break
		}

		runtime.ReadMemStats(&m)
		metrics := PerformanceMetrics{
			Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
			HeapAlloc:    m.HeapAlloc,
			HeapInuse:    m.HeapInuse,
			StackInUse:   m.StackSys,
			NumGoroutine: runtime.NumGoroutine(),
			TotalSys:     m.Sys, // 使用 Sys 而不是 TotalAlloc
		}

		heapAllocMB := float64(metrics.HeapAlloc) / 1024 / 1024
		heapInuseMB := float64(metrics.HeapInuse) / 1024 / 1024
		stackInUseMB := float64(metrics.StackInUse) / 1024 / 1024
		totalSysMB := float64(metrics.TotalSys) / 1024 / 1024 // 总系统内存

		// 打印格式修改
		fmt.Printf("%s | %6s | %11.2f MB | %12.2f MB | %13.2f MB | %11d | %12.2f MB (Total Sys)\n",
			metrics.Timestamp,
			nodeID,
			heapAllocMB,
			heapInuseMB,
			stackInUseMB,
			metrics.NumGoroutine,
			totalSysMB,
		)

		writer.Write([]string{
			metrics.Timestamp,
			nodeID,
			fmt.Sprintf("%.2f", heapAllocMB),
			fmt.Sprintf("%.2f", heapInuseMB),
			fmt.Sprintf("%.2f", stackInUseMB),
			strconv.Itoa(metrics.NumGoroutine),
			fmt.Sprintf("%.2f", totalSysMB),
		})
		writer.Flush()
	}
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

	// 对于非client节点启动性能监控
	if nodeID != "client" {
		go monitorPerformance(nodeID)
	}

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
