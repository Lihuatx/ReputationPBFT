import subprocess
import time
import sys
import threading


# 定义命令模板和数量
command_template = 'app.exe'
groups = ['N']
arg = sys.argv[2]
nodes_per_group = int(arg)
z = int(sys.argv[3])
committeeNodeNum = int(nodes_per_group / 2)
errorNodeNum = int(nodes_per_group * 0.3)
errorCommitteeNodeNum = int(committeeNodeNum * 0.3)
errorNonCommitteeNodeNum = errorNodeNum - errorCommitteeNodeNum

temp = []
# while errorNodeNum > 0:
#     temp.append(int(nodes_per_group/2) - errorNodeNum - 1)
#     errorNodeNum-=1
#
# errorNodeList = ["N" + str(i) for i in temp]
# 生成命令列表
errorCommitteeNodeNumList = [str(group) + str(i) for group in groups for i in range(1, errorCommitteeNodeNum + 1)]
errorNonCommitteeNodeNumList = [str(group) + str(i) for group in groups for i in range(committeeNodeNum, errorNonCommitteeNodeNum + committeeNodeNum)]
# 生成命令列表
commands = [(command_template, f'{group}{i}', group) for group in groups for i in range(nodes_per_group)]
print(errorCommitteeNodeNumList)
print(errorNonCommitteeNodeNumList)

i = 0
# 遍历命令和参数，然后在新的命令提示符窗口中执行
for exe, arg1, arg2 in commands:
    i+=1
    if i > z * nodes_per_group:
        break
    # 如果 app.exe 路径中包含空格，确保使用引号括起来
    if arg1 in errorNonCommitteeNodeNumList or arg1 in errorCommitteeNodeNumList:
        command = f'start cmd /k "{exe}" {arg1} {arg2} {z} {nodes_per_group} Y'
    else:
        command = f'start cmd /k "{exe}" {arg1} {arg2} {z} {nodes_per_group}'
    subprocess.Popen(command, shell=True)

time.sleep(1)

def start_command(arg):
    command = f'start cmd /k "{command_template}" client {arg}'
    subprocess.Popen(command, shell=True)

i = 0
for arg in groups:
    i += 1
    if i > z:
        break
    # 创建并启动线程
    thread = threading.Thread(target=start_command, args=(arg,))
    thread.start()



