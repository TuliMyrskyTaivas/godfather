CREATE TABLE IF NOT EXISTS term_groups (
    group_id integer REFERENCES groups,
    term_id integer REFERENCES terms,
    UNIQUE (group_id, term_id)
);