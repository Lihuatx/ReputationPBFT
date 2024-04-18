import matplotlib.pyplot as plt

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
