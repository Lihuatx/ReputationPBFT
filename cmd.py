import subprocess
import sys
import threading
import time

import saveData

exeCluster = sys.argv[1]
cluster_num = sys.argv[3] + " "
node_num = sys.argv[2] + " "
PrimaryClusterWaitTime = 5

# 定义集群中的不同模式以及服务器IP（可以按实际情况填入具体IP地址）
clusters = ['N', 'M', 'P', 'J', 'K']
cmd_head = "./test.sh "
#cluster_num = "4 "
#node_num = "5 "
base_server_ips = ["119.28.135.250", "150.109.254.120", "124.156.223.221", "43.156.31.64", "43.133.121.124"]


def BatchTest():
    testCnt = 0
    while testCnt < 10:
        print(f"\n--- Test count {testCnt + 1}")
        cmd_thread = threading.Thread(target=startCmd)
        cmd_thread.start()
        cmd_thread.join()  # 确保每次命令执行完毕后再继续

        saveData.monitor_file()
        testCnt += 1
        if exeCluster == "N":
            time.sleep(PrimaryClusterWaitTime)
    print("测试完成")

def startCmd():
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
    BatchTest()



