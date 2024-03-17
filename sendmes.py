import subprocess
import sys
import time
import datetime
# 定义第五个终端要执行的PowerShell命令

ps_command = """
$headers = @{ "Content-Type" = "application/json" }
$body = '{"clientID":"ahnhwi","operation":"SendMes1","timestamp":859381532}'
$response = Invoke-WebRequest -Uri "http://localhost:1111/req" -Method POST -Headers $headers -Body $body
"""

ps_command2 = """
$headers = @{ "Content-Type" = "application/json" }
$body = '{"clientID":"ahnhwi","operation":"SendMes2","timestamp":859381532}'
$response = Invoke-WebRequest -Uri "http://localhost:1116/req" -Method POST -Headers $headers -Body $body
"""

ps_command3 = """
$headers = @{ "Content-Type" = "application/json" }
$body = '{"clientID":"ahnhwi","operation":"SendMes3","timestamp":859381532}'
$response = Invoke-WebRequest -Uri "http://localhost:1121/req" -Method POST -Headers $headers -Body $body
"""

num = int(sys.argv[1])
if num < 4:
    if num == 1:
        subprocess.Popen(['powershell', '-Command', ps_command])
    elif num == 2:
        subprocess.Popen(['powershell', '-Command', ps_command2])
    else:
        subprocess.Popen(['powershell', '-Command', ps_command3])
else:
    print(datetime.datetime.now())
    for i in range(10):
        # 动态构建带有当前循环i值的PowerShell命令
        ps_command = f"""
        $headers = @{{ "Content-Type" = "application/json" }}
        $body = '{{"clientID":"ahnhwi","operation":"SendMes1 - {i}","timestamp":{i}}}'
        $response = Invoke-WebRequest -Uri "http://localhost:1111/req" -Method POST -Headers $headers -Body $body
        """
        ps_command2 = f"""
        $headers = @{{ "Content-Type" = "application/json" }}
        $body = '{{"clientID":"ahnhwi","operation":"SendMes2 - {i}","timestamp":{i}}}'
        $response = Invoke-WebRequest -Uri "http://localhost:1116/req" -Method POST -Headers $headers -Body $body
        """
        ps_command3 = f"""
        $headers = @{{ "Content-Type" = "application/json" }}
        $body = '{{"clientID":"ahnhwi","operation":"SendMes3","timestamp":{i}}}'
        $response = Invoke-WebRequest -Uri "http://localhost:1121/req" -Method POST -Headers $headers -Body $body
        """
        subprocess.Popen(['powershell', '-Command', ps_command])
        subprocess.Popen(['powershell', '-Command', ps_command2])
        # subprocess.Popen(['powershell', '-Command', ps_command3])
        # time.sleep(0.1)

# 在新的PowerShell窗口中执行第五个命令
# subprocess.Popen(['powershell', '-Command', ps_command])