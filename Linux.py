import subprocess
import sys
import time
import datetime

# 定义要执行的curl命令
curl_command = """
curl -X POST -H "Content-Type: application/json" -d '{"clientID":"ahnhwi","operation":"SendMes1","timestamp":859381532}' http://localhost:1111/req
"""

curl_command2 = """
curl -X POST -H "Content-Type: application/json" -d '{"clientID":"ahnhwi","operation":"SendMes2","timestamp":859381532}' http://localhost:1116/req
"""

curl_command3 = """
curl -X POST -H "Content-Type: application/json" -d '{"clientID":"ahnhwi","operation":"SendMes3","timestamp":859381532}' http://localhost:1121/req
"""

num = int(sys.argv[1])
if num < 4:
    if num == 1:
        subprocess.Popen(['bash', '-c', curl_command])
    elif num == 2:
        subprocess.Popen(['bash', '-c', curl_command2])
    else:
        subprocess.Popen(['bash', '-c', curl_command3])
else:
    print(datetime.datetime.now())
    for i in range(10):
        # 动态构建带有当前循环i值的curl命令
        dynamic_curl_command = f"""
        curl -X POST -H "Content-Type: application/json" -d '{{"clientID":"ahnhwi","operation":"SendMes1 - {i}","timestamp":{i}}}' http://localhost:1111/req
        """
        dynamic_curl_command2 = f"""
        curl -X POST -H "Content-Type: application/json" -d '{{"clientID":"ahnhwi","operation":"SendMes2 - {i}","timestamp":{i}}}' http://localhost:1116/req
        """
        dynamic_curl_command3 = f"""
        curl -X POST -H "Content-Type: application/json" -d '{{"clientID":"ahnhwi","operation":"SendMes3","timestamp":{i}}}' http://localhost:1121/req
        """
        subprocess.Popen(['bash', '-c', dynamic_curl_command])
        subprocess.Popen(['bash', '-c', dynamic_curl_command2])
        # subprocess.Popen(['bash', '-c', dynamic_curl_command3])
        time.sleep(0.1)
