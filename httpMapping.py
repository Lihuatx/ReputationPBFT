import http.server
from urllib.parse import urlparse
import requests

class Proxy(http.server.SimpleHTTPRequestHandler):
    def do_POST(self):
        # 将请求转发到不同的本地端口
        port_mapping = {
            "N0": "1110",
            "N1": "1111",
            # 根据需要添加更多映射
        }

        # 解析请求路径
        parsed_path = urlparse(self.path)
        path_parts = parsed_path.path.strip("/").split("/")  # ['N0', 'req']

        if len(path_parts) < 2:
            self.send_error(400, "Bad Request")
            return

        node_id, req_path = path_parts[0], "/".join(path_parts[1:])  # 'N0', 'req'
        target_port = port_mapping.get(node_id, None)

        if target_port is None:
            self.send_error(404, "Node not found")
            return

        content_length = int(self.headers['Content-Length'])
        post_data = self.rfile.read(content_length)

        # 构造目标 URL
        target_url = f"http://localhost:{target_port}/{req_path}"

        # 转发请求
        response = requests.post(target_url, data=post_data, headers={"Content-Type": "application/json"})

        # 将响应返回给原始请求者
        self.send_response(response.status_code)
        for header, value in response.headers.items():
            if header not in ['Content-Length', 'Transfer-Encoding', 'Content-Encoding']:
                self.send_header(header, value)
        self.end_headers()
        self.wfile.write(response.content)

def run(server_class=http.server.HTTPServer, handler_class=Proxy, port=80):
    server_address = ('', port)
    httpd = server_class(server_address, handler_class)
    print(f"Starting httpd on port {port}...")
    httpd.serve_forever()

if __name__ == "__main__":
    run()
