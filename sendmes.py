import subprocess
import sys
import time
import datetime
# 定义第五个终端要执行的PowerShell命令

for i in range(5):
    # 动态构建带有当前循环i值的PowerShell命令
    # 动态构建带有当前循环i值的PowerShell命令
    ps_command = f"""
        $headers = @{{ "Content-Type" = "application/json" }}
        $body = '{{"clientID":"ahnhwi - {i}","operation":"SendMes1 - {i}","timestamp":{i}}}'
        $response = Invoke-WebRequest -Uri "127.0.0.1:1111/req" -Method POST -Headers $headers -Body $body
        """
    ps_command2 = f"""
        $headers = @{{ "Content-Type" = "application/json" }}
        $body = '{{"clientID":"ahnhwi - {i}","operation":"SendMes2 - {i}","timestamp":{i}}}'
        $response = Invoke-WebRequest -Uri "127.0.0.1:1114/req" -Method POST -Headers $headers -Body $body
        """
    ps_command3 = f"""
        $headers = @{{ "Content-Type" = "application/json" }}
        $body = '{{"clientID":"ahnhwi - {i}","operation":"SendMes3 - {i}","timestamp":{i}}}'
        $response = Invoke-WebRequest -Uri "127.0.0.1:1118/req" -Method POST -Headers $headers -Body $body
        """
    ps_command4 = f"""
            $headers = @{{ "Content-Type" = "application/json" }}
            $body = '{{"clientID":"ahnhwi - {i}","operation":"SendMes4 - {i}","timestamp":{i}}}'
            $response = Invoke-WebRequest -Uri "127.0.0.1:1122/req" -Method POST -Headers $headers -Body $body
            """
    ps_command5 = f"""
            $headers = @{{ "Content-Type" = "application/json" }}
            $body = '{{"clientID":"ahnhwi - {i}","operation":"SendMes5 - {i}","timestamp":{i}}}'
            $response = Invoke-WebRequest -Uri "127.0.0.1:1126/req" -Method POST -Headers $headers -Body $body
            """
    subprocess.Popen(['powershell', '-Command', ps_command])
    # subprocess.Popen(['powershell', '-Command', ps_command2])
    # subprocess.Popen(['powershell', '-Command', ps_command3])
    # subprocess.Popen(['powershell', '-Command', ps_command4])
    # subprocess.Popen(['powershell', '-Command', ps_command5])
    #time.sleep(0.05)
