#!/bin/bash
# 使脚本在执行时显示命令
set -x

# 打开一个新的终端窗口并执行一个保持打开的shell（取决于你使用的终端模拟器，你可能需要更改这一行）
# 例如，对于gnome-terminal（如果你使用的是GNOME桌面）:
gnome-terminal -- bash -c "exec bash"

# 等待一段时间以确保新的终端窗口打开
sleep 1

# 执行 go build 命令
go build -o app

# 等待一段时间以确保编译完成
sleep 1

# 执行 python 命令
python start.py

# 关闭set -x，停止显示命令
set +x
