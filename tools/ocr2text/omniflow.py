# -*- coding: utf-8 -*-
"""
OmniVault 一站式工作流（唯一入口）。

一个程序搞定全部：
  vault 服务检查 + Ollama 自启 + 模型检查 + 材料转文本 + 字段归一化，
  然后交互式让用户选择下一步：
    [1] 写入 vault 存档      归一化字段自动按分类写入 OmniVault
    [2] 接管 Edge 填网页      读取 vault 字段接管真实 Edge 自动填表
    [3] 直接退出

用法:
    python omniflow.py <材料文件夹> [--token <服务令牌>] [--addr http://127.0.0.1:7200]

    --token  服务令牌(OmniVault UI「AI 接入」开启 local server 后获得);
             提供后写档/填网页默认走 HTTP, 无需解锁交互。
    --addr   vault 服务地址, 默认 http://127.0.0.1:7200

环境变量(可覆盖)：
    LLM_BASE_URL  默认 http://localhost:11434/v1
    LLM_MODEL     默认 qwen3:8b
"""

import sys
import os
import json
import time
import urllib.request
import subprocess
from pathlib import Path

HERE = Path(__file__).resolve().parent
PY = HERE / ".venv" / "Scripts" / "python.exe"
if not PY.exists():
    PY = sys.executable

VAULT_EXE = os.environ.get("OMNIVAULT_EXE", "omnivault")
OLLAMA = os.environ.get("OLLAMA_EXE", "")
MODEL = os.environ.get("LLM_MODEL", "qwen3:8b")
VAULT_ADDR = os.environ.get("VAULT_ADDR", "http://127.0.0.1:7200")
TOKEN = None

CN = {
    1: ("写入 vault 存档", "把归一化字段按分类写入 OmniVault，作为个人档案的一部分"),
    2: ("接管 Edge 填网页", "读取 vault 字段，接管调试端口上的 Edge 自动填表"),
    3: ("仅输出结果", "只在本地生成 _normalized.json，不写档也不填网页"),
    0: ("退出", None),
}


# ---------- 基础工具 ----------

def http(method, path, token=None, body=None, addr=VAULT_ADDR, timeout=10):
    """通用 HTTP 请求，返回 (status, json_or_text)。"""
    url = addr.rstrip("/") + path
    data = json.dumps(body).encode("utf-8") if body is not None else None
    req = urllib.request.Request(url, data=data, method=method)
    if body is not None:
        req.add_header("Content-Type", "application/json")
    if token:
        req.add_header("Authorization", f"Bearer {token}")
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            raw = resp.read()
            try:
                return resp.status, json.loads(raw)
            except Exception:
                return resp.status, raw.decode("utf-8", "ignore")
    except urllib.error.HTTPError as e:
        try:
            return e.code, json.loads(e.read())
        except Exception:
            return e.code, str(e)
    except Exception as e:
        return None, str(e)


def ask(prompt, default=None):
    if default is not None:
        print(f"{prompt} [默认 {default}]: ", end="")
        val = input().strip()
        return val if val else default
    return input(prompt + ": ").strip()


def menu():
    print("\n" + "=" * 56)
    print("  已完成：材料 -> 文本 -> 字段归一化")
    print("=" * 56)
    for k, (name, desc) in CN.items():
        tag = "" if name != "退出" else ""
        if k == 0:
            print(f"  [{k}] 退出")
        else:
            print(f"  [{k}] {name}")
            if desc:
                print(f"        {desc}")
    print("=" * 56)


# ---------- 1. Ollama 自启 + 模型检查 ----------

# Ollama 真实模型库候选位置(桌面版常自定目录; 默认 ~/.ollama/models 由 OLLAMA_MODELS 环境变量覆盖)
OLLAMA_MODELS_CANDIDATES = [
    os.environ.get("OLLAMA_MODELS", ""),
    os.path.expanduser("~/.ollama/models"),
]


def find_models_dir() -> str:
    """找到包含已安装模型的 Ollama models 目录(按存在的 manifest 判断)。"""
    for d in OLLAMA_MODELS_CANDIDATES:
        manifests = os.path.join(d, "manifests")
        for root, dirs, files in os.walk(manifests):
            if files:
                return d
    return None


def start_ollama() -> bool:
    """确保 Ollama 服务在线；不在则用 detached 方式后台启动。

    桌面版 Ollama 常把模型放在自定目录(如 D:\\data\\ollama models)，
    而裸启动 serve 会读默认 ~/.ollama/models(可能为空)。
    因此启动时显式注入 OLLAMA_MODELS 指向真实模型库。
    """
    st, _ = http("GET", "/api/version", addr="http://localhost:11434", timeout=2)
    if st == 200:
        # 已在线：核对 /api/tags 是否读到模型；若空则当前实例可能读错目录
        stt, tags = http("GET", "/api/tags", addr="http://localhost:11434", timeout=5)
        if stt == 200 and isinstance(tags, dict) and tags.get("models"):
            print("Ollama 服务已在运行，模型库就绪。")
            return True
        print("Ollama 已运行但模型库为空，将尝试用正确 OLLAMA_MODELS 重启。")

    print("正在启动 Ollama（自动定位真实模型目录）...")
    models_dir = find_models_dir() or os.path.expanduser("~/.ollama/models")
    if os.path.isdir(models_dir):
        print(f"  模型目录: {models_dir}")
    else:
        print(f"  [提示] 未找到模型目录，将用默认 {models_dir}")
    OLLAMA_DETACHED = None
    try:
        import subprocess as sp
        flags = 0x00000008 | 0x00000200  # DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP
        env = dict(os.environ)
        env["OLLAMA_MODELS"] = models_dir
        sp.Popen([OLLAMA, "serve"], stdout=sp.DEVNULL, stderr=sp.DEVNULL,
                 creationflags=flags, env=env, close_fds=True)
    except Exception as e:
        print(f"[提示] 自动启动 ollama 失败 ({e})，请手动运行 ollama serve。")
        return False
    for _ in range(20):
        time.sleep(1)
        st, _ = http("GET", "/api/version", addr="http://localhost:11434", timeout=2)
        if st == 200:
            print("Ollama 已就绪。")
            return True
    print("[提示] Ollama 启动超时，请手动运行 ollama serve 后重试。")
    return False


def ensure_model() -> bool:
    """确保目标模型已安装；未安装则询问是否自动拉取。通过 API 检测，稳定不挂起。"""
    installed = []
    # /api/tags 在服务就绪后可能需几秒才列出模型索引, 重试数次
    for _ in range(6):
        st, tags = http("GET", "/api/tags", addr="http://localhost:11434", timeout=8)
        if st == 200 and isinstance(tags, dict):
            installed = [m.get("name", "") for m in tags.get("models", [])]
            if installed:
                break
        time.sleep(2)
    if next((n for n in installed if n == MODEL), None):
        print(f"模型 {MODEL} 可用。")
        return True
    print(f"\n未检测到模型 {MODEL}（当前已有：{', '.join(installed) or '无'}）。")
    print("请确认已安装或先在 Ollama 中拉取: ollama pull " + MODEL)
    return False


# ---------- 2. vault 服务检查 ----------

def check_vault():
    """检查 vault 本地服务；返回atin成功与否。"""
    global VAULT_ADDR, TOKEN
    st, _ = http("GET", "/vault/status")
    if st == 200:
        print(f"vault 本地服务在线 ({VAULT_ADDR})。")
        return True
    print(f"\n未检测到 vault 本地服务 ({VAULT_ADDR})。")
    print("请先启动 OmniVault 并开启本地服务（UI 的「AI 接入」-> Enable local server），")
    print("或运行: omnivault unlock   (会询问档案密码后后台启动服务)")
    return ask("服务启动后再继续？输入 y 回车重试，或直接回车跳过/继续 [y/N]", default="n").lower() in ("y", "yes")


def get_token():
    """获取服务令牌：优先命令行/env，其次用户输入(可留空回退 CLI)。"""
    global TOKEN
    if TOKEN:
        return
    env_tok = os.environ.get("VAULT_TOKEN")
    if env_tok:
        TOKEN = env_tok
        return
    try:
        inp = input("服务令牌(可留空则用 CLI 方式): ").strip()
    except (EOFError, KeyboardInterrupt):
        inp = ""
    TOKEN = inp or None


# ---------- 3. 转文本 + 归一化 ----------

def run(script, *args, stage):
    print(f"\n========== 阶段: {stage} ==========")
    r = subprocess.run([str(PY), str(HERE / script), *args])
    if r.returncode != 0:
        print(f"[中止] {stage} 失败 (退出码 {r.returncode})")
        sys.exit(r.returncode)
    return r


def convert_and_normalize(material_dir: Path):
    out = material_dir / "_output"
    norm = material_dir / "_normalized.json"
    run("convert.py", str(material_dir), str(out), stage="转文本 (材料 -> Markdown)")
    run("normalize.py", str(out), str(norm), stage="字段归一化 (本地模型)")
    data = json.loads(norm.read_text(encoding="utf-8"))
    print(f"\n归一化结果 {len(data)} 个字段:")
    for k, v in data.items():
        print(f"  {k} = {v}")
    return data, norm


# ---------- 4. 写入 vault 存档 ----------

NORMALIZE_TO_CATEGORY = {
    "name": "identity", "gender": "identity", "birth_date": "identity",
    "nationality": "identity", "id_number": "identity", "ethnicity": "identity",
    "political_status": "identity", "native_place": "identity",
    "phone": "contact", "email": "contact", "address": "address",
}


def write_to_vault(data: dict):
    """把归一化字段写入 vault，返回成功数。无 token 时回退 CLI omnivault set。"""
    global TOKEN
    get_token()
    ok = 0
    used_prefixes = set()

    # 读取现有分类，尽量复用已有前缀(p1 等)保持一致
    st, ctx = http("GET", "/vault/context", token=TOKEN)
    cats = (ctx or {}).get("categories", {}) if st == 200 else {}
    for c in cats:
        if c.startswith("p"):
            used_prefixes.add(c.rsplit("_", 1)[0])

    print("\n写入 vault:")
    for key, value in data.items():
        if value in (None, ""):
            continue
        cat = NORMALIZE_TO_CATEGORY.get(key, "identity")
        prefix = f"p{len(used_prefixes) or 1}" if not used_prefixes else sorted(used_prefixes)[0]
        fname = key.split("-")[-1]
        field_id = f"{prefix}_{cat}.{fname}"
        if TOKEN:
            st, resp = http("PUT", f"/vault/fields/{field_id}", token=TOKEN,
                            body={"value": str(value)})
            if st in (200, 201, 204):
                print(f"  ✓ {field_id} = {value}")
                ok += 1
            else:
                print(f"  ✗ {field_id} = {value}  ({resp})")
        else:
            r = subprocess.run([VAULT_EXE, "set", f"{cat}.{fname}", str(value)],
                               capture_output=True, text=True, encoding="utf-8")
            if r.returncode == 0:
                print(f"  ✓ {cat}.{fname} = {value}")
                ok += 1
            else:
                print(f"  ✗ {cat}.{fname} = {value}  ({r.stderr.strip()})")
    print(f"写入完成: {ok}/{sum(1 for v in data.values() if v not in (None, ''))} 个字段入库。")
    return ok


# ---------- 5. 接管 Edge 填网页 ----------

def fill_edge_web():
    """调用 edge_fill.py 接管当前 Edge 填表。"""
    cmd = [str(PY), str(HERE / "edge_fill.py")]
    if TOKEN:
        cmd += ["--token", TOKEN, "--addr", VAULT_ADDR]
    print("\n接管 Edge 填表（需先以调试端口启动 Edge，见 README）。")
    return subprocess.run(cmd)


# ---------- 主流程 ----------

def main():
    global TOKEN, VAULT_ADDR
    argv = sys.argv[1:]

    # 交互式输入（无命令行参数时）：
    # 只传位置参数(文件夹)则用它，其余仍逐一询问；完全无参则全部交互询问
    def prompt_yes_no(text):
        try:
            return input(text).strip().lower() in ("y", "yes")
        except (EOFError, KeyboardInterrupt):
            return False

    # 1) 材料文件夹
    material = Path(argv[0]).resolve() if argv and not argv[0].startswith("--") else None
    if material is None:
        print(__doc__)
        print("=" * 56)
        while True:
            try:
                inp = input("\n① 请拖入或输入材料文件夹路径（内含 Word/PDF/图片）:\n> ").strip().strip('"')
            except (EOFError, KeyboardInterrupt):
                print("已取消。")
                sys.exit(0)
            if not inp:
                print("  未输入，已退出。")
                sys.exit(0)
            material = Path(inp).expanduser().resolve()
            if material.is_dir():
                break
            print(f"  ✗ 找不到目录: {material}，请重新输入。")

    # 2) 服务令牌（可选）
    if "--token" in argv:
        TOKEN = argv[argv.index("--token") + 1]
    elif not TOKEN:
        try:
            inp = input("\n② 服务令牌(可留空; OmniVault UI「AI 接入」开启 local server 后获得)\n> ").strip()
        except (EOFError, KeyboardInterrupt):
            inp = ""
        TOKEN = inp or None

    # 3) vault 服务地址（可选，默认 7200）
    if "--addr" in argv and argv.index("--addr") + 1 < len(argv):
        VAULT_ADDR = argv[argv.index("--addr") + 1].rstrip("/")
    else:
        try:
            inp = input(f"\n③ vault 服务地址(可留空, 默认 {VAULT_ADDR})\n> ").strip().rstrip("/")
        except (EOFError, KeyboardInterrupt):
            inp = ""
        if inp:
            VAULT_ADDR = inp

    if not material.is_dir():
        print(f"错误: 找不到材料目录 {material}")
        sys.exit(1)

    print("\n" + "=" * 56)
    print("OmniVault 一站式工作流启动")
    print("=" * 56)

    # 前置：模型 + 服务
    start_ollama()
    ensure_model()
    check_vault()
    get_token()

    # 转文本 + 归一化
    data, norm = convert_and_normalize(material)
    print(f"\n归一化 JSON 已保存: {norm}")

    # 用户选择下一步
    while True:
        menu()
        try:
            c = int(input("\n请选择下一步 [0-3]: ").strip())
        except (ValueError, EOFError):
            c = 3
        if c == 1:
            write_to_vault(data)
        elif c == 2:
            fill_edge_web()
        elif c == 3:
            print("本次仅输出归一化结果。")
        elif c == 0:
            print("退出。")
            return
        else:
            print("无效选择。")
            continue
        if not prompt_yes_no("\n是否返回主菜单继续? [y/N]: "):
            print("结束工作流。")
            return


if __name__ == "__main__":
    main()