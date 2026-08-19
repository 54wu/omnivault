<h1 align="center">OmniVault · 万象档案袋</h1>

**加密的个人上下文档案袋。钥匙永远在你手里。**

<p align="center">
  <a href="./LICENSE"><img src="https://img.shields.io/github/license/54wu/omnivault?v=1" alt="license"></a>
  <a href="https://github.com/54wu/omnivault/releases"><img src="https://img.shields.io/github/v/release/54wu/omnivault?v=1" alt="release"></a>
  <a href="https://github.com/54wu/omnivault/actions"><img src="https://img.shields.io/github/actions/workflow/status/54wu/omnivault/release.yml?v=1" alt="ci"></a>
  <img src="https://img.shields.io/github/go-mod/go-version/54wu/omnivault?v=1" alt="go">
</p>

---

你的人生只需填写一次。之后每位 AI 代理都从完整上下文开始，而不是一张白纸。

OmniVault 存储你的身份、文档、人际、财务、地址和偏好——每个字段都用从你的档案密码派生的密钥单独加密。代理只申请有权限范围（scope）的访问。你批准，令牌过期，密钥永不离机。

本 README 是本项目的**完整说明**：从特性、安装、界面、命令行、合并、API、MCP，到安全模型、目录结构、构建与发布、常见问题。

**中文** | [English](./README_EN.md)

---

## 目录

1. [特性](#特性)
2. [快速开始](#快速开始)
3. [安装](#安装)
4. [原生界面（档案袋 UI）](#原生界面档案袋-ui)
   - [登录与首次创建](#登录与首次创建)
   - [人员管理](#人员管理)
   - [两种视图：文档模板 / 全部数据](#两种视图文档模板--全部数据)
   - [门类大纲（可编辑）](#门类大纲可编辑)
   - [字段编辑：自定义 / 移动 / 排序 / 隐藏](#字段编辑自定义--移动--排序--隐藏)
   - [自动保存、搜索与导出](#自动保存搜索与导出)
   - [主题、语言与安全](#主题语言与安全)
   - [AI 接入（本地服务开关）](#ai-接入本地服务开关)
   - [UI 单进程架构](#ui-单进程架构)
5. [命令行 CLI](#命令行-cli)
6. [材料合并 Merge（三层分级）](#材料合并-merge三层分级)
7. [加密原理（安全模型）](#加密原理安全模型)
8. [跨设备同步（云同步 + 密钥分离）](#跨设备同步云同步-密钥分离)
9. [版本化自动备份与回档](#版本化自动备份与回档)
10. [HTTP API](#http-api)
11. [附件](#附件)
12. [服务令牌](#服务令牌)
13. [敏感级别](#敏感级别)
14. [数据模型与字段 ID 约定](#数据模型与字段-id-约定)
15. [环境变量](#环境变量)
16. [MCP 服务](#mcp-服务)
17. [本地模型接入（完全离线）](#本地模型接入完全离线)
18. [本地工作流：材料转文本 → 语义归一化 → 接管 Edge 自动填表](#本地工作流材料转文本--语义归一化--接管-edge-自动填表)
   - [一键一站式入口（推荐）](#0-一键一站式入口推荐)
   - [验证日志（2026-08-16，真实运行结果）](#31-验证日志2026-08-16真实运行结果)
19. [购物演示](#购物演示)
20. [目录结构](#目录结构)
21. [构建与开发](#构建与开发)
22. [发布（Release）](#发布release)
23. [测试](#测试)
24. [常见问题 FAQ](#常见问题-faq)
25. [协议](#协议)
26. [致谢](#致谢)
27. [贡献指南](#贡献指南-contributing)
28. [安全说明](#安全说明-security)
29. [许可证](#许可证)

---

## 特性

- **逐字段加密** — 每个字段用 AES-256-GCM 单独加密。
- **零知识** — 档案密码从不落盘；解锁期间密钥只存在于内存。
- **原生界面** — Windows 10/11 单进程 WebView2 原生窗口。无后台守护进程、无 TCP 端口、无网络依赖，任何网络过滤驱动都无法拦截。
- **多人员档案袋** — 一个档案库可管理多位人员（家庭成员、同事等），每位人员独立加密存储、独立归档。
- **文档模板** — 从模板库给每位人员放置所需文档（简历、体检表、入职登记表…），模板只是对底层字段的引用，数据与「全部数据」视图共享。
- **可编辑门类大纲** — 左侧大纲支持重命名、新增、删除门类，字段可在门类间移动。
- **命令行动态字段** — 分类和字段名可任意使用，`set/get/list` 即写即读。
- **命令行** — 完整的 CLI，便于脚本化和代理调用。
- **材料合并** — 人机协同的三层分级合并，把外部材料（如招聘申请表）并入某位人员的档案。
- **MCP 服务** — 内置 TypeScript [MCP](https://modelcontextprotocol.io/) 服务，让 AI 代理按作用域申请访问你的上下文。
- **版本化自动备份** — 解锁后、每次写入后自动快照，支持回档。
- **跨设备同步** — 加密的 `vault.db` 可同步到任何地方，密钥始终留在本机。
- **附件** — 每个字段可挂载多个加密附件。
- **服务令牌** — 长期、有作用域范围的凭据，供应用/代理使用。
- **敏感级别** — 每个字段可分 `public / standard / sensitive / critical` 四级。
- **审计日志** — 所有访问写入 `vault_access_log`，可随时查看。
- **安全提问** — 连续输错 5 次密码后，可用找回安全提问重置。

---

## 快速开始

```sh
# Windows（双击）：
#   运行源码根目录的 OmniVault.exe，首次运行在窗口内引导创建档案库。

# macOS / Linux：
curl -fsSL https://raw.githubusercontent.com/54wu/omnivault/main/install.sh | sh
omnivault onboard            # 交互式：创建档案库 + 解锁 + 填充常用字段
```

开箱即用：`omnivault ui` 打开原生界面；`omnivault set identity.full_name "张三"` 写入字段；`omnivault get identity.full_name` 读回。

> 请把密钥（secret key）妥善保存。解锁时需要**档案密码 + 密钥**两者。

---

## 安装

### Windows

从 [Releases](https://github.com/54wu/omnivault/releases) 页面下载最新 `omnivault.exe`，或从源码构建：

```sh
git clone https://github.com/54wu/omnivault.git
cd omnivault
./build.ps1            # 一键编译，生成根目录 OmniVault.exe
```

双击 `OmniVault.exe`，首次运行会在窗口内引导你创建档案库。

### macOS / Linux

```sh
curl -fsSL https://raw.githubusercontent.com/54wu/omnivault/main/install.sh | sh
omnivault onboard
```

<details>
<summary>方式二：从源码构建</summary>

```sh
git clone https://github.com/54wu/omnivault.git
cd omnivault && make build && make install
omnivault onboard
```
</details>

<details>
<summary>方式三：用 Go 直接安装</summary>

```sh
go install github.com/54wu/omnivault/cmd/omnivault@latest
omnivault onboard
```
</details>

`install.sh` 支持 macOS（arm64/amd64）与 Linux（arm64/amd64），自动从 GitHub Releases 下载对应平台的压缩包，必要时对 macOS 二进制执行 ad-hoc 签名以通过 Gatekeeper。

---

## 原生界面（档案袋 UI）

双击 `OmniVault.exe`（或运行 `omnivault ui`）会在原生 WebView2 窗口中打开档案库，窗口标题为 **OmniVault · 万象档案袋**。

### 登录与首次创建

- **首次运行**：窗口内显示「创建档案库」表单，设置档案密码（至少 8 位）后生成并展示**密钥（secret key）**，请务必妥善保存。
- **再次运行**：输入档案密码解锁。解锁后进入主界面。
- 顶部进度条显示当前人员的填写进度（`已填写 / 总字段`），**按该人员的全部字段统一计算**，在「文档模板」与「全部数据」两个视图下完全一致。

### 人员管理

顶部人员栏以标签页展示档案库中的每位人员：

- **+ 添加人员**：新建档案（每人拥有独立的数据命名空间，绝不互相串数据）。
- **✎ 重命名**：修改档案显示名称（不影响底层数据）。
- **× 删除**：删除档案并级联删除该人员全部字段与附件（需确认）。
- 切换标签即可查看/填写不同人员。

### 两种视图：文档模板 / 全部数据

工具栏右上角可切换：

- **文档模板**：把当前人员按「文档」组织。左侧「+ 添加文档」从模板库选择该档案袋需要的文档（简历、体检表、入职登记表…）。每个模板只是对底层字段的引用，数据与「全部数据」共享，一处填写处处同步。模板内支持增删条目、保存为模板、移除文档。
- **全部数据**：当前人员的全量扁平表单，按门类（分类）分组展示所有字段（含隐藏的底层字段与自定义字段）。

### 门类大纲（可编辑）

左侧「门类大纲」在「全部数据」视图中列出所有门类，可点击跳转到对应区块，并支持：

- **✎ 重命名门类**：修改门类显示名称与说明。
- **× 删除门类**：从当前人员的「全部数据」中隐藏该门类（底层字段数据仍保留在保管库）。
- **+ 添加门类**：新建自定义门类，之后可在其中添加字段；新门类也会出现在「归属门类」与「移动字段」选择器中。

以上编辑均按人员持久化，重启后保留。

### 字段编辑：自定义 / 移动 / 排序 / 隐藏

- **+ 字段**：在当前门类下新建自定义字段（可设字段 ID、显示标签、类型：文本/密码/邮箱/电话/日期/长文本、敏感度、占位符）。
- **归属门类**：新建字段时可指定归属门类（含用户添加的自定义门类）。
- **移动字段**：把字段移动到其他门类，其已填数据一并迁移。
- **拖拽排序**：字段可拖动排序（按人员/按文档分别保存）；门类区块也可拖拽重排。
- **移除/删除**：非自定义字段可从该档案袋移除（底层数据保留）；自定义字段可删除（已填内容一并删除，需确认）。
- 字段使用敏感度圆点标注级别（public/standard/sensitive/critical）。

### 自动保存、搜索与导出

- **自动保存**：字段失焦（或按回车）即写入，带 已保存/保存中/错误 状态提示；同一底层字段在多个模板中出现时实时同步。
- **搜索**：顶部搜索框按标签或内容过滤，自动切换到「全部数据」视图。
- **导出**：把当前人员的全部已填写信息导出为 **JSON / Markdown / HTML**。

### 主题、语言与安全

- 右上角 **◐** 切换深色/浅色主题（默认浅色），偏好本地持久化。
- **EN** 按钮切换中/英文界面。
- **修改密码**：用新密码重新加密全部字段（含附件），期间勿关闭窗口。
- 连续输错 5 次密码会提示使用找回安全提问。

### AI 接入（本地服务开关）

如需让外部 AI / MCP 客户端访问档案库，在「AI 接入引导」弹窗中打开 **「开启本地服务」** 开关，会在同一进程内监听 `127.0.0.1:7200`（与 `omnivault unlock` 效果一致）。关闭开关或关闭窗口即自动停止监听并清除内存密钥（**关窗即关**），无需手动管理后台进程。

「AI 接入引导」弹窗（右上角 **AI** 按钮）内置了各种接入方式的现成命令，可直接复制：

- **当前会话令牌**：本次会话的 Bearer Token（仅本机有效，重启后失效）。
- **curl 调用**：`GET http://127.0.0.1:7200/vault/fields` 读取全部字段、按字段 ID 读取单个字段、`PUT` 写回字段。
- **MCP 配置**：在 `.mcp.json` 中指向本地 TypeScript MCP 编译产物（`node mcp/dist/index.js`）并注入 `VAULT_ADDR` / `VAULT_DIR` / `VAULT_SCOPE` / `VAULT_CONSUMER` 环境变量。
- **服务令牌**：会话令牌随窗口关闭失效；如需长期、有作用域范围的接入（如常驻代理），用 CLI 创建服务令牌并设置 `VAULT_TOKEN`（详见下方[服务令牌](#服务令牌)）。
- **给 AI 的提示词示例**、**让 AI 根据外部材料自动填表**：一键复制提示词，交给已接入的 AI。

### UI 单进程架构

界面与档案库在**单进程**内通过 Go 桥接（WebView2 Bind）通信，默认不走 HTTP，因此不涉及任何 TCP 端口或命名管道，档案库永远不会「不可达」。界面 HTML 内嵌于二进制中，并在每次启动时写入 `~/.omnivault/ui-data/onboarding.html` 以 **`file://` 来源**加载，确保浏览器 `localStorage`（人员列表、字段顺序、自定义字段、主题、语言等）真正持久化、跨重启保留。

**自动刷新**：界面每 5 秒静默重新拉取一次 `/vault/context`，当检测到数据变化（例如通过 HTTP / MCP / AI 从外部写入字段）且当前没有正在编辑的输入框时，会自动重建当前档案视图。因此外部消费方写入的数据无需重启界面即可在最多 5 秒内显示出来。

---

## 命令行 CLI

```sh
omnivault onboard                        # 创建档案库、解锁并填充常用字段（交互式）
omnivault init                           # 创建新档案库
omnivault ui                             # 打开原生窗口界面（单进程）
omnivault status                         # 查看档案库状态
omnivault schema                         # 查看推荐的字段命名（--json 输出原始 JSON）

omnivault set <id> <value>               # 设置字段
omnivault get <id>                       # 读取字段
omnivault list [category]                # 列出字段
omnivault delete <id>                    # 删除字段
omnivault set-sensitivity <id> <tier>    # 设置敏感级别 (public|standard|sensitive|critical)
omnivault export                         # 导出全部解密字段为 JSON
omnivault audit                          # 查看访问审计日志

omnivault merge <material.json>          # 把外部材料并入某位人员的档案（三层分级）
omnivault backup <dest>                  # 复制加密 vault.db 到同步目录（密钥留在本机）
omnivault restore <src>                  # 从同步目录复制 vault.db 回档案库
omnivault rollback [name]                # 列出版本化备份，或回档
omnivault set-security-question          # 设置找回安全提问（连续输错 5 次密码后触发）
omnivault change-password                # 修改档案密码（重新加密所有字段）

omnivault create-service-token <consumer> # 创建长期服务令牌
omnivault list-service-tokens            # 列出有效令牌
omnivault revoke-service-token <prefix>  # 按前缀撤销令牌

# 守护进程（仅供 MCP / HTTP 消费者使用）
omnivault unlock                         # 解锁（启动后台服务守护进程）
omnivault lock                           # 上锁（停止守护进程，清空密钥）
omnivault serve                          # 前台运行服务守护进程

omnivault help                           # 打印全部命令帮助
```

字段使用点号命名：`identity.full_name`、`addresses.current.city`、`financial.filing_status`。分类和字段名可任意使用。

---

## 材料合并 Merge（三层分级）

把外部材料（如招聘网站导出的申请表）并入某位人员的档案，人机协同、自动备份、全程审计。可通过 **CLI**、**HTTP**、**MCP** 三种入口使用。

### 三层分级流程

1. **归属校验**：用材料的 `person_hint`（姓名/证件号/手机号/邮箱加权，权重 0.25/0.35/0.25/0.15）与档案库中现有人员交叉比对；归属存疑时返回候选供用户选择（`--person <id>` / `person` 参数可强制指定目标）。
2. **三级分级**：每条材料映射到档案字段后打标——
   - `auto`：值相同（无需处理）或低敏空字段的安全补填（自动可采纳）
   - `batch`：低敏字段的值冲突（需用户决定）
   - `manual`：高敏/敏感字段或列表合并（需用户决定）
3. **决定并应用**：用户对 batch/manual 项填 `action`（keep/replace/fill/add/skip），只写入这些动作；写入自动触发 `BackupNow()` 快照并留审计。

> 未映射到档案字段的标签会被忽略并提示，不写库。

### CLI 用法

```sh
# dry-run：生成决策计划（打印到终端）
omnivault merge material.json --person p1

# 保存决策计划到文件
omnivault merge material.json --person p1 --plan plan.json

# 直接采纳所有 auto 项，并输出剩余待决项
omnivault merge material.json --person p1 --auto

# 应用决策计划（只写 fill/replace/add 动作）
omnivault merge --apply plan.json
```

材料文件格式（可参考仓库根目录 `merge-material-sample.json`）：

```json
{
  "source": "example.com",
  "person_hint": { "name": "徐小明", "id_number": "440101199501011234", "phone": "13900001111", "email": "xuxiaoming@example.com" },
  "items": [
    { "label": "硕士院校", "value": "示例大学" },
    { "label": "手机号", "value": "13900001111" }
  ]
}
```

### 标签到字段的映射

材料中的中文标签（如「硕士院校」「手机号」）会先解析为规范字段（如 `identity.email`、`education.postgrad_school`），再按人员前缀组装成实际字段 ID（`p1_identity.email`）写入。可解析的标签范围覆盖档案字段的常见中文命名。

---

## 加密原理（安全模型）

```
档案密码 + 密钥 (128-bit)
  → Argon2id KDF (64MB, 3 次迭代)
  → 档案主密钥 Vault Key (256-bit, 仅存内存)
  → HKDF 按分类派生 → 分类子密钥
  → 每个字段 AES-256-GCM (12 字节随机 nonce)
```

- 档案密码**从不存储**；密钥保存在本机 `~/.omnivault/secret.key`（权限 0600），从不参与同步、从不落库。
- 解锁期间主密钥**只存在于内存**，上锁/关窗即清零。
- 每个字段**独立加密**，即使数据库被窃取，得到的也只是密文。
- 30 分钟无活动自动上锁（每次鉴权请求都会重置计时器）。
- 会话令牌：32 字节 `crypto/rand` 生成，常量时间比较。
- 每次访问写入 `vault_access_log` 审计。

---

## 跨设备同步（云同步 + 密钥分离）

把档案库想象成保险柜：保险柜（加密的 `vault.db`）可以带到任何设备，但钥匙（`secret.key`）始终跟着你。

```
设备 A                      云盘                       设备 B
omnivault backup D:\sync   →    vault.db  →→ 同步 →→   omnivault restore D:\sync
(secret.key 留在本机)        (仅密文)                 + 输入一次密码和密钥
```

1. `omnivault backup <folder>` — 只把加密的 `vault.db` 复制到你同步的文件夹（OneDrive、坚果云、git 等），密钥绝不包含在内。
2. 在另一台设备上，`omnivault restore <folder>` 把同步过来的 `vault.db` 复制进它的档案库目录。
3. `omnivault unlock` 输入一次档案密码和密钥即可。密钥可以手动输入；只有 `vault.db` 参与同步。

---

## 版本化自动备份与回档

档案库会自动快照加密数据库：

- **每次解锁后**、**每次写入后**（3 秒防抖合并）触发。
- 备份存放在 `~/.omnivault/backups/`，命名为 `vault-YYYYMMDD-HHMMSS.db`。
- 保留最近 **3 份**，更旧的自动清理。

```sh
omnivault rollback                      # 列出版本化备份
omnivault rollback vault-20260815-120000.db   # 回档到指定备份（需先 lock）
```

---

## HTTP API

服务守护进程运行在 `http://127.0.0.1:7200`，受保护接口需携带 `Authorization: Bearer <token>`。

```
GET    /vault/status                    # 档案库状态（公开）
GET    /vault/schema                    # 推荐的字段命名与敏感级别（公开）
GET    /ui                              # 引导表单（公开）
POST   /vault/unlock                    # 解锁 → 会话令牌

GET    /vault/fields                    # 列出字段元数据（不含值）
GET    /vault/fields/{id}               # 读取字段及解密值
PUT    /vault/fields/{id}               # 设置字段（{ value, sensitivity? }，upsert）
DELETE /vault/fields/{id}               # 删除字段
GET    /vault/fields/category/{name}    # 某分类下的全部字段（含值）

GET    /vault/context                   # 按分类返回完整解密内容

PUT    /vault/sensitivity/{id}          # 更新敏感级别

POST   /vault/merge/plan                # 材料合并 dry-run：分级成决策计划（不写库）
POST   /vault/merge/apply               # 应用合并决策计划（只写 fill/replace/add）

POST   /vault/tokens/service            # 创建服务令牌
GET    /vault/tokens/service            # 列出服务令牌
DELETE /vault/tokens/service/{prefix}   # 撤销服务令牌

POST   /vault/lock                      # 上锁（清零密钥）
GET    /vault/audit?limit=50            # 访问审计日志
```

`GET /vault/context` 返回结构示例：

```json
{
  "categories": {
    "identity": [
      { "id": "p1_identity.name", "category": "p1_identity", "field_name": "name", "value": "徐小明", "sensitivity": "standard" }
    ],
    "p1_education": [...]
  }
}
```

---

## 附件

每个字段可挂载多个附件。上传加密的二进制内容，再按字段列出、下载或删除。

```
POST   /vault/attachments?field=<id>    # 上传附件 (multipart，带 X-Filename 头)
GET    /vault/attachments?field=<id>    # 列出某字段的附件
GET    /vault/attachments/{id}          # 下载附件
DELETE /vault/attachments/{id}          # 删除附件
```

附件用 AES-256-GCM 加密，修改档案密码时会一并重新加密。删除字段会级联删除其附件。

---

## 服务令牌

服务令牌允许应用以长期、有作用域范围的凭据访问档案库（参考 1Password service account 模式）。

```sh
omnivault create-service-token tax-agent --scope "identity.*,financial.*" --ttl 1h
omnivault create-service-token life --scope "*" --ttl 8760h
omnivault list-service-tokens
omnivault revoke-service-token abc123
```

每次通过鉴权的请求都会重置 30 分钟自动上锁计时器，保持档案库在消费方活跃期间持续解锁。

---

## 敏感级别

| 级别 | 示例 | 行为 |
|------|------|------|
| `public` | 姓名、时区 | 自动共享给已授权消费者 |
| `standard` | 地址、雇主 | 按请求共享，并记录日志 |
| `sensitive` | 出生日期、护照号 | 需要显式批准 |
| `critical` | 身份证号、银行卡路由号 | 需要批准 + 二次验证 |

```sh
omnivault set-sensitivity identity.id_number critical
omnivault set-sensitivity preferences.timezone public
```

新字段默认级别为 `standard`。推荐字段命名与默认级别见 `omnivault schema`。

---

## 数据模型与字段 ID 约定

- **底层存储**：字段存于 SQLite `vault_fields` 表，字段 ID 形如 `分类.字段名`（如 `identity.name`、`p1_education.edu1_school`）。
- **CLI / HTTP（单人员模型）**：使用 `category.field`（如 `identity.full_name`）。分类和字段名可任意使用。
- **UI（多人员模型）**：每位人员在内部对应一个人员 ID（`p1`、`p2`…），其字段 ID 带人员前缀：`pN_分类.字段`，例如 `p1_identity.name`。同一人员 ID 的所有字段共享该前缀。
- **人员 ID 唯一性**：新建人员时自动分配一个**不与任何已有数据冲突**的 ID（同时扫描人员列表与保管库中已存在的 `pN_` 前缀），因此新档案永远是干净的，绝不会继承其他人的数据。
- **孤儿档案自动恢复**：启动时会扫描保管库中存在的 `pN_` 前缀，凡是没有对应标签页的人员会自动恢复成标签（显示名取自其 `identity.name` 字段），避免数据因本地设置丢失而不可见。
- **历史人员 ID 自动迁移**：旧版本曾用不带 `p` 前缀的纯数字 ID（如 `1`）创建人员，导致界面标签指向 `1_...` 而数据实际存在 `p1_...` 下、所有字段显示为空。启动时会自动把这种裸数字 ID 迁移为对应的 `pN` ID（含该人员的自定义字段、隐藏字段、门类编辑、排序、模板等 localStorage 记录一并迁移），并同步校正当前选中人员。
- **合并桥接**：材料合并工具把中文标签解析为规范字段后，按目标人员前缀组装成 `pN_分类.字段` 写入。
- **附件**：存储于 `vault_attachments`，按字段 ID 关联。

UI 默认提供 30+ 门类，覆盖：基本信息、住址信息、工作信息、教育经历、财务社保、银行卡/信用卡、医疗健康、紧急联系人、证件文档、社交账号、家庭信息、实习与项目、证书与获奖、求职意向、工作经历、项目经验、专业技能、证书与荣誉、语言能力、自我评价、作品集与链接、培训经历、论文与著作、专利、学生工作/社会实践、保险信息、资产信息、订阅与会员、设备与保修、安全备忘、人脉关系、偏好设置等。教育经历门类还内置国内教育阶段（幼儿园/小学/初中/高中）与高等教育（教育1/2/3）分段字段。

---

## 环境变量

| 变量 | 默认值 | 用途 |
|------|--------|------|
| `VAULT_DIR` | `~/.omnivault` | 档案库目录 |
| `VAULT_ADDR` | `http://127.0.0.1:7200` | CLI 访问的服务地址 |
| `VAULT_PORT` | `7200` | `serve` 监听端口 |
| `VAULT_TOKEN` | — | 服务令牌（MCP/HTTP 消费方，覆盖会话文件） |
| `VAULT_SCOPE` | — | 消费方允许的字段作用域（如 `*` 或 `identity.*,financial.*`） |
| `VAULT_CONSUMER` | — | 消费方名称（用于审计日志标识） |

---

## MCP 服务

项目内置一个 TypeScript 编写的 [MCP](https://modelcontextprotocol.io/) 服务，让 AI 代理可以访问你的个人上下文。完整细节见 [mcp/README.md](./mcp/README.md)。

### 安装与注册

```sh
npm install omnivault-mcp          # 或
npx -y omnivault-mcp@latest

# 注册到 Claude Code
claude mcp add vault -- npx -y omnivault-mcp@latest
```

### 工具

| 工具 | 说明 |
|------|------|
| `vault_status` | 检查档案库是否运行并已解锁 |
| `vault_get` | 按 ID 读取单个解密字段 |
| `vault_list` | 列出所有字段元数据（不含值） |
| `vault_context` | 按分类返回全部解密字段 |
| `vault_set` | 在档案库中写入加密字段 |
| `vault_merge_plan` | 材料合并 dry-run：把外部材料分级成决策计划，不写库 |
| `vault_merge_apply` | 应用合并决策计划（仅写 fill/replace/add，自动备份 + 审计） |

### 合并工作流（MCP）

`vault_merge_plan` + `vault_merge_apply` 实现「人机协同三层合并」（与 CLI/HTTP 同一套 `internal/merge` 逻辑）：

1. **归属校验**：材料的 `person_hint` 与现有人员交叉比对；归属存疑返回候选（`person` 参数强制指定）。
2. **三级分级**：`auto`（值相同/低敏空字段安全补填）/ `batch`（低敏字段冲突）/ `manual`（高敏或列表合并）。
3. **决定并应用**：对 batch/manual 填 `action`（keep/replace/fill/add/skip），`vault_merge_apply` 只写这些动作并自动备份、审计。

`vault_merge_plan` 的 `material` 参数示例：

```json
{
  "source": "example.com",
  "person_hint": { "name": "徐小明", "id_number": "4401...", "phone": "1390..." },
  "items": [
    { "label": "硕士院校", "value": "示例大学" },
    { "label": "手机号", "value": "13900001111" }
  ]
}
```

### 鉴权

MCP 服务器每次请求时自动解析令牌：

1. `VAULT_TOKEN` 环境变量 — 与服务令牌配合，适合常驻代理。
2. `~/.omnivault/.session` — `omnivault unlock` 产生的会话令牌。

令牌按请求解析，因此服务器能在档案库上锁/解锁周期中存活。

### 错误信息

| 错误 | 含义 |
|------|------|
| `vault: server not running` | 先运行 `omnivault unlock` 启动服务 |
| `vault: session expired` | 运行 `omnivault unlock` 刷新会话 |
| `vault: vault is locked` | 运行 `omnivault unlock` 解密 |
| `vault: not found` | 字段 ID 不存在 |
| `vault: not configured` | 无令牌——运行 `omnivault unlock` 或设置 `VAULT_TOKEN` |

### .mcp.json 示例

在项目根目录 `.mcp.json` 中直接指向编译产物并注入环境变量：

```json
{
  "mcpServers": {
    "vault": {
      "command": "node",
      "args": ["<项目绝对路径>\\mcp\\dist\\index.js"],
      "env": {
        "VAULT_ADDR": "http://127.0.0.1:7200",
        "VAULT_DIR": "C:\\Users\\<用户名>\\.omnivault",
        "VAULT_SCOPE": "*",
        "VAULT_CONSUMER": "trae"
      }
    }
  }
}
```

MCP 源码在 `mcp/` 目录（TypeScript，编译产物 `mcp/dist/`），修改后执行 `npm run build` 重新编译。

---

## 本地模型接入（完全离线）

**先说清楚一件事**：OmniVault 的 MCP 服务端本身**不调用大模型**——它只是把档案袋的读写工具暴露给 AI 代理。真正调用大模型（并决定 `base_url` / `api_key` / `model` 指向哪里）的是**你的 AI 代理宿主**，例如 Trae、Claude Code、Cursor 等。因此"本地 / 云端切换"的配置点，在于**代理宿主侧的模型配置**，而不是 OmniVault 的代码。

如果你的核心诉求是**材料、信息的接收完全离线、数据不出本机**，那么：

1. 在本地用 **Ollama** 跑一个开源模型（7B 级别即可，无需独显，CPU + 16GB 内存即可流畅运行）。
2. Ollama 暴露一个 **OpenAI 兼容接口**：`http://localhost:11434/v1`。
3. 把 Agent 的 `base_url` 指向它、`model` 填本地模型名，即可无缝替换，全程不联网。

### 为什么 7B 模型就够

"读档、写档"是**确定性 API 调用**，根本不耗模型；"网页填表"里唯一需要模型的是**字段语义对齐**（网页的「联系电话」→ 档案的 `contact.phone`），这是 7B 级模型（Qwen2.5-7B / Qwen3-8B）稳定胜任的能力。要求高的是"会做对"，不是"参数大"。

### 可切换本地 / 云端的配置示例

在 Agent 宿主中，把模型配置抽成环境变量或一个开关，即可一键切换：

```sh
# ---- 本地（完全离线）：Ollama 提供 OpenAI 兼容端点 ----
export LLM_BASE_URL="http://localhost:11434/v1"
export LLM_API_KEY="ollama"          # Ollama 可任意占位
export LLM_MODEL="qwen2.5:7b"        # 或 qwen3:8b

# ---- 云端：OpenAI 兼容的任意服务商 ----
export LLM_BASE_URL="https://api.openai.com/v1"
export LLM_API_KEY="sk-xxxxxxxx"
export LLM_MODEL="gpt-4o"
```

> 不同的 Agent 宿主配置位置不同（Trae 在模型设置里、Claude Code 用 `ANTHROPIC_BASE_URL` 等），但思路一致：**只要服务商暴露 OpenAI 兼容接口，把 `base_url` 指向它即可**。Ollama 的 `http://localhost:11434/v1` 正是这样一个兼容端点。

### 常见 Agent 宿主的本地接入方式

Ollama 新版提供 `ollama launch` 命令，可直接绑定本地模型并自动配好 `base_url` 启动你的 Agent，**无需手填环境变量**。已将本地模型拉好（如 `qwen3:8b`）后，直接运行：

```sh
# Hermes Agent（你正在用的）
ollama run hermes --model qwen3:8b

# OpenCode（你正在用的）
ollama launch opencode --model qwen3:8b

# Claude Code / Qwen Code / VS Code 等其它宿主
ollama launch claude --model qwen3:8b
ollama launch qwen   --model qwen3:8b
ollama launch vscode --model qwen3:8b
```

如果宿主不支持 `ollama launch`，则手动把模型的 `base_url` / `model` 指到本地即可：

| Agent 宿主 | 配置方式 | 本地值 |
|-----------|----------|--------|
| Hermes Agent | 环境变量 `OPENAI_BASE_URL` / `OPENAI_MODEL` | `http://localhost:11434/v1` / `qwen3:8b` |
| OpenCode | `opencode` 配置或环境变量 `OPENAI_BASE_URL` | `http://localhost:11434/v1` / `qwen3:8b` |
| Trae IDE | 模型设置里新增自定义模型，填接口地址 | `http://localhost:11434/v1` / `qwen3:8b` |
| WorkBuddy | 模型配置填 OpenAI 兼容接口 | `http://localhost:11434/v1` / `qwen3:8b` |

> 任一方式都只需把 `base_url` 指向 `http://localhost:11434/v1`、`model` 填 `qwen3:8b`，即可完全离线使用，数据不出本机。

### 用 Ollama 完全离线部署

```sh
# 1. 安装 Ollama（Windows 下载安装包，或）
curl -fsSL https://ollama.com/install.sh | sh

# 2. 拉取一个中文 + 工具调用稳定、7B 级别、CPU 可跑的模型
ollama pull qwen2.5:7b          # 推荐：约 4.7GB 量化，工具调用稳
# ollama pull qwen3:8b          # 如内存富余可选

# 3. 启动本地服务（默认已监听 11434，OpenAI 兼容端点为 /v1）
ollama serve

# 4. 验证（应返回模型列表，全程无网络）
curl http://localhost:11434/v1/models
```

### 完全离线接入本项目 / 本机推荐配置

| 硬件 | 推荐模型 | 能力 |
|------|----------|------|
| 当前机器（CPU + 16GB 内存，无独显） | `qwen2.5:7b`（或 `qwen3:8b`） | 读档、写档、字段语义对齐、多步填表，完全离线 |
| 更低配置（<16GB 内存） | `qwen2.5:3b` | 简单取数 / 字段对得上的填表，离线 |
| 加装独显（RTX 4060 8GB 以上） | `qwen2.5:14b` 及以上 | 更复杂动态网页、更快响应 |

接入后，Agent 通过 MCP 连接 OmniVault（见上文 [MCP 服务](#mcp-服务)），模型走 Ollama 本地端点，**材料与信息全程不离开本机**。

---

## 本地工作流：材料转文本 → 语义归一化 → 接管 Edge 自动填表

仓库提供了 `tools/ocr2text/` 下的工具脚本，串起一条完全离线的本地工作流：把日常材料（Word / PDF / 图片 / 文本）转成 Markdown，交给本地模型做字段语义归一化，最后**接管你真实打开的 Edge** 自动填表。

> 前置：Python 3.12 独立虚拟环境，依赖已装好（RapidOCR、PyMuPDF、python-docx、Playwright）。

### 0) 一键一站式入口（推荐）

不想手动分步执行？**双击 `tools/ocr2text/omniflow.bat`**（或运行 `python tools/ocr2text/omniflow.py`）打开交互式程序，程序会依次请你输入：

1. **① 材料文件夹** —— 拖入或输入内含 Word/PDF/图片的文件夹路径；
2. **② 服务令牌** —— 可留空（OmniVault UI「AI 接入」开启 local server 后获得）；
3. **③ vault 服务地址** —— 可留空，默认 `http://127.0.0.1:7200`。

随后自动执行：
1. 检测/启动 **Ollama**（自动定位真实模型目录）并核对 **qwen3:8b** 模型；
2. 检测 **vault 本地服务**（未运行会提示你先开启）；
3. 自动 **转文本**（`convert.py`）→ **字段归一化**（`normalize.py`）；
4. 归一化结束后弹出交互菜单，由你选择下一步：
   - `[1]` **写入 vault 存档** —— 把归一化字段按分类 `PUT` 进 OmniVault，成为个人档案数据；
   - `[2]` **接管 Edge 填网页** —— 读取 vault 字段，接管 `edge_start.bat` 启动的 Edge 自动填表；
   - `[3]` 仅本地输出 `_normalized.json`，不写档也不填网页。

进阶用法（跳过部分提问）：也可用命令行参数预先指定，未指定的仍需交互输入：

```sh
python tools/ocr2text/omniflow.py "你的材料文件夹" --token "你的服务令牌"
```

| 参数 | 作用 |
|------|------|
| `--token <TOKEN>` | 服务令牌；提供后写档/填网页走 HTTP，无需解锁交互 |
| `--addr <URL>` | vault 服务地址，默认 `http://127.0.0.1:7200` |

归一化已内置**关闭 qwen3 思考**（`/api/chat` + `think:false`），避免大段 reasoning 拖慢或超时。

> 说明：`--token` 之外，`VAULT_TOKEN` 环境变量也生效；两者都缺失时写档会回退 `omnivault set`（需要先 `omnivault unlock` 解锁）。

### 1) 材料转文本（RapidOCR 离线 OCR）

把含图片 / 扫描件的 Word、PDF、图片转成统一 Markdown（docx 表格也会转成 Markdown 表格）：

```sh
python tools/ocr2text/convert.py "你的材料文件夹"          # 默认输出 .md
python tools/ocr2text/convert.py "你的材料文件夹" --txt     # 改输出纯 .txt
```

输出写到 `<文件夹>/_output/`，保持原目录结构。支持的格式：`.docx`、`.pdf`（自动区分文本层 / 扫描件 OCR）、`.jpg/.png/.bmp/.webp/.tif`、`.txt/.md/.csv`。

### 2) 字段语义归一化（调本地 qwen3:8b）

读取转好的 Markdown，调用 Ollama 的 OpenAI 兼容接口，把"出生日期 / 生日 / 出生年月日"这类同义表达统一成标准字段（如 `birth_date`），并输出结构化 JSON：

```sh
python tools/ocr2text/normalize.py "你的材料文件夹/_output" "结果.json"
```

也可一条命令跑完这两步：

```sh
python tools/ocr2text/run_workflow.py "你的材料文件夹" --no-fill
```

默认读 `http://localhost:11434/v1` + `qwen3:8b`，可用环境变量 `LLM_BASE_URL` / `LLM_MODEL` 覆盖。

### 3) 接管 Edge 自动填表（从 OmniVault 直接取数）

这一步对接**你真实运行的 Microsoft Edge**，从 **OmniVault 已解密的字段**里按网页字段名自动填入。

**第一步：以调试端口启动 Edge**（双击 `tools/ocr2text/edge_start.bat`），它会用一条独立 Edge 打开浏览器；在此 Edge 中自由导航到目标网页即可：

```bat
tools/ocr2text\edge_start.bat        :: 以 --remote-debugging-port=9222 启动
```

> Edge 默认路径 `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`，如不同请改 `edge_start.bat`。

> **为什么不能直接接管你现有的 Edge？** 填表脚本通过 CDP（Chrome DevTools Protocol）与 Edge 通信，而 CDP 调试端口必须在 Edge **启动那一刻**就用 `--remote-debugging-port` 打开——它不是一个可以事后注入的开关。你日常直接双击打开的 Edge 没有这个参数，无法被"接着"接管。`edge_start.bat` 用**同一个用户数据目录**重启带调试端口的实例，因此能保留你的登录态和已打开的标签页，体验上无缝。

**第二步：开启本地服务并取得令牌**。在 OmniVault UI 的「AI 接入」对话框打开 **Enable local server**，得到一段服务令牌。

**第三步：接管并填表**。`edge_fill.py` 会用令牌通过 HTTP 读取 vault 全部字段，接管调试端口的 Edge 当前激活标签页，按网页 label 语义匹配后自动填写：

```sh
python tools/ocr2text/edge_fill.py --token "你的服务令牌"
```

常用参数：

| 参数 | 作用 |
|------|------|
| `--token <TOKEN>` | 服务令牌，走 HTTP 读取字段（推荐，vault 无需解锁交互） |
| `--fill-unknown` | 额外按字段名匹配网页 `name` / `placeholder` 兜底 |
| `--mapping map.txt` | 手动映射：每行 `字段名=页面label关键词` |
| `--addr <URL>` | vault 服务地址，默认 `http://127.0.0.1:7200` |

**中英字段自动匹配**已内置：vault 的英文 id（如 `p1_identity.date_of_birth`）会自动对应网页的中文 label（如「出生日期」）。匹配不上的会列出，可用 `--mapping` 手动指定。

### 3.1 验证日志（2026-08-16，真实运行结果）

用一个带 8 个字段的测试表单自动填表，从 vault 读取 159 个字段，最终 7/8 成功填入、数据与 vault 完全一致：

```text
从 OmniVault 读取 159 个字段值.
填写结果 (已匹配填写):
  p1_identity.name -> 姓名 = 徐小明
  p1_identity.phone -> 手机号 = 13900001111
  p1_identity.email -> 邮箱 = xuxiaoming@example.com
  p1_identity.date_of_birth -> 出生日期 = 2002-10-26
  p1_addresses.home_address -> 家庭住址 = 安徽省阜阳市颍东区阜阳五中纬一路南博万有限公司
  p1_identity.native_place -> 籍贯 = 安徽
  p1_identity.id_number -> 身份证号 = 440101199501011234
校验: 7/8 个字段被成功填入.
```

- `company` 留空是因为 vault 中该人员本就没有「公司」字段值——有值才填，语义匹配逻辑正确。
- 用一次样例表单进行回归验证可运行 `python tools/ocr2text/verify_fill.py`（自启动独立 Edge → 读字段 → 填表 → 前后校验对比）。

---

## 购物演示

一个自包含的演示，展示两个 MCP 服务协同工作：档案库提供个人上下文，模拟商店处理订单。一句话输入，订单确认输出——无需额外提问。

```sh
cd examples/shopping-demo && npm install
claude mcp add shop -- npx tsx examples/shopping-demo/src/index.ts
```

> 测试购物演示——用我的档案库数据订一件 T 恤。

代理从档案库读取你的姓名、邮箱和地址，浏览商店并下单。如果发现缺少信息（比如 T 恤尺码），它会询问你并建议存进档案库，方便下次使用。完整安装说明见 [examples/shopping-demo/README.md](./examples/shopping-demo/README.md)。

---

## 目录结构

```
.
├── cmd/omnivault/            # CLI 入口（main.go 分发全部子命令）
│   ├── main.go               # 子命令路由 + help
│   ├── cmd_*.go              # 各子命令实现（init/unlock/lock/serve/set/get/list/delete/
│   │                         #   set-sensitivity/export/audit/merge/onboard/backup/restore/
│   │                         #   rollback/security-question/change-password/tokens/ui/status/schema）
│   ├── child_windows.go      # Windows 守护进程启动/停止
│   ├── pipe_windows.go       # 命名管道（Windows 进程通信）
│   ├── dpi_windows.go        # Windows 高 DPI 与前台窗口
│   └── rsrc_windows_amd64.syso  # Windows 图标/版本资源
├── internal/
│   ├── api/                  # HTTP 服务、处理器、桥接、中间件
│   │   ├── ui/onboarding.html # 档案袋前端（单页应用，内嵌于二进制）
│   │   ├── ui.go             # go:embed 内嵌页面 + /ui 处理器
│   │   ├── handlers.go       # 字段/上下文/令牌/审计等接口
│   │   ├── handlers_attachments.go  # 附件接口
│   │   ├── handlers_merge.go # /vault/merge/plan 与 /vault/merge/apply
│   │   ├── server.go         # HTTP 服务器组装
│   │   ├── bridge.go         # WebView2 原生桥接（单进程 UI）
│   │   └── middleware.go     # Bearer 令牌鉴权
│   ├── crypto/               # KDF（Argon2id）、AES-256-GCM、HKDF 子密钥
│   ├── dpapi/                # Windows DPAPI 密钥保护（凭据管理器）
│   ├── merge/                # 材料合并：classify（三级分级）、map（标签→字段）
│   ├── store/                # SQLite CRUD（字段、附件、令牌、审计、元数据）
│   └── vault/                # 业务逻辑（init/unlock/lock、加密解密、会话、备份、schema）
├── mcp/                      # TypeScript MCP 服务（src + dist）
├── examples/shopping-demo/   # 购物演示
├── docs/                     # 使用文档（usage.md）
├── assets/                   # 图标与素材
├── build/                    # 构建配置（winres.json：Windows 图标/版本/清单）
├── tools/ocr2text/           # 辅助工具：材料转文本 → 归一化 → 接管 Edge 自动填表
├── .github/workflows/        # release.yml（push tag 触发 goreleaser）
├── .goreleaser.yml           # 多平台发布配置（darwin/linux × amd64/arm64）
├── build.ps1                 # Windows 一键编译（图标资源 + exe → 根目录）
├── 首次使用.ps1              # 一键启动（中文版）：首次使用 + 启动 + 无 exe 时自动构建
├── FirstRun.ps1              # 一键启动（英文版）
├── Makefile                  # build / test / install / clean
├── install.sh                # macOS/Linux 一键安装脚本
├── vercel.json               # 一键安装域名重定向 /install.sh
├── go.mod / go.sum           # Go 依赖
└── README.md                 # 本文件（完整文档）
```

### 运行时数据目录 `~/.omnivault/`

```
~/.omnivault/
├── vault.db          # SQLite（加密字段、附件、审计日志、令牌）
├── secret.key        # 128-bit 密钥 (权限 0600)
├── .session          # 会话令牌（解锁时创建）
├── omnivault.pid     # 运行中服务的 PID
├── backups/          # 版本化加密快照（保留 3 份）
└── ui-data/          # WebView2 配置 + 页面文件 + localStorage（界面持久化）
```

---

## 构建与开发

**技术栈**：Go 1.26+（纯 Go，无 CGO）、`modernc.org/sqlite`、`golang.org/x/crypto`（argon2/hkdf）、stdlib `net/http`、`jchv/go-webview2`。

```sh
# Windows
./build.ps1                     # 生成图标资源 → 编译根目录 OmniVault.exe
./build.ps1 -Tests              # 构建 + 运行全部测试
./build.ps1 -SkipIcon           # 跳过资源嵌入，仅生成裸 exe

# macOS / Linux
make build                      # 编译到 bin/omnivault
make test                       # go test -v -race ./...
make install                    # 安装到 /usr/local/bin
make clean                      # 清理构建产物

# 单层测试
go test -race ./internal/crypto/   # 加密层
go test -race ./internal/store/    # 存储层
go test -race ./internal/vault/    # 档案核心
go test -race ./internal/api/      # HTTP API
```

> 修改 `internal/api/ui/onboarding.html` 后重新构建即可生效（页面每次启动也会写入 `ui-data/` 覆盖旧版）。

---

## 发布（Release）

- **CI**：`.github/workflows/release.yml` — push 一个 `v*` 标签后，先跑 `go test -race ./...`，再由 [goreleaser](https://goreleaser.com) 构建并上传各平台产物（darwin/linux × amd64/arm64），同时生成 `checksums.txt`。
- **配置**：`.goreleaser.yml`（含 LICENSE、README 归档）。
- **发版流程**：可按以下步骤手动操作：
  1. 确认在 `main` 分支、工作区干净、改动已推送。
  2. 查看最近标签与自上一版以来的变更 `git log <last-tag>..HEAD --oneline`。
  3. 确定版本号（patch/minor/major），与用户确认。
  4. 运行 `go test -race ./...` 通过。
  5. 打标签并推送 `git tag -a v0.x.y -m "Release v0.x.y" && git push origin v0.x.y`。
  6. 在 GitHub Actions 页面确认 goreleaser 构建成功，产物出现在 Releases 页。

---

## 测试

```sh
make test    # 开启竞态检测（go test -v -race ./...）
```

测试使用临时目录（`t.TempDir()`），无需清理。所有测试均带 `-race` 运行。

---

## 常见问题 FAQ

- **新建人员却显示别人的数据？** 不会。人员 ID 分配会扫描保管库中已存在的 `pN_` 前缀，保证新档案为空且不与任何旧数据冲突；孤儿档案（本地设置丢失但保管库有数据）会自动恢复成标签。
- **重启后人员列表不见了？** 请确认使用的是新版（用 `file://` 加载界面）。旧版用 `SetHtml` 加载导致 `localStorage` 不可用，重启后列表重置；新版每次启动会把页面写入 `~/.omnivault/ui-data/onboarding.html` 后以 `file://` 导航加载，`localStorage` 真正持久化。
- **「开启本地服务」后外部代理连不上？** 确认开关已打开（监听 `127.0.0.1:7200`），并设置了正确的 `VAULT_TOKEN`（服务令牌）或已 `omnivault unlock` 生成会话。
- **忘记档案密码怎么办？** 设置了找回安全提问则按提示回答重置；否则密钥+密码缺一不可，无法恢复（这是零知识设计的代价）。
- **`vault: server not running`**：先运行 `omnivault unlock` 或在 UI 中打开「开启本地服务」。

---

## 协议

本项目是 **[Personal Context Protocol](https://github.com/54wu/personal-context-protocol)** 的参考实现——一个开放协议，用于 AI 代理访问个人上下文。可阅读[协议规范](https://github.com/54wu/personal-context-protocol/blob/main/specification.md)。

---

## 致谢

本项目基于 **[Personal Vault](https://github.com/54wu/personal-vault)** 项目开发，并在其基础上进行了品牌更名（OmniVault · 万象档案袋）、界面重构（单进程 WebView2 原生窗口）、功能扩展（版本化自动备份与回档、附件、服务令牌、材料合并、多人员档案袋等）。在此对原项目及其作者表示诚挚感谢。

开发过程中还得到了 **Trae Work**（AI 开发环境）的协助——涉及界面原型、调试与文档撰写，特此致谢。

---

## 贡献指南 Contributing

欢迎任何形式的贡献——修 Bug、补文档、写测试、加特性。动手前请先阅读 [CONTRIBUTING.md](./CONTRIBUTING.md) 与 [CODE_OF_CONDUCT.md](./CODE_OF_CONDUCT.md)。

**快速开始**

1. Fork 本项目，从 `main` 拉出一条 `feat/xxx` 或 `fix/xxx` 分支。
2. 需要 **Go 1.26+**。克隆后先 `go build ./...` 确认可编译。
3. 遵循 `gofmt`，为新逻辑补测试并确保 `make test`（`go test -race ./...`）全绿。
4. 使用语义化提交信息（`feat:` / `fix:` / `docs:` / `test:` …），向 `main` 提交 Pull Request。

**Issue 与交流**

- 提问、想法与讨论：[Discussions](https://github.com/54wu/omnivault/discussions)
- Bug 报告与功能请求：[Issues](https://github.com/54wu/omnivault/issues)
- 发现**安全漏洞请勿公开**，按下方 [安全说明](#安全说明-security) 的私有渠道上报。

**发布节奏**

发布由维护者负责：合并到 `main` 后打 `vX.Y.Z` 标签即可（见上「发布（Release）」，CI 会自动产出各平台安装包）。

---

## 安全说明 Security

OmniVault 围绕“逐字段加密 + 零知识 + 密钥永不离机”设计：

- 字段级 **AES-256-GCM**，密码经 **Argon2id** 派生、**HKDF** 展开为子密钥。
- 档案密码从不落盘；解锁期间密钥仅存于内存。
- `secret.key` 与 `vault.db` 分离存放，建议单独加密备份。
- 本地服务默认监听 `127.0.0.1`，令牌有作用域并设有效期。

加密原理与安全模型的完整说明，见上文「[加密原理](#加密原理安全模型)」与「[跨设备同步](#跨设备同步云同步-密钥分离)」。

**报告漏洞**：请**不要**在公开 Issue / 讨论中暴露细节，请通过 GitHub **私有漏洞报告（Security Advisory）** 提交——

[Private vulnerability reporting](https://github.com/54wu/omnivault/security/advisories)

我们会对漏洞及时评估、严格保密，并在修复发布后（经你同意）于致谢中署名。完整政策见 [SECURITY.md](./SECURITY.md)。

---

## 许可证

[MIT](./LICENSE)
