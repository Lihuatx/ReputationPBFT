clusters = ['N', 'M', 'P']  # 集群的标识符
nodes_per_cluster = 19  # 每个集群的节点数
base_port = 1110  # 基础端口号

# 初始化 NodeTable
node_table = {cluster: {f"{cluster}{i}": f"47.107.59.211:{base_port + i + (clusters.index(cluster) * nodes_per_cluster)}"
                        for i in range(nodes_per_cluster)}
              for cluster in clusters}

node_table_MP = {cluster: {f"{cluster}{i}": f"114.55.130.178:{base_port + i + (clusters.index(cluster) * nodes_per_cluster)}"
                           for i in range(nodes_per_cluster)}
                 for cluster in clusters}

# 将 NodeTable 保存到 nodetable.txt 文件中
with open('nodetable.txt', 'w') as file:
    for cluster, nodes in node_table.items():
        if cluster == "N":
            for node_id, address in nodes.items():
                file.write(f"{cluster} {node_id} {address}\n")
    for cluster, nodes in node_table_MP.items():
        if cluster != "N":
            for node_id, address in nodes.items():
                file.write(f"{cluster} {node_id} {address}\n")
