import psutil
import time
import csv
import os
import subprocess
import json

def get_tmux_pids():
    """获取所有tmux窗口中运行的app进程的PID"""
    pids = []
    try:
        # 获取所有进程
        for proc in psutil.process_iter(['pid', 'name', 'cmdline']):
            try:
                if proc.info['name'] == 'app':
                    pids.append(proc.info['pid'])
            except (psutil.NoSuchProcess, psutil.AccessDenied):
                pass
    except Exception as e:
        print(f"Error getting PIDs: {e}")
    return pids

def monitor_process(pid, output_dir, sampling_interval=1):
    """监控单个进程的资源使用情况"""
    try:
        process = psutil.Process(pid)
        cmdline = process.cmdline()
        node_id = cmdline[1]  # 获取节点ID(如N0、M1等)

        output_file = os.path.join(output_dir, f'{node_id}_performance.csv')

        with open(output_file, 'w', newline='') as f:
            writer = csv.writer(f)
            writer.writerow(['Timestamp', 'CPU_%', 'Memory_MB', 'Threads'])

            while True:
                try:
                    cpu_percent = process.cpu_percent()
                    memory_mb = process.memory_info().rss / (1024 * 1024)  # 转换为MB
                    num_threads = process.num_threads()

                    writer.writerow([
                        time.strftime('%Y-%m-%d %H:%M:%S'),
                        cpu_percent,
                        round(memory_mb, 2),
                        num_threads
                    ])

                    time.sleep(sampling_interval)

                except psutil.NoSuchProcess:
                    print(f"Process {pid} ({node_id}) terminated")
                    break

    except Exception as e:
        print(f"Error monitoring process {pid}: {e}")

def main():
    # 创建输出目录
    timestamp = time.strftime('%Y%m%d_%H%M%S')
    output_dir = f'performance_data_{timestamp}'
    os.makedirs(output_dir, exist_ok=True)

    # 等待所有节点启动
    print("Waiting for nodes to start...")
    time.sleep(5)

    # 获取所有节点的PID
    pids = get_tmux_pids()
    print(f"Found {len(pids)} nodes running")

    # 为每个进程创建监控线程
    from threading import Thread
    threads = []
    for pid in pids:
        thread = Thread(target=monitor_process, args=(pid, output_dir))
        thread.daemon = True
        thread.start()
        threads.append(thread)

    # 等待所有监控线程结束
    try:
        while True:
            time.sleep(1)
    except KeyboardInterrupt:
        print("\nStopping performance monitoring...")

if __name__ == "__main__":
    main()