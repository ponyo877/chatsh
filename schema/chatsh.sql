CREATE TABLE users (
    id VARCHAR(36) PRIMARY KEY,
    nick VARCHAR(50) NOT NULL,
    created_at TIMESTAMP
);

CREATE TABLE directories (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    parent_id VARCHAR(36),
    owner_id VARCHAR(36) REFERENCES users(id),
    created_at TIMESTAMP
);

CREATE TABLE rooms (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    directory_id VARCHAR(36) REFERENCES directories(id),
    created_at TIMESTAMP
);

CREATE TABLE messages (
    id VARCHAR(36) PRIMARY KEY,
    room_id VARCHAR(36) REFERENCES rooms(id),
    user_id VARCHAR(36) REFERENCES users(id),
    content TEXT NOT NULL,
    created_at TIMESTAMP,
    FULLTEXT INDEX content_idx (content)
);