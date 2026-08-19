# -*- coding: utf-8 -*-
"""
网页自动填表：读取归一化 JSON，用 Playwright 打开目标网页并按字段自动填写。

用法:
    python fill.py <normalized.json> --url <http://...> [--mapping map.txt] [--headless]

可选:
    --url  URL  (必填)
    --mapping  手动映射文本文件, 每行 "json键=页面label关键词"
    --headless 无头模式; 默认有头便于人工核对
"""

import sys
import json
import argparse
from pathlib import Path

# JSON 键 -> 页面 label 可能的同义词(用于语义匹配)
SYNONYMS = {
    "phone":          ["电话", "手机", "联系电话", "移动", "phone", "mobile", "tel"],
    "name":           ["姓名", "名字", "name", "用户名"],
    "email":          ["邮箱", "email", "邮件", "e-mail"],
    "gender":         ["性别", "gender", "sex"],
    "birth_date":     ["出生", "birth", "生日", "birthday"],
    "id_number":      ["身份证", "证件号", "证件", "id", "identity"],
    "address":        ["地址", "address", "住址", "家庭住址"],
    "education":      ["学历", "education", "学位"],
    "school":         ["学校", "school", "院校"],
    "major":          ["专业", "major"],
    "marital_status": ["婚姻", "marital", "已婚", "未婚"],
    "company":        ["公司", "company", "单位", "employer"],
    "job_title":      ["职位", "职务", "title", "岗位"],
    "nationality":    ["国籍", "nationality"],
    "emergency_contact": ["紧急联系人", "emergency"],
}


def load_mapping(path) -> dict:
    """读取 {json键: label关键词} 手动映射。"""
    m = {}
    if not path:
        return m
    for line in Path(path).read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if "=" in line:
            k, _, v = line.partition("=")
            m[k.strip()] = v.strip()
    return m


def fill_page(page, data, mapping):
    filled = []

    def do_fill(key, value, label_text=None, for_id=None):
        nonlocal filled
        if value in (None, ""):
            return
        candidates = []
        if for_id:
            candidates.append(page.locator(f"#{for_id}"))
        elif label_text:
            # label 内部包裹
            found = None
            for lbl in page.locator("label").all():
                try:
                    if lbl.inner_text().strip() == label_text.strip():
                        found = lbl
                        break
                except Exception:
                    continue
            if found:
                inner = found.locator("input, select, textarea")
                if inner.count():
                    candidates.append(inner.first)
        # 兜底: 手动映射的 name 匹配
        if key in mapping:
            token = mapping[key]
            nl = page.locator(f'input[name*="{token}"], select[name*="{token}"], textarea[name*="{token}"]')
            if nl.count():
                candidates.insert(0, nl.first)
        for loc in candidates:
            try:
                if loc.is_visible() and loc.count():
                    loc.fill(str(value))
                    filled.append(f"{key}={value}  (label: {label_text or for_id})")
                    return True
            except Exception:
                continue
        return False

    # 主流程: 遍历页面上所有 label
    try:
        labels = page.locator("label").all()
    except Exception:
        labels = []

    used_json = set()
    for lbl in labels:
        try:
            text = lbl.inner_text().strip()
        except Exception:
            continue
        if not text:
            continue
        for_id = lbl.get_attribute("for") or None
        # 找匹配的 JSON 键
        for key in data:
            if key in used_json:
                continue
            toks = SYNONYMS.get(key, [key])
            if any(t.lower() in text.lower() for t in toks):
                if do_fill(key, data[key], label_text=text, for_id=for_id):
                    used_json.add(key)
                break

    # 处理手动映射中未被 label 匹配到的
    for key, value in data.items():
        if key in used_json or key not in mapping:
            continue
        token = mapping[key]
        nl = page.locator(f'input[name*="{token}"], select[name*="{token}"], textarea[name*="{token}"]')
        if nl.count():
            try:
                nl.first.fill(str(value))
                filled.append(f"{key}={value}  (name映射: {token})")
            except Exception:
                pass
    return filled


def main():
    ap = argparse.ArgumentParser(description="按 JSON 自动填网页表单")
    ap.add_argument("json_path")
    ap.add_argument("--url", required=True)
    ap.add_argument("--mapping", default=None)
    ap.add_argument("--headless", action="store_true")
    args = ap.parse_args()

    data = json.loads(Path(args.json_path).read_text(encoding="utf-8"))
    mapping = load_mapping(args.mapping)

    from playwright.sync_api import sync_playwright
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=args.headless)
        page = browser.new_page(locale="zh-CN")
        page.goto(args.url)
        try:
            page.wait_for_load_state("networkidle", timeout=15000)
        except Exception:
            pass
        print(f"已打开: {args.url}\n")
        filled = fill_page(page, data, mapping)
        if filled:
            print("已填写字段:")
            for f in filled:
                print("  -", f)
        else:
            print("未匹配到任何字段。")
            print("提示: 可用 --mapping map.txt 提供映射(每行 json键=页面label关键词).")
        if not args.headless:
            input("\n请核对后按回车关闭...")
        browser.close()


if __name__ == "__main__":
    main()