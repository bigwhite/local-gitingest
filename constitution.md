# Project Constitution (项目宪法)
# Version: 1.0

本文件定义了本项目不可动摇的核心开发原则与架构铁律。
所有 AI Agent 在进行技术规划 (`/plan`) 和代码实现 (`/implement`) 时，必须无条件遵循本宪法。

---

## I. SDD 原则 (Spec-Driven Development)

**核心：规约即真理 (Spec is the Source of Truth)。**

1.  **意图先行**: 任何代码变更必须源于 `specs/` 目录下经过批准的规约 (`spec.md`) 和方案 (`plan.md`)。
2.  **禁止私自加戏**: 严禁实现规约中未定义的功能 (Gold-plating)。
3.  **偏差处理**: 如果发现规约有误，必须先修改规约，再修改代码。绝不为了迁就代码而通过“补丁”式修改规约。

---

## II. Go 语言工程哲学 (Go Engineering Philosophy)

**核心：简单、明确、组合 (Simplicity, Explicitness, Composition)。**

1.  **简单性 (Simplicity)**:
    *   **YAGNI (You Ain't Gonna Need It)**: 只实现当前需求，拒绝过度设计和未来式抽象。
    *   **标准库优先**: 除非必要，优先使用 Go 标准库（如 `net/http`, `encoding/json`）。引入第三方库需在 `plan.md` 中进行论证。

2.  **明确性 (Explicitness)**:
    *   **错误处理**: 必须显式处理错误。使用 `fmt.Errorf("%w", err)` 包装错误以保留上下文。严禁使用 `_` 忽略错误。
    *   **无全局状态**: 严禁使用全局变量传递业务状态。依赖关系必须通过构造函数或方法参数显式注入。
    *   **无魔法**: 避免使用 `init()` 函数进行隐式初始化（除了注册模式）。避免滥用 `reflect` 和 `unsafe`。

3.  **可测试性 (Testability)**:
    *   **测试先行 (Test-First)**: 遵循 TDD 流程。在实现功能前，必须先编写失败的测试。
    *   **表格驱动 (Table-Driven)**: 单元测试必须采用表格驱动测试风格。
    *   **接口隔离**: 定义小接口（Interface Segregation），以便于 Mock 和测试。

---

## III. 架构原则 (Architectural Principles)

**核心：关注点分离与单向依赖。**

1.  **分层架构**:
    *   `cmd/`: 仅包含 `main` 入口和依赖注入（Wiring）。不包含业务逻辑。
    *   `internal/`: 包含所有核心业务逻辑。
    *   `pkg/`: (可选) 仅包含旨在被外部项目复用的通用库。
2.  **依赖规则**: 依赖只能从外层指向内层。核心业务逻辑不应依赖于具体的外部实现（如数据库驱动、HTTP 框架）。

---

## IV. 治理 (Governance)

本宪法具有最高优先级。
在生成 `plan.md` 时，必须包含 **Constitution Check** 章节，逐条审查方案是否合宪。
