CREATE KEYSPACE IF NOT EXISTS paste WITH REPLICATION = {'class': 'SimpleStrategy', 'replication_factor': 1 };

CREATE TABLE IF NOT EXISTS paste.key_ranges (
    id uuid,
    prefix text,
    used boolean,
    primary key(id)
);

CREATE TABLE IF NOT EXISTS paste.User (
    id uuid,
    username text,
    password_hash text,
    created_at timestamp,
    updated_at timestamp,
    primary key(id)
);

CREATE TABLE IF NOT EXISTS paste.Paste (
    id uuid,
    ptype int,
    ptext text,
    s3_url text,
    user_id uuid,
    created_at timestamp,
    updated_at timestamp,
    primary key (id)
);

CONSISTENCY ONE;
