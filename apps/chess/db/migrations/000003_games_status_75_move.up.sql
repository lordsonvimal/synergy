-- Add 'draw_75_move' to games.status CHECK constraint.
-- SQLite cannot ALTER a CHECK constraint in place; rebuild the table.
-- This migration runs inside a transaction, so use defer_foreign_keys
-- (foreign_keys PRAGMA cannot be toggled inside a transaction).
PRAGMA defer_foreign_keys = ON;

CREATE TABLE games_new (
    id                   TEXT    PRIMARY KEY,
    session_id           TEXT    NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    white_participant_id TEXT    REFERENCES participants(id),
    black_participant_id TEXT    REFERENCES participants(id),
    time_control_ns      INTEGER,
    increment_ns         INTEGER,
    variant              TEXT    NOT NULL DEFAULT 'standard',
    initial_fen          TEXT,
    status               TEXT    NOT NULL DEFAULT 'ongoing'
                                  CHECK(status IN (
                                      'ongoing','checkmate','resigned','clock_flagged',
                                      'draw_stalemate','draw_fifty_move','draw_75_move','draw_agreement',
                                      'draw_threefold','draw_insufficient','abandoned','disconnected'
                                  )),
    winner               TEXT    CHECK(winner IN ('white', 'black')),
    created_at           INTEGER NOT NULL,
    ended_at             INTEGER
);

INSERT INTO games_new SELECT * FROM games;
DROP TABLE games;
ALTER TABLE games_new RENAME TO games;
