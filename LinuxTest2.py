import subprocess
import sys

def start_clients():
    # 创建新的tmux会话用于客户端
    subprocess.run(['tmux', 'kill-session', '-t', 'myClient'])
    subprocess.run(['tmux', 'new-session', '-d', '-s', 'myClient'])

    # 为每个集群创建一个客户端
    clusters = ['N', 'M', 'P', 'J', 'K']
    for i, cluster in enumerate(clusters):
        # 为每个集群创建一个新窗口
        window_name = f"Client-{cluster}"
        subprocess.run(['tmux', 'new-window', '-t', f'myClient:{i+1}', '-n', window_name])

        # 启动客户端
        tmux_command = f"tmux send-keys -t myClient:{i+1} './app client {cluster}' C-m"
        subprocess.Popen(['bash', '-c', tmux_command])

if __name__ == "__main__":
    start_clients()