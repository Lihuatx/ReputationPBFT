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

// 添加性能监控结构体
type PerformanceMetrics struct {
	Timestamp    string
	HeapAlloc    uint64 // 堆内存分配量
	HeapInUse    uint64 // 正在使用的堆内存
	StackInUse   uint64 // 正在使用的栈内存
	NumGoroutine int    // goroutine数量
}

// 添加性能监控函数
func monitorPerformance(nodeID string) {
	// 创建性能数据目录
	dirName := fmt.Sprintf("performance_data_%s", time.Now().Format("20060102_150405"))
	err := os.MkdirAll(dirName, 0755)
	if err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		return
	}

	// 创建CSV文件
	filename := filepath.Join(dirName, fmt.Sprintf("%s_performance.csv", nodeID))
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer file.Close()

	// 创建CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入表头
	writer.Write([]string{"Timestamp", "HeapAlloc(MB)", "HeapInUse(MB)", "StackInUse(MB)", "NumGoroutine"})

	// 创建计时器，每秒收集一次数据
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var m runtime.MemStats

	fmt.Printf("Started monitoring performance for node %s\n", nodeID)
	fmt.Println("Timestamp | HeapAlloc(MB) | HeapInUse(MB) | StackInUse(MB) | NumGoroutine")
	fmt.Println("-----------------------------------------------------------------")

	for range ticker.C {
		runtime.ReadMemStats(&m)
		metrics := PerformanceMetrics{
			Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
			HeapAlloc:    m.HeapAlloc,
			HeapInUse:    m.HeapInuse,
			StackInUse:   m.StackSys,
			NumGoroutine: runtime.NumGoroutine(),
		}

		// 转换为MB并保留两位小数
		heapAllocMB := float64(metrics.HeapAlloc) / 1024 / 1024
		heapInUseMB := float64(metrics.HeapInUse) / 1024 / 1024
		stackInUseMB := float64(metrics.StackInUse) / 1024 / 1024

		// 打印到控制台
		fmt.Printf("%s | %10.2f | %11.2f | %12.2f | %12d\n",
			metrics.Timestamp,
			heapAllocMB,
			heapInUseMB,
			stackInUseMB,
			metrics.NumGoroutine,
		)

		// 写入CSV
		writer.Write([]string{
			metrics.Timestamp,
			fmt.Sprintf("%.2f", heapAllocMB),
			fmt.Sprintf("%.2f", heapInUseMB),
			fmt.Sprintf("%.2f", stackInUseMB),
			strconv.Itoa(metrics.NumGoroutine),
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
