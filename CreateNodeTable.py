clusters = ['N', 'M', 'P']  # 集群的标识符
nodes_per_cluster = 7  # 每个集群的节点数
base_port = 1110  # 基础端口号

# 初始化 NodeTable
node_table = {cluster: {f"{cluster}{i}": f"localhost:{base_port + i + (clusters.index(cluster) * nodes_per_cluster)}"
                        for i in range(nodes_per_cluster)}
              for cluster in clusters}

# 将 NodeTable 保存到 nodetable.txt 文件中
with open('nodetable.txt', 'w') as file:
    for cluster, nodes in node_table.items():
        for node_id, address in nodes.items():
            file.write(f"{cluster} {node_id} {address}\n")
