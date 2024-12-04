import psutil
import time
import csv
import os
from datetime import datetime

def get_app_processes():
    """获取所有运行的app进程信息"""
    app_processes = {}
    for proc in psutil.process_iter(['pid', 'name', 'cmdline']):
        try:
            if proc.info['name'] == 'app' and len(proc.info['cmdline']) > 1:
                node_id = proc.info['cmdline'][1]
                app_processes[node_id] = proc.pid
        except (psutil.NoSuchProcess, psutil.AccessDenied):
            continue
    return app_processes

def monitor_performance(interval_seconds=1):
    """
    监控所有节点的性能指标并实时显示
    interval_seconds: 采样间隔(秒)
    """
    # 创建输出目录
    timestamp = datetime.now().strftime('%Y%m%d_%H%M%S')
    output_dir = f'performance_data_{timestamp}'
    os.makedirs(output_dir, exist_ok=True)

    # 获取所有app进程
    processes = get_app_processes()
    if not processes:
        print("No app processes found!")
        return

    print(f"Found {len(processes)} nodes running")
    print(f"Performance data will be saved to {output_dir}/")
    print("\nMonitoring node performance (Press Ctrl+C to stop)...\n")

    # 为每个节点创建CSV文件
    csv_files = {}
    csv_writers = {}
    for node_id, pid in processes.items():
        filename = os.path.join(output_dir, f'{node_id}_performance.csv')
        csv_files[node_id] = open(filename, 'w', newline='')
        csv_writers[node_id] = csv.writer(csv_files[node_id])
        csv_writers[node_id].writerow(['Timestamp', 'CPU_%', 'Memory_MB', 'Threads'])

    try:
        while True:
            current_time = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
            print(f"\nTimestamp: {current_time}")
            print("-" * 60)
            print(f"{'Node ID':^10} | {'CPU %':^8} | {'Memory (MB)':^12} | {'Threads':^8}")
            print("-" * 60)

            for node_id, pid in list(processes.items()):
                try:
                    proc = psutil.Process(pid)
                    # 等待一个间隔以获取准确的CPU使用率
                    cpu_percent = proc.cpu_percent()
                    memory_mb = proc.memory_info().rss / (1024 * 1024)
                    num_threads = proc.num_threads()

                    # 打印到终端
                    print(f"{node_id:^10} | {cpu_percent:^8.1f} | {memory_mb:^12.2f} | {num_threads:^8}")

                    # 写入CSV
                    csv_writers[node_id].writerow([
                        current_time,
                        cpu_percent,
                        round(memory_mb, 2),
                        num_threads
                    ])
                    # 确保数据立即写入文件
                    csv_files[node_id].flush()

                except psutil.NoSuchProcess:
                    print(f"Process for node {node_id} terminated")
                    processes.pop(node_id)
                    if not processes:
                        print("All processes terminated")
                        return

            time.sleep(interval_seconds)

    except KeyboardInterrupt:
        print("\n\nMonitoring stopped by user")
    finally:
        # 关闭所有CSV文件
        for f in csv_files.values():
            f.close()
        print(f"\nPerformance data saved to {output_dir}/")

if __name__ == "__main__":
    print("Starting performance monitoring...")
    monitor_performance()