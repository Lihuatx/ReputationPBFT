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

subprocess.run(['tmux', 'new-session', '-d', '-s', 'mySession'])

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
    # ('./app', 'P0', 'P'),
    # ('./app', 'P1', 'P'),
    # ('./app', 'P2', 'P'),
    # ('./app', 'P3', 'P'),
]

# 遍历命令和参数，然后在新的终端窗口中执行
for exe, arg1, arg2 in commands:
    tmux_command = f"tmux send-keys -t mySession '{exe} {arg1} {arg2}' C-m"
    subprocess.run(['bash', '-c', tmux_command])

time.sleep(2)

# 定义要执行的curl命令来代替PowerShell命令
curl_commands = [
    "curl -X POST -H 'Content-Type: application/json' -d '{\"clientID\":\"ahnhwi\",\"operation\":\"SendMes1\",\"timestamp\":859381532}' http://localhost:1111/req",
    "curl -X POST -H 'Content-Type: application/json' -d '{\"clientID\":\"ahnhwi\",\"operation\":\"SendMes2\",\"timestamp\":859381532}' http://localhost:1116/req",
    "curl -X POST -H 'Content-Type: application/json' -d '{\"clientID\":\"ahnhwi\",\"operation\":\"GetMyName\",\"timestamp\":859381532}' http://localhost:1121/req"
]

for curl_command in curl_commands:
    tmux_command = f"tmux send-keys -t mySession '{curl_command}' C-m"
    subprocess.run(['bash', '-c', tmux_command])
