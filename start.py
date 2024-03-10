import subprocess
import time

# 定义要在新终端中执行的命令及其参数
commands = [
    ('app.exe', 'N0'),
    ('app.exe', 'N1'),
    ('app.exe', 'N2'),
    ('app.exe', 'N3')
]

# 遍历命令和参数，然后在新的命令提示符窗口中执行
for exe, arg in commands:
    subprocess.Popen(f'start cmd /k {exe} {arg}', shell=True)

time.sleep(5)

# 定义第五个终端要执行的PowerShell命令
ps_command = """
$headers = @{ "Content-Type" = "application/json" }
$body = '{"clientID":"ahnhwi","operation":"GetMyName","timestamp":859381532}'
$response = Invoke-WebRequest -Uri "http://localhost:1111/req" -Method POST -Headers $headers -Body $body
"""

# 在新的PowerShell窗口中执行第五个命令
subprocess.Popen(['powershell', '-Command', ps_command])
