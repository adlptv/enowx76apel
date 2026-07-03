# enowX — AI Gateway & Developer Workspace

**A flexible developer workspace in a single binary.**

enowX bundles an AI gateway, a plugin runtime, a built-in terminal and file browser, and a small community layer into one local app that runs on a single port. Add your own providers, extend it with plugins, and drive it from a clean web UI or the `enx` CLI — no external services required.

---

## What is enowX?

enowX is a self-hosted **developer workspace**. It started as an OpenAI/Anthropic-compatible LLM proxy, and that's still the core, but it has grown into a small, extensible desktop-in-a-tab:

- **AI gateway** — one endpoint that speaks the OpenAI and Anthropic wire formats, normalizes every request internally, and routes it to whichever provider you point it at. Pool many accounts, warm them up, and monitor usage.
- **Plugins** — run your own tools (Go / Node / Python / prebuilt binaries) as sidecars, each with its own UI, and publish/install them from a marketplace.
- **Built-in tools** — a real PTY terminal, a file browser, an API tester, and an agent toolset, all in the same window.
- **Community layer** — sign in with Discord for chat, posts, a plugin marketplace, profiles, and optional cloud sync of your setup across devices.

Everything runs from a **single binary** that serves the API, the management UI, and the local database on one port. State lives in a pure-Go SQLite database, so there are no external dependencies to run it.

## Install

### Install script

Linux and macOS:

```sh
curl -fsSL https://raw.githubusercontent.com/adlptv/enowx76apel/main/install.sh | sh
```

Windows (PowerShell):

```powershell
irm https://raw.githubusercontent.com/adlptv/enowx76apel/main/install.ps1 | iex
```

The installer downloads the latest release for your OS/arch, verifies its checksum, and installs the `enx` binary (to `/usr/local/bin` on Unix, or `%LOCALAPPDATA%\Programs\enx` on Windows). Override the location with `ENX_INSTALL_DIR`, or pin a version with `ENX_VERSION=vX.Y.Z`.

### Download a release binary

Prebuilt binaries for Linux, macOS, and Windows (amd64 and arm64) are attached to every release on the [Releases page](https://github.com/adlptv/enowx76apel/releases). Download the asset for your platform, rename it to `enx` (or `enx.exe` on Windows), make it executable, and put it on your `PATH`.

### Build from source

Requires Go 1.26+ and Node 22+.

```sh
git clone https://github.com/adlptv/enowx76apel.git
cd enowx76apel
make build      # builds the web UI, embeds it, and produces bin/enx
./bin/enx
```

## Quickstart

Start enowX (it runs in the background by default):

```sh
enx
```

It listens on `127.0.0.1:1430`. Open **http://localhost:1430** for the workspace UI. On first launch you set a dashboard password; from there you can add provider accounts, install plugins, and (optionally) sign in with Discord.

Once a provider account is in the pool, send a standard OpenAI request:

```sh
curl http://localhost:1430/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

Streaming works with `"stream": true` and is returned as OpenAI-style server-sent events. Anthropic-format requests are accepted at `/anthropic/v1/messages`.

## The `enx` CLI

The binary is `enx`; running it with no arguments starts the server. It also has a small set of commands for headless / VPS use:

```sh
enx                   # start the server in the background
enx start -f          # run in the foreground (this terminal)
enx stop              # stop the server
enx restart           # restart it
enx status            # is it running? pid + address + version
enx doctor            # environment checklist (runtimes, config, server)
enx update [--check]  # self-update to the latest release
enx tunnel start      # expose the dashboard via a public URL (Cloudflare)
enx tunnel stop       # tear the tunnel down
enx version           # print the version
enx autoclaw-login    # launch AutoClaw auto-login script
```

On a headless box, `enx start` + `enx tunnel start` prints a public URL; opening it prompts you to set the dashboard password on first visit, so remote access is gated by default.

## AI gateway

### Providers

Providers today include OpenAI-compatible upstreams, CodeBuddy (global + CN), Kiro, Codex, Antigravity, Leonardo, Suno, **AutoClaw**, and your own **custom providers** defined in the UI.

### Routing

The `model` field selects the upstream provider:

- A `provider/model` prefix routes explicitly, e.g. `codebuddy/...` or `kiro/...`.
- Known prefixes route automatically (`kiro-...`, `codebuddy-...`, `autoclaw-...`, `glm-...`).
- Anything else falls back to the OpenAI-compatible upstream.

Inbound OpenAI and Anthropic traffic is normalized into one internal request. Outbound, each provider re-encodes only what it needs: providers that already speak OpenAI pass through unchanged; providers with their own formats are normalized per provider.

### Endpoints

- `POST /v1/chat/completions` — OpenAI-compatible chat completions.
- `POST /v1/images/generations` — image generation.
- `POST /anthropic/v1/messages` — Anthropic-compatible messages.
- `GET /health` — health check.
- `GET /api/*` — management API used by the UI.
- `/` — embedded workspace UI.

### AutoClaw Provider

AutoClaw is a specialized upstream for AutoGLM / Z.ai accounts. It provides:

- **Token management** — automatic refresh before expiry
- **Wallet monitoring** — background checking every 5 minutes
- **Exhaustion detection** — auto-skips depleted accounts
- **Model routing** — models: `auto`, `cheap`, `deepseek`, `glm-*`, `autoclaw-*`

Add accounts via:
1. Web UI → Providers → Add provider → AutoClaw
2. CLI: `enx autoclaw-login` launches the Python auto-login script
3. API: `POST /api/accounts/autoclaw/manual` or `/api/accounts/autoclaw/refresh`

See [AUTOCLAW_INTEGRATION.md](./AUTOCLAW_INTEGRATION.md) for full documentation.

## Plugins

Plugins are sidecar apps with their own UI, launched by enowX and reachable at `/plugins/<id>/`. They can be written in Go, Node, Python, served as static files, or shipped as a **prebuilt binary** (`bin` runtime) so users run them with no toolchain installed. Create one from the Plugins app, or install a published plugin from the marketplace. Official plugins can ship prebuilt cross-platform binaries.

## Configuration

Configuration is read from the environment and an optional `config.json` in the runtime directory.

| Variable            | Default     | Description                       |
| ------------------- | ----------- | --------------------------------- |
| `ENOWX_PORT`        | `1430`      | Listen port.                      |
| `ENOWX_HOST`        | `127.0.0.1` | Listen address.                   |
| `ENOWX_RUNTIME_DIR` | `~/.enowx`  | Data directory (SQLite database). |
| `ENOWX_LOG_LEVEL`   | `info`      | Log verbosity.                    |

State is stored locally in a pure-Go SQLite database; no external services are required to run enowX.

## Security model

enowX is local-first. The management surface (terminal, file browser, agent, and the rest of `/api`) is trusted only from the same machine; any request that arrives from elsewhere — e.g. through a tunnel — must carry a valid dashboard session, and a session can only exist once you've set a password. The dashboard password never leaves your machine, and credentials synced to the cloud are end-to-end encrypted (the server only ever sees ciphertext).

## Development

Linux and macOS:

```sh
./dev.sh
```

Windows (PowerShell):

```powershell
./dev.ps1
```

This installs dependencies if needed, then runs the backend and frontend on one port (`http://localhost:1430`) with hot reload and no build step. The Go server proxies the SPA and its hot-reload channel to an internal Vite dev server.

The built-in terminal uses a PTY and currently works on Linux and macOS; on Windows the rest of the app runs but the terminal is disabled.

## Community and support

For issues, feature requests, or discussions, join us on GitHub:

- Repository: https://github.com/adlptv/enowx76apel
- Issues: https://github.com/adlptv/enowx76apel/issues

## License

See [LICENSE](LICENSE).
