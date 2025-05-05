CREATE TABLE users (
    token        TEXT     PRIMARY KEY,
    display_name TEXT     NOT NULL,
    created_at   DATETIME NOT NULL,
    UNIQUE (token)
);
CREATE INDEX idx_users_nick ON users (nick);

CREATE TABLE directories (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    name        TEXT     NOT NULL,
    parent_id   INTEGER  NOT NULL REFERENCES directories(id),
    owner_token TEXT     NOT NULL REFERENCES users(token),
    path        TEXT     NOT NULL,
    created_at  DATETIME NOT NULL,
    UNIQUE (parent_id, name),
    UNIQUE (path)
);

CREATE TABLE rooms (
    id           INTEGER  PRIMARY KEY AUTOINCREMENT,
    name         TEXT     NOT NULL,
    directory_id INTEGER  NOT NULL REFERENCES directories(id),
    owner_token  TEXT     NOT NULL REFERENCES users(token),
    path         TEXT     NOT NULL,
    created_at   DATETIME NOT NULL,
    UNIQUE (directory_id, name),
    UNIQUE (path)
);

CREATE TABLE messages (
    id           INTEGER  PRIMARY KEY AUTOINCREMENT,
    room_id      INTEGER  NOT NULL REFERENCES rooms(id),
    display_name TEXT     NOT NULL,
    content      TEXT     NOT NULL,
    created_at   DATETIME NOT NULL
);
CREATE INDEX idx_messages_room_created ON messages (room_id, created_at DESC);
