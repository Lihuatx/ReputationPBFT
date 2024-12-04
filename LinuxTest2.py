import subprocess
import time
import sys

def start_clients():
    try:
        # 先清理已存在的客户端session
        subprocess.run(['tmux', 'kill-session', '-t', 'myClient'], stderr=subprocess.DEVNULL, stdout=subprocess.DEVNULL)
        time.sleep(1)  # 等待session完全关闭

        # 创建新的tmux会话用于客户端
        subprocess.run(['tmux', 'new-session', '-d', '-s', 'myClient'])
    except Exception as e:
        print(f"Session handling error: {e}")

    # 为每个集群创建一个客户端
    clusters = ['N', 'M', 'P', 'J', 'K']
    for i, cluster in enumerate(clusters):
        try:
            # 为每个集群创建一个新窗口
            window_name = f"Client-{cluster}"
            subprocess.run(['tmux', 'new-window', '-t', f'myClient:{i+1}', '-n', window_name])
            time.sleep(0.1)  # 短暂延时避免创建太快

            # 启动客户端
            tmux_command = f"tmux send-keys -t myClient:{i+1} './app client {cluster}' C-m"
            subprocess.Popen(['bash', '-c', tmux_command])
        except Exception as e:
            print(f"Error creating client window for cluster {cluster}: {e}")

if __name__ == "__main__":
    start_clients()