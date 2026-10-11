# Conventions

## Layout

```
main.go        entry point, hands off to cmd
cmd/           CLI commands: parse flags, call into internal/
internal/      all logic, private to metron
  store/       SQLite: migrations, queries, generated code
  runner/      starting and supervising the agent process
  receiver/    OTLP receiver that saves spans
  render/      terminal output
```

- No `src/`, `utils/`, `helpers/` or `common/`. Name packages by what they do.
- Split large packages by file, not into new packages.

## Modularity

- `cmd` stays thin: no SQL or formatting in commands; they call `store` and `render`.
- One command per file in `cmd/`, built by a `newXxxCmd()` function. No package-level state, except build-time values like `Version`.
- Pass dependencies in instead of reaching for them: take an `io.Writer` and `now`, not `os.Stdout` and `time.Now()`.
- Private by default: export only what another package needs.
- `internal/` packages never import `cmd`.
- Define interfaces in the package that uses them (e.g. `receiver.SpanWriter`), not the one that implements them.
- One concern per file; shared helpers go in a named file inside the package (e.g. `render/format.go`).

## Code

- Wrap errors with context: `fmt.Errorf("list runs: %w", err)`.
- Comments are one line saying what something is or does. Most code needs none.
- Every package has a one-line package comment (`doc.go` or `main.go`).
- Nullable values are pointers (`*string`), like optional fields in TypeScript.
- Times are stored as unix nanoseconds.
- Named constants instead of magic numbers.
- Pass `cmd.Context()` down to every database call.
- A command that needs a specific exit code returns `*cmd.ExitError`; `main` exits with its code.
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

- Tests live next to the code they test, in `_test.go` files.
- Prefer table-driven tests.
- Use `package x_test` to test through the public API, `package x` to test private helpers.
- Use `t.TempDir()` for databases; never touch `.metron/` in tests.

## Tooling

- `golangci-lint fmt` formats, `golangci-lint run` lints and type checks.
- Git hooks (lefthook): format and lint on commit, `go test -race ./...` on push.
- Tools install to `~/go/bin`, which must be on `PATH`.

## Git

- Commit messages follow Conventional Commits: `feat(store): filter runs by tag`.
- Commit `go.sum` and generated code in `internal/store/db/`.
