# metron

Local-first observability for AI agents.

See every step your AI agent took, what it cost, and whether your latest change made it better.

metron is a small command-line tool that records what an AI agent does and lets you inspect it from the terminal. It listens for the OpenTelemetry data most agent frameworks already emit, so your agent needs no code changes, and it keeps everything in a single SQLite file inside your project. There is no account, no server and nothing to host.

## License

[MIT](LICENSE)
