## 这是测试不同node num 参数的实验

import subprocess
import sys
import threading
import time

import saveData

exeCluster = sys.argv[1]
cluster_num = "5 "
node_numList = ["10 ","16 ","22 ","28 ","34 ","40 "]
node_numList = ["45 ","65 ","80 ","90 ","100 "]
node_numList = ["40 "]
#node_numList = ["90 ","100 "]

PrimaryClusterWaitTime = 12

# 定义集群中的不同模式以及服务器IP（可以按实际情况填入具体IP地址）
clusters = ['N', 'M', 'P', 'J', 'K']
cmd_head = "./test.sh "
base_server_ips = ["43.135.76.128", "43.131.229.242", "43.133.172.45", "43.156.33.133", "43.128.203.61"]

def BatchTest(node_num, cluster_num):
    testCnt = 0
    xls_file = 'data.xls'
    # 检查 data.xls 文件是否存在，不存在则创建
    if not saveData.exists(xls_file):
        with open(xls_file, 'w') as xls:
            xls.write(f"Duration time(N = {node_num} Z = {cluster_num})\n")
    else:
        with open(xls_file, 'a') as xls:
            xls.write(f"Duration time(N = {node_num} Z = {cluster_num})\n")
    while testCnt < 5:
        print(f"\n--- Test count {testCnt + 1}")

        cmd_thread = threading.Thread(target=startCmd, args=(node_num, cluster_num))
        cmd_thread.start()
        cmd_thread.join()  # 确保每次命令执行完毕后再继续

        saveData.monitor_file()
        testCnt += 1
        if exeCluster == "N":
            time.sleep(PrimaryClusterWaitTime)
    print(f"测试完成: (N = {node_num} Z = {cluster_num})\n")

def startCmd(node_num, cluster_num):
    # 遍历每个集群模式生成并执行命令
    for i, mode in enumerate(clusters):
        # 复制基础IP列表以用于修改
        server_ips = base_server_ips.copy()
        # 当前模式对应的服务器IP设置为"0.0.0.0"
        server_ips[i] = "0.0.0.0"
        # 生成命令字符串
        cmd = cmd_head + node_num + cluster_num + ' '.join(server_ips) + ' ' + mode
        # 打印生成的命令
        if exeCluster == clusters[i]:
            print("Executing command:", cmd)
            # 使用subprocess.run执行命令
            subprocess.run(cmd, shell=True)

if __name__ == "__main__":
    for node_num in node_numList:
        BatchTest(node_num, cluster_num)
