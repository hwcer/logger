# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`github.com/hwcer/logger` — 纯 Go 日志库，零外部依赖，Go 1.24.0。支持控制台、文件（带轮转）、网络三种输出后端，通过插件式 `Output` 接口扩展。

## Build & Test

```bash
# 从项目根目录运行
go build ./...
go test ./... -v

# 单个测试
go test -run TestFile -v
go test -run TestClose -v
```

## Architecture

### 核心链路

```
Debug/Info/Error/...() → Logger.Sprint() → Logger.Write() → Output.Write()
```

- `Logger` 持有 `map[string]Output`，按 name 注册多个输出后端，写入时遍历全部
- 消息低于 `Logger.level` 时在 `Write()` 层直接丢弃
- 包级函数（`logger.Debug()` 等）代理到 `defaultLogger` 全局实例（callDepth=3）

### Output 接口

```go
type Output interface {
    Write(message *Message)
    Close() error
}
```

三个内置实现：

| 实现 | 文件 | 要点 |
|------|------|------|
| `console` | console.go | ANSI 彩色输出（Windows 下关闭颜色），全局单例 `Console` |
| `File` | file.go | 4MB bufio + 独立 goroutine 异步写入；按日期（默认月）或文件大小轮转 |
| `Conn` | conn.go | TCP/UDP 网络日志，自动重连，支持分号分隔多地址 |

### Message 结构

`Message{Path, Time, Level, Stack, Content}` — `Path` 通过 `runtime.Caller(callDepth)` 获取，`Stack` 在 Error/Panic/Fatal 时自动捕获。

### callDepth 约定

直接使用 `*Logger` 实例时 callDepth=2；通过包级函数（`default.go`）调用时 callDepth=3。新增包装层需相应调整。

### 文件轮转

`File` 通过 `fileNameFormatterFunc` 决定文件名和轮转时机——返回新文件名时触发轮转。默认按月（`log.YYYYMM.log`），可通过 `SetFileSize()` 叠加按大小轮转。