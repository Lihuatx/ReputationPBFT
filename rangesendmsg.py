import subprocess
import time

# 开始执行命令，类似于Bash脚本中的 set -x
print("Starting commands...")

# 执行 go build 命令
print("Building Go application...")
subprocess.run(['go', 'build', '-o', 'app'])

# 等待一段时间以确保编译完成
print("Waiting for build to finish...")
time.sleep(1)

subprocess.run(['tmux', 'new-session', '-d', '-s', 'myPBFT'])

# 确保你的app文件有执行权限
commands = [
    ('./app', 'N0', 'N'),
    ('./app', 'N1', 'N'),
    ('./app', 'N2', 'N'),
    ('./app', 'N3', 'N'),
    ('./app', 'M0', 'M'),
    ('./app', 'M1', 'M'),
    ('./app', 'M2', 'M'),
    ('./app', 'M3', 'M'),
    ('./app', 'P0', 'P'),
    ('./app', 'P1', 'P'),
    ('./app', 'P2', 'P'),
    ('./app', 'P3', 'P'),
]

# 遍历命令和参数，然后在新的终端窗口中执行
# 为每个命令创建新的 Tmux 窗口，并在该窗口中执行命令
index = 0
for index, (exe, arg1, arg2) in enumerate(commands):
    # 为每个进程创建新窗口，窗口名为 "app-Nx"
    window_name = f"app-{arg1}"
    subprocess.run(['tmux', 'new-window', '-t', f'myPBFT:{index + 1}', '-n', window_name])
    time.sleep(0.1)

    # 构建在新窗口中执行的命令
    tmux_command = f"tmux send-keys -t myPBFT:{index + 1} '{exe} {arg1} {arg2}' C-m"
    subprocess.run(['bash', '-c', tmux_command])

time.sleep(2)

# 定义要执行的curl命令来代替PowerShell命令
curl_commands = [
    "curl -H 'Content-Type: application/json' -X POST -d '{\"clientID\":\"ahnhwi\",\"operation\":\"SendMsg1\",\"timestamp\":859381532}' http://localhost:1111/req",
    "curl -H 'Content-Type: application/json' -X POST -d '{\"clientID\":\"ahnhwi\",\"operation\":\"SendMsg2\",\"timestamp\":859381532}' http://localhost:1116/req",
    "curl -H 'Content-Type: application/json' -X POST -d '{\"clientID\":\"ahnhwi\",\"operation\":\"SendMsg3\",\"timestamp\":859381532}' http://localhost:1121/req",
]

# 在 Tmux 会话中添加一个新窗口，用于执行 curl 命令
curl_window_name = "curl-commands"
subprocess.run(['tmux', 'new-window', '-t', 'myPBFT', '-n', curl_window_name])

# 等待一段时间确保窗口创建完成
time.sleep(1)

# 获取刚创建的 curl 命令窗口的索引
# 假设这是紧接着之前创建的窗口，其索引为 len(commands) + 1
curl_window_index = len(commands) + 1

# 将 curl 命令写入一个临时脚本文件
with open("curl_commands.sh", "w") as script_file:
    for i in range(100):
        for base_command in curl_commands:
            # 使用字符串的 format 方法将 i 插入到 operation 字段的值中
            modified_command = base_command.replace("SendMsg1", "SendMsg1-{}".format(i)).replace("SendMsg2", "SendMsg2-{}".format(i)).replace("SendMsg3", "SendMsg3-{}".format(i))
            script_file.write(modified_command + "\n")

# 然后在 Tmux 中执行这个脚本
tmux_command = f'tmux send-keys -t myPBFT:{curl_window_index} "bash curl_commands.sh" C-m'
subprocess.run(['bash', '-c', tmux_command])

