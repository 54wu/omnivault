# -*- coding: utf-8 -*-
"""
字段归一化：读取转换为 Markdown/文本的材料目录，调用本地 Ollama (qwen3:8b)
将材料里的字段语义归一化为一份结构化 JSON。

用法:
    python normalize.py <材料目录> [输出json路径]

可选环境变量:
    LLM_BASE_URL  (默认 http://localhost:11434/v1)
    LLM_MODEL     (默认 qwen3:8b)

输出:
    <材料目录>/_normalized.json  (默认)
"""

import sys
import os
import json
import urllib.request
from pathlib import Path

DEFAULT_URL = os.environ.get("LLM_BASE_URL", "http://localhost:11434/v1")
DEFAULT_MODEL = os.environ.get("LLM_MODEL", "qwen3:8b")

# 常见字段映射提示(供模型参考, 可按需自行增删)
CANON_KEYS = [
    "name", "gender", "birth_date", "nationality", "id_number",
    "phone", "email", "address",
    "education", "school", "degree", "major", "graduation_date",
    "job_title", "company", "work_since", "salary",
    "marital_status", "emergency_contact", "emergency_phone",
    "bank_name", "bank_account",
]

SYSTEM_PROMPT = f"""你是个人信息提取助理。请从给定的材料文本中，把"表达同一含义的字段"统一成标准字段名，并提取其值。

标准字段名列表(取值为空或有歧义就省略该字段):
{CANON_KEYS}

同义字段示例:
- 联系电话 / 手机号 / 手机号码 / 电话      -> phone
- 出生日期 / 生日 / 出生年月日            -> birth_date
- 婚姻状况 / 是否已婚 / 婚否              -> marital_status
- 现居地址 / 住址 / 家庭住址             -> address

要求:
1. 只输出一个 JSON 对象, 不要输出任何其它文字、解释或代码块标记。
2. 键必须取自标准字段名列表; 若材料中出现标准列表中不存在的字段, 请用下划线风格自定义新键。
3. 值保留原文或做轻微规范化(如日期转成 YYYY-MM-DD)。"""


def call_llm(material_text: str) -> str:
    body = {
        "model": DEFAULT_MODEL,
        "messages": [
            {"role": "system", "content": SYSTEM_PROMPT},
            {"role": "user", "content": material_text},
        ],
        "stream": False,
        # 仅对 Ollama 原生端点有效：关闭 qwen3 思考，显著提速并避免超时
        "think": False,
        "options": {"num_predict": 2048},
    }
    # 优先用 Ollama 原生 /api/chat（支持 think:false 关闭思考，速度稳定）
    # 回退到 OpenAI 兼容 /v1/chat/completions（think 字段会被忽略但不报错）
    variants = [
        (DEFAULT_URL.rstrip("/").replace("/v1", "") + "/api/chat", body),
        (DEFAULT_URL.rstrip("/") + "/chat/completions", {
            "model": DEFAULT_MODEL,
            "messages": body["messages"],
            "stream": False,
        }),
    ]
    last_err = None
    for url, payload in variants:
        try:
            req = urllib.request.Request(
                url,
                data=json.dumps(payload).encode("utf-8"),
                headers={"Content-Type": "application/json"},
            )
            with urllib.request.urlopen(req, timeout=400) as resp:
                data = json.load(resp)
            if url.endswith("/api/chat"):
                return data["message"]["content"]
            return data["choices"][0]["message"]["content"]
        except Exception as e:
            last_err = e
            continue
    raise RuntimeError(f"调用 LLM 失败: {last_err}")


def _strip_think(content: str) -> str:
    """去掉 qwen3 的 思考 <reasoning> 包裹(若存在)与多余的代围块标记。"""
    import re
    content = re.sub(r"<reasoning>.*?</reasoning>", "", content, flags=re.S)
    content = re.sub(r"```json\s*", "", content)
    content = re.sub(r"```", "", content)
    return content.strip()


def parse_json(content: str) -> dict:
    content = _strip_think(content)
    start = content.find("{")
    end = content.rfind("}")
    if start == -1 or end == -1 or end < start:
        raise ValueError(f"未能在模型输出中解析出 JSON: {content[:300]}")
    return json.loads(content[start:end + 1])


def main():
    if len(sys.argv) < 2:
        print(__doc__)
        sys.exit(1)

    src_dir = Path(sys.argv[1]).resolve()
    out_json = Path(sys.argv[2]).resolve() if len(sys.argv) > 2 else src_dir / "_normalized.json"

    if not src_dir.is_dir():
        print(f"错误: 找不到目录 {src_dir}")
        sys.exit(1)

    # 汇总输入目录下所有 md/txt (该目录即 convert.py 的输出目录)
    texts = []
    for p in src_dir.rglob("*"):
        if not p.is_file() or p.suffix.lower() not in (".md", ".txt"):
            continue
        texts.append(f"===== 材料: {p.name} =====\n" + p.read_text(encoding="utf-8", errors="ignore"))
    if not texts:
        print("错误: 目录下没有 .md / .txt 材料")
        sys.exit(1)

    material = "\n\n".join(texts)
    print(f"已汇总 {len(texts)} 份材料, 总字数 {len(material)}, 调用 {DEFAULT_MODEL}...")

    content = call_llm(material)
    result = parse_json(content)

    out_json.parent.mkdir(parents=True, exist_ok=True)
    out_json.write_text(json.dumps(result, ensure_ascii=False, indent=2), encoding="utf-8")
    print(f"\n归一化完成: {out_json}")
    for k, v in result.items():
        print(f"  {k} = {v}")


if __name__ == "__main__":
    main()