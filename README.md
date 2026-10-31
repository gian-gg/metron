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

See what an agent did in a session, step by step. With no argument it shows the latest session, or pass an ID or ID prefix from `ls`:

```bash
metron show
metron show 4d27
```

Spans are kept in `.metron/metron.db` in the directory where `serve` runs. Use `--db` to put it elsewhere, and `--port`, `--grpc-port` or `--listen` if the defaults do not suit. `--listen 0.0.0.0` exposes the unauthenticated listener to your network.

## License

[MIT](LICENSE)
