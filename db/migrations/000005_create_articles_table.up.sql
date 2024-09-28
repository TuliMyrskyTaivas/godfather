CREATE TABLE IF NOT EXISTS articles (
    id BIGSERIAL PRIMARY KEY,
    source_id INTEGER REFERENCES sources,
    group_id INTEGER REFERENCES groups,
    uri VARCHAR NOT NULL,
    headline VARCHAR NOT NULL,
    metadata JSONB
)