CREATE TABLE users (
    token        TEXT     PRIMARY KEY,
    display_name TEXT     NOT NULL,
    created_at   DATETIME NOT NULL,
    UNIQUE (token)
);
CREATE INDEX idx_users_display_name ON users (display_name);

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

INSERT INTO users (token, display_name, created_at) VALUES
('admin', 'Administrator', '2025-05-01 00:00:00');

INSERT INTO directories (name, parent_id, owner_token, path, created_at) VALUES
('', 0, 'admin', '/', '2025-05-01 00:00:00');

INSERT INTO directories (name, parent_id, owner_token, path, created_at) VALUES
('mnt', 1, 'admin', '/mnt', '2025-05-01 00:00:00'),
('srv', 1, 'admin', '/srv', '2025-05-01 00:00:00'),
('opt', 1, 'admin', '/opt', '2025-05-01 00:00:00'),
('media', 1, 'admin', '/media', '2025-05-01 00:00:00'),
('usr', 1, 'admin', '/usr', '2025-05-01 00:00:00'),
('lost+found', 1, 'admin', '/lost+found', '2025-05-01 00:00:00'),
('snap', 1, 'admin', '/snap', '2025-05-01 00:00:00'),
('var', 1, 'admin', '/var', '2025-05-01 00:00:00'),
('home', 1, 'admin', '/home', '2025-05-01 00:00:00'),
('root', 1, 'admin', '/root', '2025-05-01 00:00:00'),
('proc', 1, 'admin', '/proc', '2025-05-01 00:00:00'),
('dev', 1, 'admin', '/dev', '2025-05-01 00:00:00'),
('etc', 1, 'admin', '/etc', '2025-05-01 00:00:00'),
('boot', 1, 'admin', '/boot', '2025-05-01 00:00:00'),
('tmp', 1, 'admin', '/tmp', '2025-05-01 00:00:00'),
('run', 1, 'admin', '/run', '2025-05-01 00:00:00'),
('sys', 1, 'admin', '/sys', '2025-05-01 00:00:00');