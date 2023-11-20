CREATE KEYSPACE IF NOT EXISTS paste WITH REPLICATION = {'class': 'SimpleStrategy', 'replication_factor': 1 };

CREATE TABLE IF NOT EXISTS paste.KeyRanges (
    id uuid,
    prefix text,
    used boolean,
    primary key(id)
);

CREATE TABLE IF NOT EXISTS paste.User (
    id uuid PRIMARY KEY,
    username text,
    password_hash text,
    created_at timestamp,
    updated_at timestamp
);

CREATE TABLE IF NOT EXISTS paste.Paste (
    id uuid PRIMARY KEY,
    ptype int,
    ptext text,
    s3_url text,
    user_id uuid,
    created_at timestamp,
    updated_at timestamp
);

CREATE TABLE IF NOT EXISTS paste.PasteKeys (
    key text,
    paste_id uuid,
    expires_at timestamp,
    PRIMARY KEY ((key), paste_id)
) WITH CLUSTERING ORDER BY (paste_id DESC);

CONSISTENCY ONE;
