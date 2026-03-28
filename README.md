# Rillan

Rillan is a local OpenAI-compatible proxy daemon written in Go. Milestone one delivered the daemon shell, `rillan serve`, `rillan init`, config loading with env overrides, health/readiness endpoints, a validated `/v1/chat/completions` route, and one real outbound provider path through OpenAI-compatible upstreams. Milestone two adds a deterministic local indexing substrate with `rillan index` and `rillan status`, embedded SQLite metadata storage, text/code chunking, and local vector records.

## Current Scope

- local API boundary on `127.0.0.1:8420`
- Cobra CLI with `rillan serve` and `rillan init`
- YAML-backed runtime config plus env overrides
- `GET /healthz`
- `GET /readyz`
- `POST /v1/chat/completions`
- structured request logging with request IDs
- OpenAI-compatible upstream passthrough as the first real provider path
- one explicit local index root
- manual `rillan index` rebuilds
- `rillan status` for local corpus state
- embedded SQLite metadata, chunk, and vector storage

Still out of scope: retrieval-time context injection, MCP, background file watching, local model orchestration, provider failover, and audit ledger work.

## Provider Policy

- OpenAI-compatible upstreams are the default path in milestone one.
- Anthropic is intentionally non-default and discouraged for now.
- Anthropic is represented in config only as an explicit future-facing seam; it is not implemented as a runtime provider in milestone one.
- No unofficial access paths, shared credentials, scraped sessions, or browser-cookie flows are supported.

## Configuration

Default config path:

- macOS: `~/Library/Application Support/rillan/config.yaml`
- Linux: `${XDG_CONFIG_HOME:-~/.config}/rillan/config.yaml`

Related local directories:

- data: macOS `~/Library/Application Support/rillan/data`, Linux `${XDG_DATA_HOME:-~/.local/share}/rillan`
- logs: macOS `~/Library/Logs/rillan`, Linux `${XDG_STATE_HOME:-~/.local/state}/rillan/logs`

Filesystem ownership for milestone two:

- config owns runtime settings only; `rillan init` writes a starter config but does not build the index
- `index.root` names one local directory to scan; if the value is relative, it is resolved relative to the config file location
- the data directory owns embedded runtime state, currently `index/index.db` for index runs, documents, chunks, and vectors
- the logs directory is reserved for daemon log output and stays separate from config and data

Environment overrides:

- `RILLAN_SERVER_HOST`
- `RILLAN_SERVER_PORT`
- `RILLAN_SERVER_LOG_LEVEL`
- `RILLAN_PROVIDER_TYPE`
- `RILLAN_OPENAI_BASE_URL`
- `RILLAN_OPENAI_API_KEY` or `OPENAI_API_KEY`
- `RILLAN_ANTHROPIC_ENABLED`
- `RILLAN_ANTHROPIC_BASE_URL`
- `RILLAN_ANTHROPIC_API_KEY` or `ANTHROPIC_API_KEY`
- `RILLAN_LOCAL_MODEL_BASE_URL`
- `RILLAN_INDEX_ROOT`
- `RILLAN_INDEX_INCLUDES`
- `RILLAN_INDEX_EXCLUDES`
- `RILLAN_INDEX_CHUNK_SIZE_LINES`
- `RILLAN_VECTOR_STORE_MODE`

Use `configs/rillan.example.yaml` as the checked-in reference config.

## Quickstart

Initialize a starter config:

```bash
go run ./cmd/rillan init
```

Set your OpenAI API key:

```bash
export OPENAI_API_KEY="your-key"
```

Start the daemon:

```bash
go run ./cmd/rillan serve
```

Build the local index:

```bash
go run ./cmd/rillan index --config ./testdata/configs/index-smoke.yaml
go run ./cmd/rillan status --config ./testdata/configs/index-smoke.yaml
```

`rillan status` reports the configured root, current counts, the last successful index time, the last indexing error if any, and the SQLite database path.

Check health:

```bash
curl http://127.0.0.1:8420/healthz
curl http://127.0.0.1:8420/readyz
```

Send a chat completion request:

```bash
curl -X POST http://127.0.0.1:8420/v1/chat/completions \
  -H 'content-type: application/json' \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"ping"}]}'
```

## Development

Run tests:

```bash
go test ./...
go test -cover ./...
```

Build the binary:

```bash
go build ./...
```

## Repository Decisions

Repo-local architecture decisions are documented in `adrs/` while the codebase is still small and changing quickly.
