<div align="center">

# metron

**Lightweight, harness-agnostic AI agent observability and evaluation CLI.**

See every step your agent took, how many tokens it used, and whether your latest change made it better.

[Quick start](#quick-start) ·
[Commands](#commands) ·
[How it works](#how-it-works) ·
[Documentation](docs/architecture.md)

</div>

---

```
$ metron compare v1 v2
comparing v1 (1 session) with v2 (1 session)

METRIC                  v1     v2     CHANGE
traces                  2      3      —
duration (mean)         3.2s   4s     +26%
traces with errors      0%     0%     —
input tokens / trace    324    917    +183%
output tokens / trace   50     274    +448%
```

## Why metron

Changing a prompt, a model or a tool and wondering whether the agent got better is guesswork without a record of what it did. metron is that record, with nothing to host.

- **Lightweight.** A single binary and one SQLite file. There is no account, server or dashboard to run.
- **Harness-agnostic.** It speaks OpenTelemetry (OTLP over HTTP and gRPC) on the standard ports, so any framework or SDK that exports traces works with no metron-specific code.
- **Observability.** Every trace is a timeline of the model calls and tool calls the agent made, in order, with token counts.
- **Evaluation.** Tag a run, change something, tag the next one, and compare duration, errors, calls and tokens between the two.
- **Local by design.** Everything stays in your project. Nothing leaves your machine.

## Requirements

- Go 1.26 or later, to install.
- An app that exports OpenTelemetry **traces** over OTLP to the default endpoint, `localhost:4318` (HTTP) or `localhost:4317` (gRPC).
- For models and token counts, spans that carry the standard [GenAI attributes](https://opentelemetry.io/docs/specs/semconv/gen-ai/): `gen_ai.request.model`, `gen_ai.usage.input_tokens` and `gen_ai.usage.output_tokens`. Spans without them are still recorded, with those columns empty.

## Install

```bash
go install github.com/gian-gg/metron@latest
```

## Quick start

**1. Start the listener** in your project and tag the version you are about to test:

```bash
metron serve --tag v1
```

**2. Run your agent.** Stop `serve` with Ctrl+C when it is done.

**3. Change something**, start the listener again with a new tag, and run the agent again:

```bash
metron serve --tag v2
```

**4. Compare the two:**

```bash
metron compare v1 v2
```

**5. Look at one version step by step:**

```
$ metron show v1
tag v1 · 1 session · done

trace 7bd8ea0e · 2.5s · 3 spans
OFFSET   KIND    NAME                 MODEL           DURATION   TOKENS            STATUS
+0s      other   agent.run            —               2.5s       —
+100ms   model   chat example-model   example-model   1.1s       219 in · 10 out
+1.2s    tool    tool search          —               1.3s       —
```

`show` lists every trace under the tag; one is shown here.

## Commands

| Command | What it does |
|---|---|
| `metron serve [--tag T]` | Listen for traces and record them under a tag |
| `metron ls` | List what was recorded, grouped by tag |
| `metron show [tag or session]` | Show the spans of a tag or session as a timeline |
| `metron compare <a> <b>` | Compare two tags or sessions side by side |
| `metron completion <shell>` | Generate shell tab-completion |

Run `metron <command> --help` for the flags of each.

## How it works

```
your agent ──OTLP──▶ metron serve ──▶ .metron/metron.db ──▶ ls · show · compare
```

`serve` accepts trace exports, reads the kind, model and token counts out of each span's GenAI attributes, and stores the span. Every run of `serve` is a session; sessions that share a tag are treated as one, so a tag names a version you can compare. The database is created in the directory where `serve` runs.

## Data and privacy

metron makes no network calls. By default the listener binds to `127.0.0.1` only and has no authentication, so it is reachable from your machine and nothing else. Passing `--listen 0.0.0.0` exposes it to your network.

metron stores every attribute your app puts on a span. If your app records prompts or other sensitive text there, it ends up in `.metron/metron.db`, so keep that folder out of version control:

```bash
echo ".metron/" >> .gitignore
```

## Documentation

- [Architecture](docs/architecture.md): components, storage, how each command works, and the decisions behind them.
- [Conventions](docs/conventions.md): code layout and the rules for working in this repo.

## License

[MIT](LICENSE)
