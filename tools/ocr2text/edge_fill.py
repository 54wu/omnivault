# -*- coding: utf-8 -*-
"""
接管真实 Edge 填表：从 OmniVault 读取已解密字段，接管 调试端口 启动的 Edge 自动填表。

前置:
    1. 先运行  edge_start.bat  启动带调试端口的 Edge (默认端口 9222), 打开目标网页。
    2. 确保 vault 已解锁:  omnivault unlock

用法:
    python edge_fill.py [--debug-port 9222] [--omnivault <exe路径>] [--mapping map.txt] [--fill-unknown]

说明:
    - 默认连接 http://127.0.0.1:9222 下的 Edge, 复用其当前活动标签(用户打开的网页)。
    - 数据来自 `omnivault export` 的解密字段; 用字段名(field_name/id)与页面 label 做语义匹配。
    - 匹配不上且未开 --fill-unknown 时, 会列出未填字段供人工处理。
"""

import sys
import os
import json
import argparse
import subprocess
from pathlib import Path

VAULT_EXE = os.environ.get("OMNIVAULT_EXE", "omnivault")
DEFAULT_PORT = 9222
CDP = f"http://127.0.0.1:{DEFAULT_PORT}"

# 字段名 -> 网页 label 同义词(主要靠 vault 的 field_name 关联; 这里是补充多语别名)
SYNONYMS = {
    "name":       ["姓名", "名字", "name", "用户名", "full name"],
    "gender":     ["性别", "gender", "sex"],
    "phone":      ["电话", "手机", "联系电话", "移动", "mobile", "tel", "phone"],
    "email":      ["邮箱", "email", "邮件", "e-mail"],
    "birth":      ["出生", "birth", "生日", "birthday", "date of birth"],
    "address":    ["地址", "address", "住址", "家庭住址", "street", "city"],
    "marital":    ["婚姻", "marital", "已婚", "未婚"],
    "company":    ["公司", "company", "单位", "employer"],
    "job":        ["职位", "职务", "title", "岗位"],
    "education":  ["学历", "education", "学位"],
    "school":     ["学校", "school", "院校"],
    "major":      ["专业", "major"],
}

# vault 英文字段末段关键词 -> 网页中文 label 同义词(用于英字段匹配中文 label)
FIELD_KEYWORDS = {
    "name":           ["姓名", "名字", "名称"],
    "full_name":      ["姓名", "全名", "名字"],
    "phone":          ["电话", "手机", "联系电话", "手机号", "联系方式"],
    "alt_phone":      ["备用电话", "其他电话", "备用手机", "第二电话"],
    "email":          ["邮箱", "电子邮件", "邮件"],
    "email_backup":   ["备用邮箱", "其他邮箱", "备用邮件"],
    "date_of_birth":  ["出生日期", "生日", "出生"],
    "birth_date":     ["出生日期", "生日", "出生"],
    "birth_place":    ["出生地", "出生地点"],
    "gender":         ["性别"],
    "nationality":    ["国籍", "民族国"],
    "ethnicity":      ["民族"],
    "id_number":      ["身份证号", "身份证", "证件号", "证件号码"],
    "id_expiry":      ["身份证有效期", "证件有效期", "有效期"],
    "marital_status": ["婚姻状况", "婚姻", "已婚", "未婚"],
    "political_status": ["政治面貌", "政治"],
    "height":         ["身高"],
    "weight":         ["体重"],
    "native_place":   ["籍贯", "原籍"],
    "hukou_location": ["户口", "户籍", "户口所在地"],
    "household_location": ["家庭住址", "家庭地址"],
    "address":        ["住址", "地址", "居住地址", "家庭住址", "地址"],
    "home_address":   ["家庭住址", "现住址", "家庭地址"],
    "home_city":      ["城市", "市", "所在城市"],
    "home_district":  ["区", "区县", "所在区"],
    "street":         ["街道", "路"],
    "school":         ["学校", "院校", "就读学校"],
    "high_school":    ["高中", "中学"],
    "degree":         ["学位", "学历层次"],
    "degree_type":    ["学历", "学位类型", "学历类型"],
    "graduation_date": ["毕业时间", "毕业日期"],
    "employer":       ["单位", "工作单位", "公司", "雇主"],
    "company":        ["公司", "单位", "企业"],
    "title":          ["职位", "职务", "岗位", "职称"],
    "job_title":      ["职位", "职务", "岗位"],
    "internship_position": ["实习岗位", "实习职位", "实习职务"],
    "internship_org": ["实习单位", "实习公司", "实习机构"],
    "position":       ["职位", "职务", "岗位"],
    "emergency_contact": ["紧急联系人", "紧急联络人"],
    "contact_address": ["联系地址", "通讯地址", "联络地址"],
    "mother_name":    ["母亲姓名", "母亲名字"],
    "mother_org":     ["母亲单位", "母亲工作单位"],
    "father_name":    ["父亲姓名", "父亲名字"],
    "hobby":          ["爱好", "兴趣爱好"],
}


def load_mapping(path):
    m = {}
    if not path:
        return m
    for line in Path(path).read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if "=" in line:
            k, _, v = line.partition("=")
            m[k.strip()] = v.strip()
    return m


def vault_fields(exe=VAULT_EXE, token=None, addr="http://127.0.0.1:7200"):
    """读取 vault 全部字段 -> {字段id: 值}。

    优先用 HTTP + token (需已开启本地服务), 拿不到再回退本地 CLI export。
    """
    data = None
    if token:
        try:
            import urllib.request
            req = urllib.request.Request(
                addr.rstrip("/") + "/vault/context",
                headers={"Authorization": f"Bearer {token}"},
            )
            with urllib.request.urlopen(req, timeout=10) as resp:
                data = json.load(resp)
        except Exception as e:
            print(f"[提示] HTTP 读取 vault 失败 ({e}), 回退 CLI export...")
    if data is None:
        try:
            out = subprocess.run([exe, "export"], capture_output=True, text=True, encoding="utf-8", timeout=30)
        except Exception as e:
            print(f"[错误] 无法运行 vault: {e}")
            print("  请确认 exe 路径, 且 vault 已解锁 (omnivault unlock), 或提供 --token。")
            sys.exit(1)
        if out.returncode != 0:
            print(f"[错误] vault export 失败: {out.stderr.strip()}")
            print("  提示: 请先开启本地服务(OmniVault UI 的 启用本地服务)或用 --token 提供服务令牌。")
            sys.exit(1)
        data = json.loads(out.stdout or "{}")

    fields = {}
    cats = data.get("categories") or {}
    for cat, items in cats.items():
        for it in items:
            fid = it.get("id") or ""
            fname = it.get("field_name") or ""
            val = it.get("value") or ""
            fields[fid] = val
            if fname and fname != fid:
                fields[fname] = val
    return fields


def _best_page(page):
    """取 Edge 当前激活标签; 否则取第一个非空网页。"""
    for ctx in page.context.pages:
        if ctx.url and ctx.url not in ("about:blank",):
            return ctx
    return page


def fill_fields(page, fields, mapping, allow_unknown):
    filled = []
    used = set()  # 已被填的 JSON 键

    # 收集页面所有 label
    labels = []
    try:
        labels = page.locator("label").all()
    except Exception:
        labels = []

    # 预处理 label 文本 -> for_id / 包裹控件
    for lbl in labels:
        try:
            text = (lbl.inner_text() or "").strip()
        except Exception:
            continue
        if not text:
            continue
        for_id = lbl.get_attribute("for") or None

        # 在 vault fields 中找一个匹配此 label 文本的字段
        matched_key = _match_field(fields, text, mapping)
        if matched_key is None:
            continue
        value = fields[matched_key]
        if not value:
            continue

        loc = None
        if for_id:
            cand = page.locator(f"#{for_id}")
            if cand.count() and cand.first.is_visible():
                loc = cand.first
        else:
            inner = lbl.locator("input, select, textarea")
            if inner.count() and inner.first.is_visible():
                loc = inner.first
        if loc is None:
            continue
        try:
            loc.fill(str(value))
            filled.append(f"{matched_key} -> {text} = {value}")
            used.add(matched_key)
        except Exception:
            pass

    # --fill-unknown: 把所有未匹配到的 vault 字段, 用字段名里的词去 page 找 name/placeholder 填入
    if allow_unknown:
        for key, value in fields.items():
            if key in used or not value:
                continue
            # 用 key 的每一段分词尝试匹配 input name/placeholder
            for token in key.split("."):
                nl = page.locator(f'input[name*="{token}"], textarea[name*="{token}"], input[placeholder*="{token}"]')
                if nl.count():
                    try:
                        nl.first.fill(str(value))
                        filled.append(f"{key}(unknown) -> {token} = {value}")
                        used.add(key)
                        break
                    except Exception:
                        pass
    return filled, used


def _match_field(fields, label_text, mapping):
    """在 fields 中找最匹配此 label 的字段键。优先级: mapping=字段中西文包含=关键词=同义词。"""
    lt = label_text.lower()
    # 1) 手动映射
    for key, token in mapping.items():
        if token.lower() in lt and key in fields:
            return key
    # 2) vault 字段名/键 直接包含在 label 中
    for key in fields:
        kl = str(key).lower()
        if kl and kl in lt:
            return key
    # 3) 字段关键字 -> 中文 label 同义匹配 (vault 字段是英文 id, 网页 label 是中文)
    for key in fields:
        kl = str(key).lower()  # 如 p1_identity.phone / citizenship_etc
        key_word = kl.rsplit(".", 1)[-1]  # 取末段关键词 phone / birth_place
        for kw in key_word.replace("_", "").split():
            for cn in FIELD_KEYWORDS.get(key_word, []) + FIELD_KEYWORDS.get(kw, []):
                if cn in lt:
                    return key
    # 4) 通用同义词表
    for key, toks in SYNONYMS.items():
        if any(t.lower() in lt for t in toks):
            for fk in fields:
                fk_l = str(fk).lower()
                if key in fk_l or any(t in fk_l for t in toks):
                    return fk
    return None


def main():
    ap = argparse.ArgumentParser(description="接管 Edge, 从 OmniVault 取数据自动填表")
    ap.add_argument("--debug-port", type=int, default=DEFAULT_PORT)
    ap.add_argument("--omnivault", default=VAULT_EXE)
    ap.add_argument("--mapping", default=None)
    ap.add_argument("--fill-unknown", action="store_true", help="把未精确匹配的 vault 字段也按 name/placeholder 尝试填入")
    ap.add_argument("--token", default=None, help="服务令牌(开启本地服务后获得), 优先用它走 HTTP 读取字段")
    ap.add_argument("--addr", default="http://127.0.0.1:7200", help="vault 服务地址, 默认 127.0.0.1:7200")
    ap.add_argument("--no-wait", action="store_true", help="填完后不等待回车, 自动断开连接")
    args = ap.parse_args()

    fields = vault_fields(args.omnivault, token=args.token, addr=args.addr)
    print(f"从 OmniVault 读取 {len(fields)} 个字段值\n")

    mapping = load_mapping(args.mapping)

    try:
        from playwright.sync_api import sync_playwright
    except ImportError:
        print("缺少 playwright, 请运行: pip install playwright")
        sys.exit(1)

    cdp = f"http://127.0.0.1:{args.debug_port}"
    with sync_playwright() as p:
        try:
            browser = p.chromium.connect_over_cdp(cdp)
        except Exception as e:
            print(f"[错误] 无法连接到调试端口 {cdp}")
            print("  请先运行 edge_start.bat 启动带调试端口的 Edge。")
            print(f"  原始错误: {e}")
            sys.exit(1)
        ctx = browser.contexts[0] if browser.contexts else browser.new_context()
        pages = [c for c in ctx.pages if c.url and c.url not in ("about:blank",)]
        if not pages:
            page = ctx.new_page()
            page.goto("about:blank")
            pages = [page]
        target = pages[0]
        print(f"接管 Edge, 当前页面: {target.url}\n")

        filled, used = fill_fields(target, fields, mapping, args.fill_unknown)
        if filled:
            print("已填写字段:")
            for f in filled:
                print("  -", f)
        else:
            print("未匹配到任何字段。可尝试 --fill-unknown, 或提供 --mapping map.txt。")
        unfilled = [k for k in fields if k not in used]
        print("\n本次未填入的 vault 字段数:", len(filled), " | 已匹配使用:", len(used))
        if not args.fill_unknown:
            print("提示: 加 --fill-unknown 会用字段名尝试匹配未被精确命中的控件。")

        # 停留让用户确认
        if not args.no_wait:
            input("\n请核对 Edge 中的填写结果, 按回车关闭连接 (不关闭 Edge)。")

        # 注意: 只断开连接, 不关闭浏览器
        print("已断开与 Edge 的连接(Edge 保持打开)。")


if __name__ == "__main__":
    main()