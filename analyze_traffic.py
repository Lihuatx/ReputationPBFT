#!/usr/bin/python3
import subprocess
import re
import json
from collections import defaultdict

def get_ports_from_nodetable():
    ports = {}  # 用于存储节点ID和端口的映射
    with open('nodetable.txt', 'r') as file:
        for line in file:
            parts = line.strip().split()
            if len(parts) == 3:
                cluster, node_id, address = parts
                port = address.split(':')[1]
                ports[port] = f"{cluster}-{node_id}"
    return ports

def create_port_filter(ports):
    return ' or '.join([f'port {port}' for port in ports])

def analyze_capture_file(ports):
    port_stats = defaultdict(lambda: {'packets': 0, 'bytes': 0})

    # 分析每个端口的流量
    for port in ports:
        # 获取数据包数量
        cmd = f"tcpdump -r ltdbft_capture.pcap -nn 'port {port}' | wc -l"
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
        packets = int(result.stdout.strip())

        # 获取字节数
        cmd = f"tcpdump -r ltdbft_capture.pcap -nn 'port {port}' -q | awk '{{print $NF}}' | awk '{{sum += $1}} END {{print sum}}'"
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
        bytes_count = int(result.stdout.strip() or 0)

        port_stats[port] = {
            'node': ports[port],
            'packets': packets,
            'bytes': bytes_count,
            'kb': round(bytes_count / 1024, 2)
        }

    return port_stats

def print_results(stats):
    print("\n============= Traffic Analysis Results =============")
    print("\nPer Node Traffic Statistics:")
    print(f"{'Node ID':<15} {'Port':<8} {'Packets':<10} {'Size (KB)':<10}")
    print("-" * 45)

    total_packets = 0
    total_bytes = 0

    for port, data in sorted(stats.items()):
        print(f"{data['node']:<15} {port:<8} {data['packets']:<10} {data['kb']:<10.2f}")
        total_packets += data['packets']
        total_bytes += data['bytes']

    print("\nSummary:")
    print(f"Total Packets: {total_packets}")
    print(f"Total Traffic: {round(total_bytes/1024/1024, 2)} MB")

def main():
    ports = get_ports_from_nodetable()
    port_filter = create_port_filter(ports)

    # 开始抓包
    print("Starting packet capture...")
    print("Press Ctrl+C to stop capturing...")

    try:
        subprocess.run(f"sudo tcpdump -i any '({port_filter})' -w ltdbft_capture.pcap", shell=True)
    except KeyboardInterrupt:
        print("\nCapture stopped by user")

    # 分析结果
    print("\nAnalyzing capture file...")
    stats = analyze_capture_file(ports)
    print_results(stats)

if __name__ == "__main__":
    main()