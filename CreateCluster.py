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
    print("Starting commands...")

    # 执行 go build 命令
    print("Building Go application...")
    subprocess.run(['go', 'build', '-o', 'app'])

    # 等待一段时间以确保编译完成
    print("Waiting for build to finish...")
    print("Create %s cluster" % arg)
    time.sleep(1)

    subprocess.run(['tmux', 'new-session', '-d', '-s', 'myPBFT'])


    # 根据提供的 arg 值过滤命令
    filtered_commands = [(exe, arg1, arg2) for exe, arg1, arg2 in commands if arg2 == arg]

    # 遍历过滤后的命令列表
    index = 0
    for index, (exe, arg1, arg2) in enumerate(filtered_commands):
        window_name = f"app-{arg1}"
        subprocess.run(['tmux', 'new-window', '-t', f'myPBFT:{index + 1}', '-n', window_name])
        time.sleep(0.1)

        tmux_command = f"tmux send-keys -t myPBFT:{index + 1} '{exe} {arg1} {arg2} {z} {nodes_per_group}' C-m"
        subprocess.run(['bash', '-c', tmux_command])
    # 启动客户端
    time.sleep(1)
    # 遍历过滤后的命令列表


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python script.py <arg>")
        sys.exit(1)
    arg = sys.argv[1] # clusterName
    run_commands(arg)

