@echo off
REM 打开终端
start cmd /k

REM 等待终端打开
timeout /t 1 >nul

REM 执行 python 命令
python start.py
