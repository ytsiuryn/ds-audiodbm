-- +goose Up
-- +goose StatementBegin
SELECT
  'up SQL query';

CREATE TABLE audio.album_entry (
    id SERIAL PRIMARY KEY,
    path VARCHAR(255) UNIQUE NOT NULL,
	json JSONB,
    status audio.entry_status,
	last_modified TIMESTAMP NOT NULL
);
CREATE INDEX idx_albumentry_path ON audio.album_entry (path);
CREATE INDEX idx_albumentry_release ON audio.album_entry USING gin (json);

CREATE TABLE audio.suggestion (
    entry_id INTEGER REFERENCES audio.album_entry (id),
	ext_db audio.ext_db NOT NULL,
	ext_id VARCHAR(32) NOT NULL,
	json JSONB,
	score REAL,
	PRIMARY KEY (entry_id, ext_db, ext_id)
);
CREATE INDEX idx_suggestion_body ON audio.suggestion USING gin (json);

CREATE TABLE audio.bad_suggestion (
	entry_id INTEGER REFERENCES audio.album_entry (id),
	ext_db audio.ext_db NOT NULL,
	ext_id VARCHAR(32) NOT NULL,
	PRIMARY KEY (entry_id, ext_db, ext_id)
);

CREATE TABLE audio.picture (
	entity_type   audio.entity NOT NULL,
	entity_id INTEGER NOT NULL,
	pict_type audio.pict_type NOT NULL,
	width    SMALLINT,
	height   SMALLINT,
	mime   VARCHAR(20),
	notes    TEXT,
	data     BYTEA,
	PRIMARY KEY (entity_type, entity_id, pict_type)
);

CREATE TABLE audio.actor (
	entry_id INTEGER REFERENCES audio.album_entry (id),
	name VARCHAR(100) NOT NULL,
	ids    VARCHAR(65)[][2] NOT NULL,
	entity_mask INTEGER,
	PRIMARY KEY (entry_id, name)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT
  'down SQL query';

-- DROP TABLE audio.unprocessed;
DROP TABLE audio.actor;
DROP TABLE audio.picture;
DROP TABLE audio.bad_suggestion;
DROP TABLE audio.suggestion;
DROP TABLE audio.album_entry;
-- +goose StatementEnd
