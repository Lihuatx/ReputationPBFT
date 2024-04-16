import socket

def start_server():
    server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    server_socket.bind(('0.0.0.0', 2000))
    server_socket.listen(1)
    print("服务端启动，监听端口2000。")

    while True:
        client_socket, addr = server_socket.accept()
        print(f"从 {addr} 接收到连接")
        while True:
            data = client_socket.recv(1024)
            if not data:
                break
            print(f"收到消息：{data.decode()}")
        client_socket.close()
        print("连接已关闭")

if __name__ == "__main__":
    start_server()
