import subprocess
import time
import sys

# 定义命令模板和数量
command_template = 'app.exe'
groups = ['N', 'M', 'P','J',"K"]
arg = sys.argv[2]
nodes_per_group = int(arg)
z = int(sys.argv[3])

# 生成命令列表
commands = [(command_template, f'{group}{i}', group) for group in groups for i in range(nodes_per_group)]

# 遍历命令和参数，然后在新的命令提示符窗口中执行
for exe, arg1, arg2 in commands:
    # 如果 app.exe 路径中包含空格，确保使用引号括起来
    command = f'start cmd /k "{exe}" {arg1} {arg2} {z}'
    subprocess.Popen(command, shell=True)

time.sleep(1)


