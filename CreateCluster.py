import subprocess
import time
import sys

def run_commands(arg):
    print("Starting commands...")

    # 执行 go build 命令
    print("Building Go application...")
    subprocess.run(['go', 'build', '-o', 'app'])

    # 等待一段时间以确保编译完成
    print("Waiting for build to finish...")
    time.sleep(1)

    subprocess.run(['tmux', 'new-session', '-d', '-s', 'myPBFT'])

    commands = [
        ('./app', 'N0', 'N'),
        ('./app', 'N1', 'N'),
        ('./app', 'N2', 'N'),
        ('./app', 'N3', 'N'),
        ('./app', 'N4', 'N'),
        ('./app', 'N5', 'N'),
        ('./app', 'N6', 'N'),
        ('./app', 'N7', 'N'),
        ('./app', 'N8', 'N'),
        ('./app', 'N9', 'N'),
        ('./app', 'M0', 'M'),
        ('./app', 'M1', 'M'),
        ('./app', 'M2', 'M'),
        ('./app', 'M3', 'M'),
        ('./app', 'M4', 'M'),
        ('./app', 'M5', 'M'),
        ('./app', 'M6', 'M'),
        ('./app', 'M7', 'M'),
        ('./app', 'M8', 'M'),
        ('./app', 'M9', 'M'),
        ('./app', 'P0', 'P'),
        ('./app', 'P1', 'P'),
        ('./app', 'P2', 'P'),
        ('./app', 'P3', 'P'),
        ('./app', 'P4', 'P'),
        ('./app', 'P5', 'P'),
        ('./app', 'P6', 'P'),
        ('./app', 'P7', 'P'),
        ('./app', 'P8', 'P'),
        ('./app', 'P9', 'P'),
    ]

    # 根据提供的 arg 值过滤命令
    filtered_commands = [(exe, arg1, arg2) for exe, arg1, arg2 in commands if arg2 == arg]

    # 遍历过滤后的命令列表
    for index, (exe, arg1, arg2) in enumerate(filtered_commands):
        window_name = f"app-{arg1}"
        subprocess.run(['tmux', 'new-window', '-t', f'myPBFT:{index + 1}', '-n', window_name])
        time.sleep(0.1)

        tmux_command = f"tmux send-keys -t myPBFT:{index + 1} '{exe} {arg1} {arg2}' C-m"
        subprocess.run(['bash', '-c', tmux_command])

    time.sleep(2)

def run_commands_MP():
    print("Starting commands...")

    # 执行 go build 命令
    print("Building Go application...")
    subprocess.run(['go', 'build', '-o', 'app'])

    # 等待一段时间以确保编译完成
    print("Waiting for build to finish...")
    time.sleep(1)

    subprocess.run(['tmux', 'new-session', '-d', '-s', 'myPBFT'])

    commands = [
        ('./app', 'N0', 'N'),
        ('./app', 'N1', 'N'),
        ('./app', 'N2', 'N'),
        ('./app', 'N3', 'N'),
        ('./app', 'N4', 'N'),
        ('./app', 'N5', 'N'),
        ('./app', 'N6', 'N'),
        ('./app', 'N7', 'N'),
        ('./app', 'N8', 'N'),
        ('./app', 'N9', 'N'),
        ('./app', 'M0', 'M'),
        ('./app', 'M1', 'M'),
        ('./app', 'M2', 'M'),
        ('./app', 'M3', 'M'),
        ('./app', 'M4', 'M'),
        ('./app', 'M5', 'M'),
        ('./app', 'M6', 'M'),
        ('./app', 'M7', 'M'),
        ('./app', 'M8', 'M'),
        ('./app', 'M9', 'M'),
        ('./app', 'P0', 'P'),
        ('./app', 'P1', 'P'),
        ('./app', 'P2', 'P'),
        ('./app', 'P3', 'P'),
        ('./app', 'P4', 'P'),
        ('./app', 'P5', 'P'),
        ('./app', 'P6', 'P'),
        ('./app', 'P7', 'P'),
        ('./app', 'P8', 'P'),
        ('./app', 'P9', 'P'),
    ]

    # 根据提供的 arg 值过滤命令
    filtered_commands = [(exe, arg1, arg2) for exe, arg1, arg2 in commands if arg2 != "N"]

    # 遍历过滤后的命令列表
    for index, (exe, arg1, arg2) in enumerate(filtered_commands):
        window_name = f"app-{arg1}"
        subprocess.run(['tmux', 'new-window', '-t', f'myPBFT:{index + 1}', '-n', window_name])
        time.sleep(0.1)

        tmux_command = f"tmux send-keys -t myPBFT:{index + 1} '{exe} {arg1} {arg2}' C-m"
        subprocess.run(['bash', '-c', tmux_command])

    time.sleep(2)

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python script.py <arg>")
        sys.exit(1)
    arg = sys.argv[1]
    if arg == "N":
        run_commands(arg)
    else:
        run_commands_MP()
