# Go Modules 项目结构与依赖管理总结

## 基本概念回顾与拓展

- **go mod（模块化管理）**  
  每个含 `go.mod` 的目录即为一个模块（Module）。  
  不再依赖 GOPATH，模块可位于任意目录。  
  用于声明模块名及所有依赖和版本（require），可通过 replace 重定向本地模块路径。

- **package（包）**  
  以文件夹为单位组织，同一文件夹下所有 `.go` 文件属于同一个包。  
  一个模块可包含多个包。  
  包之间通过 import 导入，路径为 module 路径 + 子目录名。

- **GOROOT**  
  Go 安装目录（如 `/usr/local/go`），包含编译器、标准库等。  
  不建议修改，仅供内部使用。

- **GOPATH**  
  虽不再作为项目根目录，但仍用于：  
  - 缓存依赖模块：`$GOPATH/pkg/mod/`  
  - 安装工具包：`$GOPATH/bin/`（如 `go install` 安装的 CLI 工具）

---
## 项目结构与依赖行为示例
```
/home/user/myproject/ ← 当前项目根目录（模块根）
├── go.mod            ← 模块定义文件，声明依赖
├── go.sum            ← 依赖版本校验文件
├── main.go           ← 主程序入口，package main
├── service/          ← 自定义包，模块内代码组织
│   └── service.go
├── utils/
│   └── utils.go
└── vendor/ (可选)     ← 第三方依赖本地副本，使用 `go mod vendor` 生成

$GOPATH/pkg/mod/ ← 所有下载的第三方依赖模块统一缓存路径  
├─ github.com/gin-gonic/gin@v1.9.1/  
├─ golang.org/x/net@v0.0.0-xxxxxx/  
└─ ...  
```
- 每个子目录为一个 package  
- 同一个模块下的 package 可直接通过相对路径导入  
- 项目依赖的第三方库会被下载到 `$GOPATH/pkg/mod/`  
- 通过 `import "github.com/xxx/yyy"` 使用第三方包  
- 依赖关系自动写入 `go.mod`，完整校验写入 `go.sum`

---

## 常用命令说明（简洁版）

| 命令         | 功能                                      |
|--------------|-------------------------------------------|
| go mod init  | 初始化 go.mod 文件                        |
| go get       | 获取第三方依赖并写入 go.mod               |
| go build     | 构建项目（自动拉取依赖）                   |
| go run       | 编译并运行（自动拉取依赖）                 |
| go install   | 构建并安装到 `$GOPATH/bin`                 |
| go mod tidy  | 清理未用依赖，补全遗漏依赖                 |

---

## Go 关键路径及项目结构说明

- **项目目录**  
  包含 `go.mod` 的文件夹为模块根。模块内部可按功能划分多个包（package），包是 Go 的基本代码组织单元。项目代码与自定义包统一管理，灵活组织，支持独立版本控制。

- **GOPATH**  
  主要用于存放第三方依赖的模块缓存（`$GOPATH/pkg/mod`）。  
  无论项目位置如何，拉取的依赖会统一缓存在此路径下，多个模块共享缓存，避免重复下载。  
  通过 `go install` 安装的 CLI 工具默认放入 `$GOPATH/bin`。

- **GOROOT**  
  Go 安装目录，包含标准库源码及工具链（如 go、compile、link 等），一般不需要修改。

- **依赖管理机制**  
  `go.mod`：声明模块名、依赖模块及其版本（require），可通过 replace 替换路径。  
  `go.sum`：记录所有依赖模块的哈希校验值，确保依赖未被篡改，支持安全校验。  

---

## Go 项目依赖与构建流程

1. 安装 Go 语言环境（默认安装到 `/usr/local/go`），设置环境变量 `GOROOT` 与 `PATH`  
2. 使用 `go get` 获取依赖模块，支持模块之间相互依赖（包括第三方库）  
3. 使用 `go install` 安装可执行包，产物默认输出到 `$GOPATH/bin`  
4. 使用 `import` 引入模块内外部包，依赖统一由 `go.mod` 管理  
5. 同一 package 内，函数可跨文件直接使用；每个子文件夹即为一个 package  
6. `go.mod` 声明模块名、require 的依赖版本、replace 的本地路径替代项  
7. `go.sum` 记录所有依赖模块的哈希校验值，保障依赖一致与安全  
8. `go build`、`go test`、`go mod tidy` 等命令自动更新 `go.sum` 文件
