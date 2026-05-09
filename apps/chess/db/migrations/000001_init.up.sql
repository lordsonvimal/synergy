CREATE TABLE sessions (
    id           TEXT    PRIMARY KEY,
    session_type TEXT    NOT NULL CHECK(session_type IN ('play', 'class')),
    title        TEXT,
    status       TEXT    NOT NULL DEFAULT 'pending'
                         CHECK(status IN ('pending', 'active', 'completed', 'abandoned')),
    created_at   INTEGER NOT NULL,
    started_at   INTEGER,
    ended_at     INTEGER
);

CREATE TABLE participants (
    id           TEXT    PRIMARY KEY,
    session_id   TEXT    NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    role         TEXT    NOT NULL CHECK(role IN ('coach', 'white', 'black', 'student', 'spectator')),
    display_name TEXT    NOT NULL,
    token        TEXT    NOT NULL UNIQUE,
    joined_at    INTEGER NOT NULL,
    left_at      INTEGER
);

CREATE TABLE games (
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
                                      'draw_stalemate','draw_fifty_move','draw_agreement',
                                      'draw_threefold','draw_insufficient','abandoned','disconnected'
                                  )),
    winner               TEXT    CHECK(winner IN ('white', 'black')),
    created_at           INTEGER NOT NULL,
    ended_at             INTEGER
);

CREATE TABLE moves (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id       TEXT    NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    session_id    TEXT    NOT NULL,
    seq           INTEGER NOT NULL,
    uci           TEXT    NOT NULL,
    san           TEXT    NOT NULL,
    fen           TEXT    NOT NULL,
    move_number   INTEGER NOT NULL,
    color         TEXT    NOT NULL CHECK(color IN ('white', 'black')),
    w_rem_ns      INTEGER NOT NULL,
    b_rem_ns      INTEGER NOT NULL,
    lag_comp_ns   INTEGER NOT NULL DEFAULT 0,
    think_time_ns INTEGER NOT NULL,
    played_at     INTEGER NOT NULL,
    UNIQUE(game_id, seq)
);

CREATE TABLE game_events (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id     TEXT    NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    session_id  TEXT    NOT NULL,
    event_type  TEXT    NOT NULL,
    payload     TEXT,
    occurred_at INTEGER NOT NULL
);

CREATE INDEX idx_moves_game_id       ON moves(game_id);
CREATE INDEX idx_game_events_game_id ON game_events(game_id);

CREATE TABLE chat_messages (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id     TEXT    NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    participant_id TEXT    NOT NULL REFERENCES participants(id),
    body           TEXT    NOT NULL,
    created_at     INTEGER NOT NULL
);
