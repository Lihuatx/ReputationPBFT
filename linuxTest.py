import subprocess
import time
import datetime

cnt = 0
for i in range(40):
    cnt += 1
    curl_command = f"""
        curl -X POST "http://127.0.0.01:1110/req" -H "Content-Type: application/json" -d '{{"clientID":"ahnhwi - {i}","operation":"SendMes1 - {i}","timestamp":{i}}}'
        """
    curl_command2 = f"""
        curl -X POST "http://127.0.0.1:1135/req" -H "Content-Type: application/json" -d '{{"clientID":"ahnhwi - {i}","operation":"SendMes2 - {i}","timestamp":{i}}}'
        """
    curl_command3 = f"""
        curl -X POST "http://127.0.0.1:1160/req" -H "Content-Type: application/json" -d '{{"clientID":"ahnhwi - {i}","operation":"SendMes3 - {i}","timestamp":{i}}}'
        """
    subprocess.Popen(['bash', '-c', curl_command3])
    subprocess.Popen(['bash', '-c', curl_command2])
    #subprocess.Popen(['bash', '-c', curl_command])

    if cnt % 20 == 0:
        # time.sleep(1)  # 如果需要，取消注释此行以使用 sleep
        pass
