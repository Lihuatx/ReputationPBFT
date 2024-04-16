import socket

def send_message():
    client_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    client_socket.connect(('114.55.130.178', 2000))
    message = "link"
    client_socket.sendall(message.encode())
    print("消息已发送")
    client_socket.close()

if __name__ == "__main__":
    send_message()
