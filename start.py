import subprocess
import time

# 定义要在新终端中执行的命令及其参数
commands = [
    ('app.exe', 'N0', 'N','n'),
    ('app.exe', 'N1', 'N','n'),
    ('app.exe', 'N2', 'N','n'),
    ('app.exe', 'N3', 'N', 'n'),
    ('app.exe', 'N4', 'N','n'),
    ('app.exe', 'N5', 'N','n'),
    ('app.exe', 'N6', 'N','n'),
    ('app.exe', 'M0', 'M','n'),
    ('app.exe', 'M1', 'M','n'),
    ('app.exe', 'M2', 'M','n'),
    ('app.exe', 'M3', 'M','n'),
    ('app.exe', 'M4', 'M','n'),
    ('app.exe', 'M5', 'M','n'),
    ('app.exe', 'M6', 'M','n'),
    ('app.exe', 'P0', 'P','n'),
    ('app.exe', 'P1', 'P','n'),
    ('app.exe', 'P2', 'P','n'),
    ('app.exe', 'P3', 'P','n'),
]

# 遍历命令和参数，然后在新的命令提示符窗口中执行
for exe, arg1, arg2,arg3 in commands:
    # 如果 app.exe 路径中包含空格，确保使用引号括起来
    command = f'start cmd /k "{exe}" {arg1} {arg2} {arg3}'
    subprocess.Popen(command, shell=True)

time.sleep(6)


for i in range(3):
    # 动态构建带有当前循环i值的PowerShell命令
    ps_command = f"""
        $headers = @{{ "Content-Type" = "application/json" }}
        $body = '{{"clientID":"ahnhwi","operation":"SendMes1 - {i}","timestamp":{i}}}'
        $response = Invoke-WebRequest -Uri "http://localhost:1110/req" -Method POST -Headers $headers -Body $body
        """
    ps_command2 = f"""
        $headers = @{{ "Content-Type" = "application/json" }}
        $body = '{{"clientID":"ahnhwi","operation":"SendMes2 - {i}","timestamp":{i}}}'
        $response = Invoke-WebRequest -Uri "http://localhost:1129/req" -Method POST -Headers $headers -Body $body
        """
    ps_command3 = f"""
        $headers = @{{ "Content-Type" = "application/json" }}
        $body = '{{"clientID":"ahnhwi","operation":"SendMes3 - {i}","timestamp":{i}}}'
        $response = Invoke-WebRequest -Uri "http://localhost:1148/req" -Method POST -Headers $headers -Body $body
        """
    subprocess.Popen(['powershell', '-Command', ps_command])
    subprocess.Popen(['powershell', '-Command', ps_command2])
    subprocess.Popen(['powershell', '-Command', ps_command3])
    time.sleep(0.015)