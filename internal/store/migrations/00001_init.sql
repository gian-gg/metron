-- +goose Up
-- Times are unix nanoseconds.
CREATE TABLE sessions (
    id         TEXT PRIMARY KEY,
    tag        TEXT,
    start_time INTEGER NOT NULL,
    end_time   INTEGER
);

CREATE INDEX sessions_start_time ON sessions (start_time);
CREATE INDEX sessions_tag ON sessions (tag);

CREATE TABLE spans (
    trace_id       TEXT NOT NULL,
    span_id        TEXT NOT NULL,
    parent_span_id TEXT,
    session_id     TEXT NOT NULL REFERENCES sessions (id) ON DELETE CASCADE,
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

CREATE INDEX spans_session_id ON spans (session_id);
CREATE INDEX spans_start_time ON spans (start_time);

-- +goose Down
DROP TABLE spans;
DROP TABLE sessions;
