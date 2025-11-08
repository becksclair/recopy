# Contributing to recopy

Prototype-mode ground rules:

1. **Read first**: Every session starts by reviewing `specs/recopy_plan.md`, the `/research` notes, and `TODO.md`.
2. **Scope ruthlessly**: Implement the thinnest slice that proves value. Prefer stubs and TODOs over speculative plumbing.
3. **Toolchain**: Develop with Go 1.25.3. Install from the official tarball; Ubuntu 24.04 packages lag behind.
4. **Dependencies**: Use stdlib unless a new library clearly saves ≥30 minutes or mitigates risk.
5. **Testing**: Add at most one happy-path smoke test per feature unless fixing a regression.
6. **Docs/Tracking**: Update `TODO.md` (checkboxes) and `/research` (key insights or decisions) as you progress.
7. **Quality gates**: Run `go fmt ./...` and `go build ./cmd/recopy` before sending work. If modules exist, run relevant linters or tests.
