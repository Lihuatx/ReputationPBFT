import os
import time
from os.path import exists

def monitor_file():
    file_name = 'costTime.txt'
    xls_file = 'data.xls'

    # 检查文件是否存在，如果存在则删除（处理上一次的测试数据）
    if exists(file_name):
        os.remove(file_name)
        print(f"Removed existing {file_name}")

    # 等待文件再次出现
    while not exists(file_name):
        print("Waiting for file to appear...")
        time.sleep(5)  # 每5秒检查一次

    # 读取 duration 并处理
    with open(file_name, 'r') as file:
        duration = file.read()
        # print(f"Duration found: {duration}")

        # 检查 data.xls 文件是否存在，不存在则创建
        if not exists(xls_file):
            with open(xls_file, 'w') as xls:
                xls.write('Duration\n')

        # 读取 data.xls 以确定已有数据的数量
        with open(xls_file, 'r') as xls:
            # 读取所有行并计算行数，减一是因为标题行也算一行
            count = len(xls.readlines()) - 1

        # 将 duration 追加到 data.xls 中
        with open(xls_file, 'a') as xls:
            xls.write(f"{duration}\n")

        print(f"Duration {duration} added to {xls_file} as entry number {count + 1}")

    return count

if __name__ == "__main__":
    monitor_file()
