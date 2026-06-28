# CLAUDE.md

Before doing anything, read these in order (they live one level up, in `~/V2`):

1. `../HANDOFF.md` — current state, what's done, what's next, gotchas.
2. `../AGENTS.md` — hard rules (very modular, less comments, deps inward to
   `core`, never name reference repos, CGO_ENABLED=0).
3. `../ARCHITECTURE.md` — full design.

This repo (`~/V2/enowx`) is the code; `~/V2` is the workspace (docs, logo.png,
`.multibrain/`). `go build ./...` is the source of truth — ignore stale gopls
"not in workspace" errors. Dev: `./dev.sh` (air + vite hot reload, open
http://localhost:5173).
