CREATE TABLE users (
    id         VARCHAR(36)  PRIMARY KEY,
    nick       VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    INDEX (nick)
);

CREATE TABLE directories (
    id         VARCHAR(36)  PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,
    parent_id  VARCHAR(36)  NOT NULL REFERENCES directories(id),
    owner_id   VARCHAR(36)  NOT NULL REFERENCES users(id),
    created_at TIMESTAMP    NOT NULL,
    UNIQUE (parent_id, name)
);

CREATE TABLE rooms (
    id           VARCHAR(36)  PRIMARY KEY,
    name         VARCHAR(100) NOT NULL,
    directory_id VARCHAR(36)  NOT NULL REFERENCES directories(id),
    created_at   TIMESTAMP    NOT NULL,
    UNIQUE (directory_id, name)
);

CREATE TABLE messages (
    id         VARCHAR(36) PRIMARY KEY,
    room_id    VARCHAR(36) NOT NULL REFERENCES rooms(id),
    user_id    VARCHAR(36) NOT NULL REFERENCES users(id),
    content    TEXT        NOT NULL,
    created_at TIMESTAMP   NOT NULL,
    INDEX (room_id, created_at DESC)
);