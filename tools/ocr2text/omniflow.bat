@echo off
chcp 65001 >nul
REM ============================================================
REM  OmniVault 一站式工作流 启动器（交互式）
REM  双击本文件即可打开 omniflow.py，然后按提示输入
REM  材料文件夹、服务令牌、vault 地址。
REM ============================================================
cd /d "%~dp0"

if exist ".venv\Scripts\python.exe" (
  ".venv\Scripts\python.exe" -X utf8 omniflow.py
) else (
  python -X utf8 omniflow.py
)

echo.
echo 程序已退出。
pause