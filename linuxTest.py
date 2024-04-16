import subprocess
import sys
import time
import socket
import threading

arg = sys.argv[1]

def close_socket(client_socket):
    client_socket.close()
    print("连接已关闭")

def extract_port():
    # 初始化端口变量
    PortN, PortM, PortP = None, None, None

    # 打开并读取文件
    with open('nodetable.txt', 'r') as file:
        for line in file:
            # 移除行尾的换行符并分割
            parts = line.strip().split()
            if len(parts) == 3:
                node_type, node_id, address = parts
                ip, port = address.split(':')

                # 根据节点ID分配端口号到对应变量
                if node_id == "N0":
                    PortN = port
                elif node_id == "M0":
                    PortM = port
                elif node_id == "P0":
                    PortP = port

    # 打印获取到的端口
    print(f"PortN: {PortN}")
    print(f"PortM: {PortM}")
    print(f"PortP: {PortP}")

    # 返回端口信息，以便在其他地方使用
    return PortN, PortM, PortP

PortN, PortM, PortP = extract_port()

if arg == "N":
    client_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    client_socket.connect(('49.51.228.140', 2000))
    message = "link"
    client_socket.sendall(message.encode())
    client_socket_2 = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    client_socket_2.connect(('43.132.214.22', 2000))
    message = "link"
    client_socket_2.sendall(message.encode())
    # while True:
    #     try:
    #         response = client_socket_2.sendall(message.encode())
    #         if response == b'':
    #             break
    #     except socket.error as e:
    #         break

    for i in range(1000):
        curl_command = f"""
            curl -X POST "http://127.0.0.01:{PortN}/req" -H "Content-Type: application/json" -d '{{"clientID":"ahnhwi - {i}","operation":"SendMes1 - {i}","timestamp":{i}}}'
            """
        subprocess.Popen(['bash', '-c', curl_command])
else:
    server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    server_socket.bind(('0.0.0.0', 2000))
    server_socket.listen(1)
    print("服务端启动，监听端口2000。")


    client_socket, addr = server_socket.accept()
    print(f"从 {addr} 接收到连接")

    close_thread = threading.Thread(target=close_socket, args=(client_socket,))
    close_thread.start()
    if arg == "M":
        for i in range(1000):
            curl_command = f"""
                curl -X POST "http://127.0.0.01:{PortM}/req" -H "Content-Type: application/json" -d '{{"clientID":"ahnhwi - {i}","operation":"SendMes1 - {i}","timestamp":{i}}}'
                """
            subprocess.Popen(['bash', '-c', curl_command])
    else:
        for i in range(1000):
            curl_command = f"""
                curl -X POST "http://127.0.0.01:{PortP}/req" -H "Content-Type: application/json" -d '{{"clientID":"ahnhwi - {i}","operation":"SendMes1 - {i}","timestamp":{i}}}'
                """
            subprocess.Popen(['bash', '-c', curl_command])

# cnt = 0
# for i in range(40):
#     cnt += 1
#     curl_command = f"""
#         curl -X POST "http://127.0.0.01:1110/req" -H "Content-Type: application/json" -d '{{"clientID":"ahnhwi - {i}","operation":"SendMes1 - {i}","timestamp":{i}}}'
#         """
#     curl_command2 = f"""
#         curl -X POST "http://43.132.214.22:1114/req" -H "Content-Type: application/json" -d '{{"clientID":"ahnhwi - {i}","operation":"SendMes2 - {i}","timestamp":{i}}}'
#         """
#     curl_command3 = f"""
#         curl -X POST "http://49.51.228.140:1118/req" -H "Content-Type: application/json" -d '{{"clientID":"ahnhwi - {i}","operation":"SendMes3 - {i}","timestamp":{i}}}'
#         """
#     subprocess.Popen(['bash', '-c', curl_command3])
#     subprocess.Popen(['bash', '-c', curl_command2])
#     subprocess.Popen(['bash', '-c', curl_command])
#
#     if cnt % 20 == 0:
#         # time.sleep(1)  # 如果需要，取消注释此行以使用 sleep
#         pass
