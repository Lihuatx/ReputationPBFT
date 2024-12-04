#!/usr/bin/python3
import subprocess
import re
from collections import defaultdict
import time

def get_ports_from_nodetable():
    ports = {}
    with open('nodetable.txt', 'r') as file:
        for line in file:
            parts = line.strip().split()
            if len(parts) == 3:
                cluster, node_id, address = parts
                port = address.split(':')[1]
                # 使用完整的标识符作为键
                ports[port] = f"{cluster}-{node_id}"
    return ports

def create_port_filter(ports):
    return ' or '.join([f'port {port}' for port in ports])

def analyze_capture_file(ports):
    print("\nAnalyzing traffic for each port...")
    port_stats = defaultdict(lambda: {
        'sent_packets': 0, 'sent_bytes': 0,
        'received_packets': 0, 'received_bytes': 0
    })
    total_ports = len(ports)

    for i, port in enumerate(ports, 1):
        print(f"\rAnalyzing port {port} ({i}/{total_ports})", end='', flush=True)

        try:
            # 分析发送的数据包
            cmd = f"tcpdump -r ltdbft_capture.pcap -nn 'src port {port}' | wc -l"
            result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
            sent_packets = int(result.stdout.strip())

            # 分析发送的字节数
            cmd = f"tcpdump -r ltdbft_capture.pcap -nn 'src port {port}' -q | awk '{{print $NF}}' | awk '{{sum += $1}} END {{print sum}}'"
            result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
            sent_bytes = int(result.stdout.strip() or 0)

            # 分析接收的数据包
            cmd = f"tcpdump -r ltdbft_capture.pcap -nn 'dst port {port}' | wc -l"
            result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
            received_packets = int(result.stdout.strip())

            # 分析接收的字节数
            cmd = f"tcpdump -r ltdbft_capture.pcap -nn 'dst port {port}' -q | awk '{{print $NF}}' | awk '{{sum += $1}} END {{print sum}}'"
            result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
            received_bytes = int(result.stdout.strip() or 0)

            port_stats[port] = {
                'node': ports[port],
                'sent_packets': sent_packets,
                'sent_bytes': sent_bytes,
                'sent_kb': round(sent_bytes / 1024, 2),
                'received_packets': received_packets,
                'received_bytes': received_bytes,
                'received_kb': round(received_bytes / 1024, 2),
                'total_kb': round((sent_bytes + received_bytes) / 1024, 2)
            }
        except Exception as e:
            print(f"\nError analyzing port {port}: {e}")

    print("\nAnalysis complete!")
    return port_stats

def print_results(stats):
    print("\n======================= Traffic Analysis Results =======================")
    print("\nPer Node Traffic Statistics:")
    print(f"{'Node ID':<15} {'Port':<8} {'Sent(KB)':<10} {'Recv(KB)':<10} {'Total(KB)':<10}")
    print("-" * 70)

    total_sent = 0
    total_received = 0

    # 按照集群和节点ID排序
    def sort_key(x):
        node_id = x[1]['node']
        cluster, number = node_id.split('-')
        return (cluster, int(number[1:]))  # 例如 'N-N0' 会返回 ('N', 0)

    for port, data in sorted(stats.items(), key=sort_key):
        print(f"{data['node']:<15} {port:<8} {data['sent_kb']:<10.2f} {data['received_kb']:<10.2f} {data['total_kb']:<10.2f}")
        total_sent += data['sent_bytes']
        total_received += data['received_bytes']

    print("\nSummary:")
    print(f"Total Sent: {round(total_sent/1024/1024, 2):,} MB")
    print(f"Total Received: {round(total_received/1024/1024, 2):,} MB")
    print(f"Total Traffic: {round((total_sent + total_received)/1024/1024, 2):,} MB")

def main():
    print("Reading node table...")
    ports = get_ports_from_nodetable()
    port_filter = create_port_filter(ports)

    print("Starting packet capture...")
    print("Press Ctrl+C to stop capturing...")

    try:
        subprocess.run(f"sudo tcpdump -i any '({port_filter})' -w ltdbft_capture.pcap", shell=True)
    except KeyboardInterrupt:
        print("\nCapture stopped by user")
        time.sleep(1)

    print("\nStarting analysis...")
    stats = analyze_capture_file(ports)
    print_results(stats)

if __name__ == "__main__":
    main()