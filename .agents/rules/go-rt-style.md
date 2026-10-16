# GO-RT Strict Coding Style & Architecture Guardrails

1. **Package Boundaries:**
   - `internal/domain`: Zero external imports. Defines `TestCase`, `Vars`, `Metadata`, `Assertion`, `EvalResult`, and core interfaces (`Provider`, `Plugin`, `Strategy`, `Grader`).
   - `internal/template`: SSTI-safe rendering (`SafeRenderer`). Never evaluate `injectVar` content through template parsing; always inject `injectVar` via boundary replacement after template evaluation.
   - `internal/provider`: Implements `domain.Provider`. Multi-turn strategies (`crescendo`, `hydra`, `goat`) must be implemented as Decorator structs wrapping a target `domain.Provider`.
   - `internal/plugin`: Uses `BaseGenerator` for 20-item batching, deduplication, refusal detection (`IsBasicRefusal`), and `<Modifiers>` block injection.
   - `internal/grader`: Runs fast deterministic refusal check before invoking the Judge LLM with attribution-hardened rubrics.

2. **Code Hygiene:**
   - Keep functions between **10 and 15 lines**.
   - Write table-driven tests (`*_test.go`) for every new package.
   - Always verify compilation and tests via `go test -race ./...`.
