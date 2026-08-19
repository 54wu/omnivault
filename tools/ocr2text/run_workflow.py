# -*- coding: utf-8 -*-
"""
OmniVault 材料 -> 填表 完整工作流(一条命令)。

流程:
  1. convert.py    材料文件夹 -> Markdown/文本
  2. normalize.py  汇总 md, 调本地 qwen3:8b -> 归一化 JSON
  3. fill.py       用 JSON + Playwright 打开网页自动填表

用法:
    python run_workflow.py <材料目录> --url <http://...> [--no-fill] [--txt] [--headless]

参数:
    <材料目录>   包含 Word/PDF/图片 的材料文件夹(必填)
    --url       目标网页表单 URL(填表阶段需要)
    --no-fill   只跑到第 2 步(场化出 JSON), 不自动填表
    --txt       转文本输出 txt(默认 md)
    --headless  fill 使用无头模式
    --mapping   手动映射文件(传给 fill)

环境变量(可覆盖):
    LLM_BASE_URL  默认 http://localhost:11434/v1
    LLM_MODEL     默认 qwen3:8b

示例:
    python run_workflow.py "D:/我的材料" --url "http://127.0.0.1:8000/form" --no-fill
"""

import sys
import json
import argparse
import subprocess
from pathlib import Path

HERE = Path(__file__).resolve().parent
PY = HERE / ".venv" / "Scripts" / "python.exe"
if not PY.exists():
    PY = sys.executable  # 回退到当前 python


def run(script: str, *args, stage: str):
    print(f"\n========== 阶段: {stage} ==========")
    cmd = [str(PY), str(HERE / script), *args]
    result = subprocess.run(cmd)
    if result.returncode != 0:
        print(f"[工作流中止] {stage} 阶段失败 (退出码 {result.returncode})")
        sys.exit(result.returncode)
    return result


def main():
    ap = argparse.ArgumentParser(description="OmniVault 材料 -> 填表 完整工作流")
    ap.add_argument("material_dir")
    ap.add_argument("--url", default=None)
    ap.add_argument("--no-fill", action="store_true")
    ap.add_argument("--txt", action="store_true")
    ap.add_argument("--headless", action="store_true")
    ap.add_argument("--mapping", default=None)
    args = ap.parse_args()

    material = Path(args.material_dir).resolve()
    if not material.is_dir():
        print(f"错误: 找不到材料目录 {material}")
        sys.exit(1)

    # 阶段 1: 转 Markdown
    convert_args = [str(material)]
    if args.txt:
        convert_args.append("--txt")
    run("convert.py", *convert_args, stage="1/3 转文本")

    # 阶段 2: 归一化 JSON (输入=第1阶段输出目录)
    convert_out = material / "_output"
    norm_json = material / "_normalized.json"
    if not convert_out.exists():
        print(f"错误: 找不到转换输出目录 {convert_out}, 请先运行阶段1")
        sys.exit(1)
    run("normalize.py", str(convert_out), str(norm_json), stage="2/3 字段归一化")

    print("\n归一化结果 JSON:")
    print(Path(norm_json).read_text(encoding="utf-8"))

    # 阶段 3: 填表(可选)
    if not args.no_fill:
        if not args.url:
            print("\n[工作流提示] 未提供 --url, 跳过填表阶段。输出 JSON 已保存在上面。")
        else:
            fill_args = [str(norm_json), "--url", args.url]
            if args.mapping:
                fill_args += ["--mapping", args.mapping]
            if args.headless:
                fill_args.append("--headless")
            run("fill.py", *fill_args, stage="3/3 网页填表")
    else:
        print("\n[跳过] 已按 --no-fill 跳过填表阶段。")

    print("\n========== 工作流完成 ==========")


if __name__ == "__main__":
    main()