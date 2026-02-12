# gocloc

`gocloc` 是一个基于 **FSM（有限状态机）** 的代码度量 CLI 工具，目标是统计：

- `total`（总行数）
- `code`（代码行）
- `comment`（注释行）
- `blank`（空白行）

与传统正则实现不同，`gocloc` 通过语言级状态机处理复杂场景，例如：

- `x := 1 // comment` 同时计为 code + comment
- `"hello // world"` 字符串内注释符号不误判
- 支持嵌套块注释（如 Rust、SQL）
- 支持 `=begin/=end` 块注释（Ruby）

## 运行环境

- Go `1.25`

## 安装与构建

```bash
go mod tidy
go build -o gocloc .
```

## 命令说明

### 1) `gocloc version`

显示版本号。

```bash
gocloc version
```

### 2) `gocloc language`

展示已支持语言与后缀。

```bash
gocloc language
```

### 3) `gocloc scan [path]`

扫描目录或单文件，统计已注册语言文件。

```bash
# 默认 table 输出
gocloc scan .

# JSON 输出 + 导出到指定路径
gocloc scan . --format json --output result.json

# 调整并发 worker 数
gocloc scan . --workers 8
```

参数：

- `--format`：`table`（默认）或 `json`
- `--output`：JSON 导出路径，默认 `output.json`
- `--workers`：并发 worker 数，默认 `CPU 核心数`

## 当前支持语言

- Go: `.go`
- JavaScript: `.js`, `.mjs`, `.cjs`
- TypeScript: `.ts`, `.tsx`
- Python: `.py`
- Rust: `.rs`
- Ruby: `.rb`
- Java: `.java`
- C/C++: `.c`, `.cc`, `.cpp`, `.cxx`, `.h`, `.hh`, `.hpp`, `.hxx`
- SQL: `.sql`

## 架构说明

- `cmd/`：Cobra 命令层（`version`、`language`、`scan`）
- `internal/scanner/`：并发调度与扫描聚合
- `internal/languages/`：每个语言一个独立 FSM 引擎文件
- `internal/report/`：table/json 输出与 JSON 文件导出
- `internal/model/`：统一数据模型

## 设计要点

1. **并发扫描**：目录遍历后把任务分发给 worker 并发处理。
2. **流式读取**：每个文件使用 `bufio.Reader` 按行读取，适配大文件。
3. **行级双计数**：同一行可同时计入 `code` 与 `comment`。
4. **语言隔离**：每种语言独立 FSM，避免通用正则误判。
