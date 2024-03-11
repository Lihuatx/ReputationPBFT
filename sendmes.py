import subprocess
# 定义第五个终端要执行的PowerShell命令
ps_command = """
$headers = @{ "Content-Type" = "application/json" }
$body = '{"clientID":"ahnhwi","operation":"GetMyName","timestamp":859381532}'
$response = Invoke-WebRequest -Uri "http://localhost:1111/req" -Method POST -Headers $headers -Body $body
"""

# 在新的PowerShell窗口中执行第五个命令
subprocess.Popen(['powershell', '-Command', ps_command])