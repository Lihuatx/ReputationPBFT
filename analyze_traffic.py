#!/usr/bin/python3
import subprocess
import re

def get_ports_from_nodetable():
    ports = set()
    with open('nodetable.txt', 'r') as file:
        for line in file:
            # 从每行提取端口号
            match = re.search(r':(\d+)$', line)
            if match:
                ports.add(match.group(1))
    return sorted(list(ports))  # 返回排序后的端口列表

def create_port_filter(ports):
    # 创建tcpdump的端口过滤表达式
    return ' or '.join([f'port {port}' for port in ports])

def analyze_traffic():
    ports = get_ports_from_nodetable()
    port_filter = create_port_filter(ports)

    # 开始抓包
    print("Starting packet capture...")
    capture_cmd = f"sudo tcpdump -i any '({port_filter})' -w ltdbft_capture.pcap"
    print(f"Running command: {capture_cmd}")
    print("Press Ctrl+C to stop capturing...")

    try:
        subprocess.run(capture_cmd, shell=True)
    except KeyboardInterrupt:
        print("\nCapture stopped by user")

    # 分析结果
    print("\nAnalyzing capture file...")

    # 1. 总体统计
    print("\n=== Overall Statistics ===")
    subprocess.run("tcpdump -r ltdbft_capture.pcap | wc -l", shell=True)
    subprocess.run("tcpdump -r ltdbft_capture.pcap -qnn | awk '{print $NF}' | awk '{sum += $1} END {print \"Total bytes transferred: \"sum}'", shell=True)

    # 2. 按端口统计
    print("\n=== Per Port Statistics ===")
    for port in ports:
        print(f"\nPort {port}:")
        # 统计特定端口的包数
        packet_count_cmd = f"tcpdump -r ltdbft_capture.pcap -nn 'port {port}' | wc -l"
        print("Packet count:")
        subprocess.run(packet_count_cmd, shell=True)
        # 统计特定端口的字节数
        bytes_cmd = f"tcpdump -r ltdbft_capture.pcap -nn 'port {port}' -q | awk '{{print $NF}}' | awk '{{sum += $1}} END {{print \"Bytes transferred: \"sum}}'"
        subprocess.run(bytes_cmd, shell=True)

if __name__ == "__main__":
    analyze_traffic()