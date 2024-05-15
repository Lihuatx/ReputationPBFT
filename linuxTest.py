import subprocess
import sys
import time
import socket
import threading

arg = sys.argv[1]

base_server_ips = ["43.132.126.36", "43.131.248.12", "43.128.253.129", "150.109.6.41", "43.133.117.50"]


def close_socket(client_socket):
    client_socket.close()
    print("连接已关闭")

def extract_port():
    # 初始化端口变量
    PortN, PortM, PortP, PortJ, PortK = None, None, None, None, None

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
                elif node_id == "J0":
                    PortM = port
                elif node_id == "K0":
                    PortP = port

    # 返回端口信息，以便在其他地方使用
    return PortN, PortM, PortP, PortJ, PortK

PortN, PortM, PortP, PortJ, PortK = extract_port()

if arg == "N":
    subprocess.run(['tmux', 'kill-session', '-t', 'myClient'])
    subprocess.run(['tmux', 'new-session', '-d', '-s', 'myClient'])

    command = f"./app client N"
    subprocess.run(['tmux', 'new-window', '-t', f'myClient:{1}', '-n', "Client-1"])

    client_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    client_socket.connect((base_server_ips[1], 2000))
    message = "link"
    client_socket.sendall(message.encode())
    # client_socket_2 = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    # client_socket_2.connect((base_server_ips[2], 2000))
    # message = "link"
    # client_socket_2.sendall(message.encode())
    # client_socket_3 = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    # client_socket_3.connect((base_server_ips[3], 2000))
    # message = "link"
    # client_socket_3.sendall(message.encode())
    # client_socket_4 = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    # client_socket_4.connect((base_server_ips[4], 2000))
    # message = "link"
    # client_socket_4.sendall(message.encode())

    tmux_command = f"tmux send-keys -t myClient:{1} './app client N' C-m"

    subprocess.Popen(['bash', '-c', tmux_command])

else:
    subprocess.run(['tmux', 'kill-session', '-t', 'myClient'])
    subprocess.run(['tmux', 'new-session', '-d', '-s', 'myClient'])

    command = f"./app client N"
    subprocess.run(['tmux', 'new-window', '-t', f'myClient:{1}', '-n', "Client-1"])

    server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    server_socket.bind(('0.0.0.0', 2000))
    server_socket.listen(1)
    print("服务端启动，监听端口2000。")


    client_socket, addr = server_socket.accept()
    print(f"从 {addr} 接收到连接")

    close_thread = threading.Thread(target=close_socket, args=(client_socket,))
    close_thread.start()
    tmux_command = f"tmux send-keys -t myClient:{1} './app client {arg}' C-m"

    subprocess.Popen(['bash', '-c', tmux_command])

