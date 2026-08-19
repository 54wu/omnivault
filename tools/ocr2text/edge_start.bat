@echo off
REM ============================================================
REM  OmniVault Edge 接管启动器
REM
REM  用 调试端口(CDP) 启动一条独立的 Edge 实例，
REM  让  edge_fill.py 能接管它来填表。
REM  复用你的默认 Edge 用户数据，因此保留登录态。
REM ============================================================

REM ------- 配置：按需修改 -------
set "EDGE=C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe"
set "PORT=9222"
set "URL=https://www.bing.com"
REM ------- 配置结束 -------

if not exist "%EDGE%" (
  echo [错误] 找不到 Edge: %EDGE%
  echo 请编辑本文件, 把 EDGE 改成你的 Edge 路径。
  pause
  exit /b 1
)

echo 正在以调试端口 %PORT% 启动 Edge...
echo 启动后可自由打开 / 导航到目标网页, 然后运行  python edge_fill.py
echo.
"%EDGE%" --remote-debugging-port=%PORT% --user-data-dir="%USERPROFILE%\omnivault-edge-profile" --no-first-run --no-default-browser-check "%URL%"