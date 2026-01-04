# Claude Code Project Guide

**Project**: Go SDD Template
**Role**: 你是本项目的核心开发者，精通 Go 语言与 SDD 方法论。
**Context**: 本项目基于 **Spec-Driven Development** 模式开发。

---

## 🔗 核心原则导入
> **IMPORTANT**: 在执行任何任务前，必须首先阅读并遵循以下原则：
@./constitution.md

---

## 🛠️ 构建与测试 (Build & Test)
本项目使用 `Makefile` 标准化所有操作。请优先调用 `make` 命令，而非直接运行 `go` 命令。

*   **构建**: `make build` (输出到 `bin/`)
*   **测试**: `make test` (运行所有单元测试)
*   **检查**: `make lint` (运行 golangci-lint 和 go vet)
*   **清理**: `make clean`
*   **Docker**: `make docker-build`

---

## 🛤️ 开发工作流 (Workflow)

### 1. 新功能开发 (SDD 流程)
1.  **需求**: 协助用户在 `specs/` 下创建新目录，并生成 `spec.md`。
2.  **计划**: 基于 `spec.md` 和 `constitution.md`，生成 `plan.md`。
3.  **任务**: 将 `plan.md` 拆解为 `tasks.md`。
4.  **执行**: 严格按照 `tasks.md` 的顺序，遵循 TDD 进行编码。

### 2. 代码提交
*   **Commit Message**: 必须遵循 Conventional Commits。
    *   格式: `<type>(<scope>): <subject>`
    *   示例: `feat(auth): add jwt middleware`
*   **Pre-check**: 在提交前，必须确保 `make lint` 和 `make test` 通过。

---

## 📂 目录结构说明
*   `cmd/`: 应用程序入口。
*   `internal/`: 私有业务逻辑。
*   `specs/`: 需求、方案与任务文档。
*   `.claude/`: AI 配置与自定义指令。
