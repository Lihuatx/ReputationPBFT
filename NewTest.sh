#!/bin/bash

# $1: 每个集群的节点数量
# $2: 委员会节点数量

n=$1
z=$2
localhost="0.0.0.0"

# 创建节点表
python3 CreateNodeTable2.py "$n" "$localhost"

# 关闭已存在的tmux会话
tmux kill-session -t myPBFT

# 创建并启动所有集群的节点
for cluster in N M P J K
do
    python3 CreateCluster2.py "$cluster" "$n" "$z"
done

# 启动客户端
python3 LinuxTest2.py "N"
