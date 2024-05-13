import matplotlib.pyplot as plt


# def read_and_average_scores(file_list):
#     # 初始化数据结构，假设每个文件有相同的轮次数和节点
#     scores_dict = {f'N{i}': [] for i in range(4)}
#     count_files = len(file_list)
#
#     for filename in file_list:
#         with open(filename, 'r') as file:
#             lines = file.readlines()
#
#         temp_data = {f'N{i}': [] for i in range(4)}  # 临时存储单个文件的数据
#         for line in lines:
#             if line.startswith('N'):
#                 node_id, score = line.split(':')
#                 score = int(score.strip())
#                 temp_data[node_id.strip()].append(score)
#
#         # 将当前文件的数据累加到总数据中
#         for node_id in scores_dict:
#             if len(scores_dict[node_id]) == 0:  # 如果是第一个文件，直接赋值
#                 scores_dict[node_id] = temp_data[node_id]
#             else:  # 否则累加得分
#                 scores_dict[node_id] = [sum(x) for x in zip(scores_dict[node_id], temp_data[node_id])]
#
#     # 计算平均得分
#     average_scores = {node_id: [score / count_files for score in scores] for node_id, scores in scores_dict.items()}
#     return average_scores
#
#
# def plot_data(data):
#     plt.figure(figsize=(10, 5))
#     for node_id, scores in data.items():
#         rounds = list(range(len(scores)))  # 生成轮次列表
#         plt.plot(rounds, scores, marker='o', label=node_id)
#
#     plt.title('Average Node Scores Over Rounds')
#     plt.xlabel('Round Number')
#     plt.ylabel('Average Score')
#     plt.legend()
#     plt.grid(True)
#     plt.show()
#
#
# # 文件名从 'scores1.txt' 到 'scores5.txt'
# file_list = [f'scores{i}.txt' for i in range(1, 2)]
# average_scores = read_and_average_scores(file_list)
# plot_data(average_scores)

def read_scores(filename):
    with open(filename, 'r') as file:
        lines = file.readlines()

    data = {'N0': [], 'N1': [], 'N2': [], 'N3': []}  # 初始化节点数据字典
    for line in lines:
        if line.startswith('N'):
            node_id, score = line.split(':')
            score = int(score.strip())
            data[node_id.strip()].append(score)

    return data

def plot_data(data):
    plt.figure(figsize=(10, 5))
    for node_id, scores in data.items():
        rounds = list(range(len(scores)))  # 生成轮次列表
        plt.plot(rounds, scores, marker='o', label=node_id)

    plt.title('Node Scores Over Rounds')
    plt.xlabel('Round Number')
    plt.ylabel('Score')
    plt.legend()
    plt.grid(True)
    plt.show()

# 假设你的文件名为 'scores.txt'
data = read_scores('scores.txt')
plot_data(data)

