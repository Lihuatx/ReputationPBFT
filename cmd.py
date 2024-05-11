import subprocess
import sys

exeCluster = sys.argv[1]
cluster_num = sys.argv[2] + " "
node_num = sys.argv[3] + " "

# 定义集群中的不同模式以及服务器IP（可以按实际情况填入具体IP地址）
clusters = ['N', 'M', 'P', 'J', 'K']
cmd_head = "./test.sh "
#cluster_num = "4 "
#node_num = "5 "
base_server_ips = ["server1", "server2", "server3", "server4", "server5"]

# 遍历每个集群模式生成并执行命令
for i, mode in enumerate(clusters):
    # 复制基础IP列表以用于修改
    server_ips = base_server_ips.copy()
    # 当前模式对应的服务器IP设置为"0.0.0.0"
    server_ips[i] = "0.0.0.0"
    # 生成命令字符串
    cmd = cmd_head + cluster_num + node_num + ' '.join(server_ips) + ' ' + mode
    # 打印生成的命令
    if exeCluster == clusters[i]:
        print("Executing command:", cmd)
        # 使用subprocess.run执行命令
        subprocess.run(cmd, shell=True)
