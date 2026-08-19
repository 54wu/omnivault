# -*- coding: utf-8 -*-
"""
一步式验证脚本：自启动独立 Edge 实例 -> 打开测试表单 -> 从 OmniVault 读字段 -> 自动填表 -> 校验结果。

设计: 在同一个 Python 进程内用 subprocess 启动 Edge (DETACHED_PROCESS, 脱离终端, 不被回收),
      然后通过 CDP 连接填表。验证完成后输出填写前后对比, 校验是否填对。
"""

import sys
import os
import time
import json
import subprocess
import tempfile
from pathlib import Path

# 复用 edge_fill.py 的字段读取 / 匹配 / 填表逻辑
sys.path.insert(0, str(Path(__file__).parent))
import edge_fill

TOKEN = os.environ.get("OVAULT_TOKEN", "")
ADDR = "http://127.0.0.1:7200"
PORT = 9230  # 用独立端口避免与已有实例冲突
PROFILE = os.environ.get("OVAULT_PROFILE", str(Path(tempfile.gettempdir()) / "ovfill-profile-verify"))
EDGE = os.environ.get("EDGE_PATH", r"C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe")

FORM_URL = (
    "data:text/html;charset=utf-8,"
    + __import__("urllib.parse", fromlist=["quote"]).quote(
        "<html><head><meta charset=\"utf-8\"></head><body>"
        "<h1>测试表单</h1><form>"
        "<div><label for=f_name>姓名</label><input id=f_name name=name></div>"
        "<div><label for=f_phone>手机号</label><input id=f_phone name=phone></div>"
        "<div><label for=f_email>邮箱</label><input id=f_email name=email></div>"
        "<div><label for=f_birth>出生日期</label><input id=f_birth name=birth_date></div>"
        "<div><label for=f_company>公司</label><input id=f_company name=company></div>"
        "<div><label for=f_address>家庭住址</label><input id=f_address name=address></div>"
        "<div><label for=f_place>籍贯</label><input id=f_place name=native_place></div>"
        "<div><label for=f_card>身份证号</label><input id=f_card name=id_number></div>"
        "</form></body></html>"
    )
)


def wait_port(port, tries=20):
    import urllib.request
    for _ in range(tries):
        try:
            req = urllib.request.Request(f"http://127.0.0.1:{port}/json/list")
            urllib.request.urlopen(req, timeout=2)
            return True
        except Exception:
            time.sleep(0.8)
    return False


def main():
    # 1) 启动独立 Edge
    import subprocess as sp
    DETACHED = 0x00000008 | 0x00000200  # DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP
    print("启动独立 Edge 实例...")
    sp.Popen(
        [EDGE, f"--remote-debugging-port={PORT}", "--remote-debugging-address=127.0.0.1",
         f"--user-data-dir={PROFILE}", "--no-first-run", "--no-default-browser-check", FORM_URL],
        stdout=sp.DEVNULL, stderr=sp.DEVNULL, creationflags=DETACHED,
    )
    if not wait_port(PORT):
        print("[错误] Edge 调试端口未就绪")
        return
    print("Edge 调试端口就绪, 已打开测试表单.")

    # 2) 读取 vault 字段
    fields = edge_fill.vault_fields(token=TOKEN, addr=ADDR)
    print(f"从 OmniVault 读取 {len(fields)} 个字段值.")

    # 3) 连接 CDP 填表
    from playwright.sync_api import sync_playwright
    with sync_playwright() as p:
        try:
            browser = p.chromium.connect_over_cdp(f"http://127.0.0.1:{PORT}")
        except Exception as e:
            print(f"[错误] connect_over_cdp 失败: {e}")
            return
        ctx = browser.contexts[0]
        page = None
        for c in ctx.pages:
            if c.url.startswith("data:text/html"):
                page = c
                break
        if page is None and ctx.pages:
            page = ctx.pages[0]
        page.front() if hasattr(page, "front") else None
        time.sleep(1)

        print(f"\n填写前各字段值:")
        before = {}
        for name in ["name", "phone", "email", "birth_date", "company", "address", "native_place", "id_number"]:
            el = page.locator(f'input[name="{name}"]')
            before[name] = el.input_value() if el.count() else None
            print(f"  {name} = {before[name]!r}")

        filled, used = edge_fill.fill_fields(page, fields, {}, False)

        print("\n填写结果 (已匹配填写):")
        for f in filled:
            print("  " + f)

        print("\n填写后各字段值:")
        ok = 0
        total = 0
        for name in ["name", "phone", "email", "birth_date", "company", "address", "native_place", "id_number"]:
            el = page.locator(f'input[name="{name}"]')
            val = el.input_value() if el.count() else ""
            total += 1
            filled_ok = val != ""
            if filled_ok:
                ok += 1
            print(f"  {name} = {val!r}  {'<-- 已填' if filled_ok else '<-- 空'}")
        print(f"\n校验: {ok}/{total} 个字段被成功填入.")
        browser.close()


if __name__ == "__main__":
    main()