import subprocess
import time
import sys

# 定义命令模板和数量
command_template = './app'
groups = ['N', 'M', 'P', 'J', 'K']
nodes_per_group = 19
arg = sys.argv[2]
nodes_per_group = int(arg)
z = int(sys.argv[3])

# 生成命令列表
commands = [(command_template, f'{group}{i}', group) for group in groups for i in range(nodes_per_group)]

def run_commands(arg):
    print("Building Go application...")
    subprocess.run(['go', 'build', '-o', 'app'])

    # 只在启动第一个集群(N)时创建新的tmux session
    if arg == 'N':
        try:
            subprocess.run(['tmux', 'kill-session', '-t', 'myPBFT'], stderr=subprocess.DEVNULL, stdout=subprocess.DEVNULL)
            time.sleep(1)  # 等待session完全关闭
            subprocess.run(['tmux', 'new-session', '-d', '-s', 'myPBFT'])
        except Exception as e:
            print(f"Session handling error: {e}")

    # 为每个集群分配不同的窗口索引范围
    base_index = {'N': 0, 'M': 50, 'P': 100, 'J': 150, 'K': 200}
    start_index = base_index[arg]

    # 根据提供的 arg 值过滤命令
    filtered_commands = [(exe, arg1, arg2) for exe, arg1, arg2 in commands if arg2 == arg]

    # 遍历过滤后的命令列表
    for i, (exe, arg1, arg2) in enumerate(filtered_commands):
        window_name = f"app-{arg1}"
        window_index = start_index + i + 1

        try:
            subprocess.run(['tmux', 'new-window', '-t', f'myPBFT:{window_index}', '-n', window_name])
            time.sleep(0.1)  # 稍微等待一下，避免创建窗口太快

            tmux_command = f"tmux send-keys -t myPBFT:{window_index} '{exe} {arg1} {arg2} {z} {nodes_per_group}' C-m"
            subprocess.run(['bash', '-c', tmux_command])
        except Exception as e:
            print(f"Error creating window {window_name}: {e}")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python script.py <arg>")
        sys.exit(1)
    arg = sys.argv[1] # clusterName
    run_commands(arg)