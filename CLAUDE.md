# CLAUDE.md

Primary guidance for coding agents in this repo lives in `agents.md`. Read it first — the guardrails, project structure, testing discipline, and commit conventions apply to every change.

This file only adds details specific to Claude Code sessions.

## Knowledge Graph (graphify)

A local knowledge graph of the repo is maintained under `graphify-out/`:

- `graphify-out/graph.json` — nodes + edges (each edge tagged `EXTRACTED`, `INFERRED`, or `AMBIGUOUS`)
- `graphify-out/graph.html` — interactive visualization
- `graphify-out/GRAPH_REPORT.md` — god nodes, cross-community bridges, suggested questions

All graphify artifacts (`graphify-out/`, `.graphify_*`, `raw/`) are gitignored.

Post-commit and post-checkout git hooks rebuild the code graph automatically. Doc/ADR changes flag `graphify-out/.needs_update` and require `/graphify --update` (semantic re-extraction).

Before editing unfamiliar areas, orient with:

- `/graphify query "<question>"` — grounded Q&A with source citations, follows only actual edges
- `/graphify path "A" "B"` — shortest connection between two concepts
- `/graphify explain "<node>"` — node + its neighborhood in plain language

Never trust `INFERRED` or `AMBIGUOUS` edges without checking the source file.

## When a rebuild is needed

- Code-only changes → the post-commit hook handles it, no action required.
- Changes to `adrs/`, `docs/`, `README.md`, or `agents.md` → run `/graphify --update` to refresh semantic extraction.
