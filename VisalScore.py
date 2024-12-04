import matplotlib.pyplot as plt


def read_and_average_scores(file_list):
    # 初始化数据结构，假设每个文件有相同的轮次数和节点
    scores_dict = {f'N{i}': [] for i in range(4)}
    count_files = len(file_list)

    for filename in file_list:
        with open(filename, 'r') as file:
            lines = file.readlines()

        temp_data = {f'N{i}': [] for i in range(4)}  # 临时存储单个文件的数据
        for line in lines:
            if line.startswith('N'):
                node_id, score = line.split(':')
                score = int(score.strip())
                temp_data[node_id.strip()].append(score)

        # 将当前文件的数据累加到总数据中
        for node_id in scores_dict:
            if len(scores_dict[node_id]) == 0:  # 如果是第一个文件，直接赋值
                scores_dict[node_id] = temp_data[node_id]
            else:  # 否则累加得分
                scores_dict[node_id] = [sum(x) for x in zip(scores_dict[node_id], temp_data[node_id])]

    # 计算平均得分
    average_scores = {node_id: [score / count_files for score in scores] for node_id, scores in scores_dict.items()}
    return average_scores


def plot_data(data):
    plt.figure(figsize=(10, 8))  # 调整图片尺寸为正方形
    # 设置字体大小
    font = {'size': 18}
    plt.rc('font', **font)
    for node_id, scores in data.items():
        rounds = list(range(len(scores)))  # 生成轮次列表
        plt.plot(rounds, scores, marker='o', label=node_id)

    plt.xlabel('Transaction Round')
    plt.ylabel('Trust Score')
    plt.legend()
    plt.grid(True)
    # 获取当前图像
    fig = plt.gcf()

    # 设置标题并将其放置在图像下方
    fig.text(0.5, 0.005, '(a) Four Honest Nodes', ha='center', fontsize=16)

    plt.savefig('HonestNodeScore.png')
    #plt.savefig('HonestNodeScore.png')

    plt.show()

def plot_data2(data):
    plt.figure(figsize=(10, 8))  # 调整图片尺寸为正方形
    # 设置字体大小
    font = {'size': 18}
    plt.rc('font', **font)
    for node_id, scores in data.items():
        rounds = list(range(len(scores)))  # 生成轮次列表
        plt.plot(rounds, scores, marker='o', label=node_id)

    plt.xlabel('Transaction Round')
    plt.ylabel('Trust Score')
    plt.legend()
    plt.grid(True)
    # 获取当前图像
    fig = plt.gcf()

    # 设置标题并将其放置在图像下方
    fig.text(0.5, 0.005, '(b) One Byzantine Node(Misbehaves from the Start)', ha='center', fontsize=16)

    plt.savefig('StartByzantineNodeScore.png')
    #plt.savefig('HonestNodeScore.png')

    plt.show()

def plot_data3(data):
    plt.figure(figsize=(10, 8))  # 调整图片尺寸为正方形
    # 设置字体大小
    font = {'size': 18}
    plt.rc('font', **font)
    for node_id, scores in data.items():
        rounds = list(range(len(scores)))  # 生成轮次列表
        plt.plot(rounds, scores, marker='o', label=node_id)

    plt.xlabel('Transaction Round')
    plt.ylabel('Trust Score')
    plt.legend()
    plt.grid(True)
    # 获取当前图像
    fig = plt.gcf()

    # 设置标题并将其放置在图像下方
    fig.text(0.5, 0.005, '(c) One Byzantine Node(Misbehaves from the Middle)', ha='center', fontsize=16)

    plt.savefig('MidByzantineNodeScore.png')
    #plt.savefig('HonestNodeScore.png')

    plt.show()

# 文件名从 'scores1.txt' 到 'scores5.txt'
file_list = [f'scores{i}.txt' for i in range(1, 5)]
average_scores = read_and_average_scores(file_list)
plot_data(average_scores)

# 文件名从 'scores1.txt' 到 'scores5.txt'
file_list = [f'scores{i}.txt' for i in range(10, 13)]
average_scores = read_and_average_scores(file_list)
plot_data2(average_scores)

# 文件名从 'scores1.txt' 到 'scores5.txt'
file_list = [f'scores{i}.txt' for i in range(20, 23)]
average_scores = read_and_average_scores(file_list)
plot_data3(average_scores)



# def read_scores(filename):
#     with open(filename, 'r') as file:
#         lines = file.readlines()
#
#     data = {'N0': [], 'N1': [], 'N2': [], 'N3': []}  # 初始化节点数据字典
#     for line in lines:
#         if line.startswith('N'):
#             node_id, score = line.split(':')
#             score = int(score.strip())
#             data[node_id.strip()].append(score)
#
#     return data
#
# def plot_data(data):
#     plt.figure(figsize=(10, 5))
#     for node_id, scores in data.items():
#         rounds = list(range(len(scores)))  # 生成轮次列表
#         plt.plot(rounds, scores, marker='o', label=node_id)
#
#     plt.title('Node Scores Over Rounds')
#     plt.xlabel('Round Number')
#     plt.ylabel('Score')
#     plt.legend()
#     plt.grid(True)
#     plt.show()
#
# # 假设你的文件名为 'scores.txt'
# data = read_scores('scores.txt')
# plot_data(data)

