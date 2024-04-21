import sys

clusters = ['N', 'M', 'P','J','K']  # 集群的标识符
nodes_per_cluster = 19  # 每个集群的节点数
arg = sys.argv[1]
nodes_per_cluster = int(arg)
server1 = sys.argv[2]
server2 = sys.argv[3]
server3 = sys.argv[4]
server4 = sys.argv[5]
server5 = sys.argv[6]

base_port = 1110  # 基础端口号

# 初始化 NodeTable
node_table = {cluster: {f"{cluster}{i}": f"{server1}:{base_port + i + (clusters.index(cluster) * nodes_per_cluster)}"
                        for i in range(nodes_per_cluster)}
              for cluster in clusters}

node_table_M = {cluster: {f"{cluster}{i}": f"{server2}:{base_port + i + (clusters.index(cluster) * nodes_per_cluster)}"
                           for i in range(nodes_per_cluster)}
                 for cluster in clusters}

node_table_P = {cluster: {f"{cluster}{i}": f"{server3}:{base_port + i + (clusters.index(cluster) * nodes_per_cluster)}"
                           for i in range(nodes_per_cluster)}
                 for cluster in clusters}

node_table_J = {cluster: {f"{cluster}{i}": f"{server4}:{base_port + i + (clusters.index(cluster) * nodes_per_cluster)}"
                           for i in range(nodes_per_cluster)}
                 for cluster in clusters}

node_table_K = {cluster: {f"{cluster}{i}": f"{server5}:{base_port + i + (clusters.index(cluster) * nodes_per_cluster)}"
                           for i in range(nodes_per_cluster)}
                 for cluster in clusters}

# 将 NodeTable 保存到 nodetable.txt 文件中
with open('nodetable.txt', 'w') as file:
    for cluster, nodes in node_table.items():
        if cluster == "N":
            for node_id, address in nodes.items():
                file.write(f"{cluster} {node_id} {address}\n")
                if node_id == "N0":
                    print(address)

    for cluster, nodes in node_table_M.items():
        if cluster == "M":
            for node_id, address in nodes.items():
                file.write(f"{cluster} {node_id} {address}\n")
                if node_id == "M0":
                    print(address)

    for cluster, nodes in node_table_P.items():
        if cluster == "P":
            for node_id, address in nodes.items():
                file.write(f"{cluster} {node_id} {address}\n")
                if node_id == "P0":
                    print(address)

    for cluster, nodes in node_table_P.items():
        if cluster == "J":
            for node_id, address in nodes.items():
                file.write(f"{cluster} {node_id} {address}\n")
                if node_id == "J0":
                    print(address)

    for cluster, nodes in node_table_P.items():
        if cluster == "K":
            for node_id, address in nodes.items():
                file.write(f"{cluster} {node_id} {address}\n")
                if node_id == "K0":
                    print(address)