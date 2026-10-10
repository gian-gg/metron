-- +goose Up
-- Times are unix nanoseconds.
CREATE TABLE runs (
    id         TEXT PRIMARY KEY,
    tag        TEXT,
    case_name  TEXT,
    command    TEXT NOT NULL,
    start_time INTEGER NOT NULL,
    end_time   INTEGER,
    exit_code  INTEGER
);

CREATE INDEX runs_start_time ON runs (start_time);
CREATE INDEX runs_tag ON runs (tag);

CREATE TABLE spans (
    trace_id       TEXT NOT NULL,
    span_id        TEXT NOT NULL,
    parent_span_id TEXT,
    run_id         TEXT REFERENCES runs (id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    kind           TEXT NOT NULL DEFAULT 'other',
    status         TEXT NOT NULL DEFAULT 'unset',
    start_time     INTEGER NOT NULL,
    end_time       INTEGER NOT NULL,
    model          TEXT,
    input_tokens   INTEGER,
    output_tokens  INTEGER,
    cost_usd       REAL,
    attributes     TEXT NOT NULL DEFAULT '{}',
    PRIMARY KEY (trace_id, span_id)
);

CREATE INDEX spans_run_id ON spans (run_id);
CREATE INDEX spans_start_time ON spans (start_time);

-- +goose Down
DROP TABLE spans;
DROP TABLE runs;
