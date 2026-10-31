# metron

Local-first observability for AI agents.

See every step your AI agent took, what it cost, and whether your latest change made it better.

metron is a small command-line tool that records what an AI agent does and lets you inspect it from the terminal. It listens for the OpenTelemetry data most agent frameworks already emit, so your agent needs no code changes, and it keeps everything in a single SQLite file inside your project. There is no account, no server and nothing to host.

## Usage

Start the listener in your project. Tag the session so versions can be compared later:

```bash
metron serve --tag v1
```

metron listens on the standard OTLP ports, `127.0.0.1:4318` (HTTP) and `127.0.0.1:4317` (gRPC). An app that exports OpenTelemetry traces with its default settings reaches it with no configuration.

Stop it with Ctrl+C, start it again with another tag for the next version, and list what was recorded:

```bash
metron ls
metron ls --tag v1
```

Sessions that share a tag are grouped, so starting `serve --tag v1` twice adds to `v1`. `metron ls --sessions` lists every session on its own.

See what an agent did, step by step. Pass a tag to see all of its sessions, or a session ID from `ls --sessions` to see one. With no argument it shows the most recent:

```bash
metron show
metron show v1
metron show 4d27
```

Compare two tags, or two sessions, side by side. The change column is the second relative to the first:

```bash
metron compare v1 v2
```

It reports duration, errors, model and tool calls, and tokens per trace. Check the `traces` row first: a few traces on either side is a weak comparison.

Spans are kept in `.metron/metron.db` in the directory where `serve` runs. Use `--db` to put it elsewhere, and `--port`, `--grpc-port` or `--listen` if the defaults do not suit. `--listen 0.0.0.0` exposes the unauthenticated listener to your network.

## License

[MIT](LICENSE)
