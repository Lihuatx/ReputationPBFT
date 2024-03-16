import subprocess
import time

# 定义要在新终端中执行的命令及其参数
commands = [
    ('app.exe', 'N0', 'N'),
    ('app.exe', 'N1', 'N'),
    ('app.exe', 'N2', 'N'),
    ('app.exe', 'N3', 'N'),
    ('app.exe', 'M0', 'M'),
    ('app.exe', 'M1', 'M'),
    ('app.exe', 'M2', 'M'),
    ('app.exe', 'M3', 'M'),
    #('app.exe', 'P0', 'P'),
    #('app.exe', 'P1', 'P'),
    #('app.exe', 'P2', 'P'),
    #('app.exe', 'P3', 'P'),
]

# 遍历命令和参数，然后在新的命令提示符窗口中执行
for exe, arg1, arg2 in commands:
    # 如果 app.exe 路径中包含空格，确保使用引号括起来
    command = f'start cmd /k "{exe}" {arg1} {arg2}'
    subprocess.Popen(command, shell=True)

time.sleep(2)

# 定义第五个终端要执行的PowerShell命令
ps_command = """
$headers = @{ "Content-Type" = "application/json" }
$body = '{"clientID":"ahnhwi","operation":"SendMes1","timestamp":859381532}'
$response = Invoke-WebRequest -Uri "http://localhost:1111/req" -Method POST -Headers $headers -Body $body
"""

# 定义第五个终端要执行的PowerShell命令
ps_command2 = """
$headers = @{ "Content-Type" = "application/json" }
$body = '{"clientID":"ahnhwi","operation":"SendMes2","timestamp":859381532}'
$response = Invoke-WebRequest -Uri "http://localhost:1116/req" -Method POST -Headers $headers -Body $body
"""



# 定义第五个终端要执行的PowerShell命令
ps_command3 = """
$headers = @{ "Content-Type" = "application/json" }
$body = '{"clientID":"ahnhwi","operation":"GetMyName","timestamp":859381532}'
$response = Invoke-WebRequest -Uri "http://localhost:1121/req" -Method POST -Headers $headers -Body $body
"""

# 在新的PowerShell窗口中执行第五个命令
subprocess.Popen(['powershell', '-Command', ps_command])

# time.sleep(2)


subprocess.Popen(['powershell', '-Command', ps_command2])


#subprocess.Popen(['powershell', '-Command', ps_command3])

