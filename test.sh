#!/bin/bash

# 第一个参数赋值给 n
n=$1

# 第二个参数赋值给 z
z=$2

# 第三个参数赋值给 client
client=$3

# 第四个参数赋值给 server
server=$4

# 第五个参数赋值给 Cluster
Cluster=$5

# 调用 Python 脚本，传递变量值
python3 CreateNodeTable.py "$n" "$client" "$server"

# 关闭名为 myPBFT 的 tmux 会话
tmux kill-session -t myPBFT

# 调用另一个 Python 脚本，传递 Cluster 变量的值
python3 CreateCluster.py "$Cluster" "$n"
