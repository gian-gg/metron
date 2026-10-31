# Conventions

## Layout

```
main.go             entry point, hands off to cmd
cmd/                root command, --db flag, registers the commands
  ls/ show/ compare/ serve/   one package per command
  internal/session/ what the read commands share: opening the database
internal/           all logic, private to metron
  store/            SQLite: queries and methods
    db/ migrations/ queries/  generated code, schema, SQL
  receiver/         OTLP receiver that saves spans
    spans/          OTLP export to span rows
  normalize/        kind, model and tokens from GenAI attributes
  compare/          metrics for two sets of traces
  render/           terminal output, one package per view
    internal/ui/    styles, table layout and formatting the views share
  testutil/         helpers shared by the command tests
```

- No `src/`, `utils/`, `helpers/` or `common/`. Name packages by what they do.
- A package per command and per view; split a package by file first, and into a package only when the parts have a different job or a shared piece forces it.
- Code shared by sibling packages lives in an `internal/` folder beside them, so it is not public surface.
- Methods of one type stay in one package, so `store` is split by file.

## Modularity

- `cmd` stays thin: no SQL or formatting in commands; they call `store` and `render`.
- One package per command in `cmd/`, built by its `New()` function. No package-level state, except build-time values like `Version`.
- Pass dependencies in instead of reaching for them: take an `io.Writer` and `now`, not `os.Stdout` and `time.Now()`.
- Private by default: export only what another package needs.
- `internal/` packages never import `cmd`.
- Define interfaces in the package that uses them (e.g. `receiver.SpanWriter`), not the one that implements them.
- One concern per file; shared helpers go in a named file inside the package (e.g. `render/internal/ui/format.go`).

## Code

- Wrap errors with context: `fmt.Errorf("list sessions: %w", err)`.
- Comments are one line saying what something is or does. Most code needs none.
- Every package has a one-line package comment (`doc.go` or `main.go`).
- Nullable values are pointers (`*string`), like optional fields in TypeScript.
- Times are stored as unix nanoseconds.
- Named constants instead of magic numbers.
- Pass `cmd.Context()` down to every database call.
- Functions that update a value return a new copy instead of changing it in place.

## Database

- Schema lives in `internal/store/migrations/`. Add a new numbered file for every change; never edit one that has shipped.
- Queries live in `internal/store/queries/`. Run `sqlc generate` after changing either folder.
- Never edit `internal/store/db/`; it is generated.
- For optional filters, write the column comparison first so sqlc infers the type: `WHERE tag = sqlc.narg(tag) OR sqlc.narg(tag) IS NULL`.
- Read-only commands never create `.metron/`.

## Output

- Colors and styles live in `internal/render/style.go`.
- Output is plain text when piped; no trailing spaces.
- Commands write to `cmd.OutOrStdout()` so tests can capture output.

## Tests

- Tests live next to the code they test, in `_test.go` files. Command tests share `internal/testutil`.
- Prefer table-driven tests.
- Use `package x_test` to test through the public API, `package x` to test private helpers.
- Use `t.TempDir()` for databases; never touch `.metron/` in tests.

## Tooling

- `golangci-lint fmt` formats, `golangci-lint run` lints and type checks.
- Git hooks (lefthook): format and lint on commit, `go test -race ./...` on push.
- Tools install to `~/go/bin`, which must be on `PATH`.

## Git

- Commit messages follow Conventional Commits: `feat(store): filter sessions by tag`.
- Commit `go.sum` and generated code in `internal/store/db/`.
