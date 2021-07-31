-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';

CREATE TYPE audio.ext_db AS ENUM (
    'rutracker',
    'discogs',
    'musicbrainz'
);

CREATE TYPE audio.entity AS ENUM (
	'actor',
    'work',
    'composition',
    'record',
    'album_entry',
    'suggestion',
    'track',
    'label',
    'disc'
);

CREATE TYPE audio.entry_status AS ENUM (
    'without_mandatory_tags',
    'with_mandatory_tags',
    'finalyzed'
);

CREATE TYPE audio.pict_type AS ENUM (
	'png_icon',
	'other_icon',
	'cover_front',
	'cover_back',
	'leaflet',
	'media',
	'lad_artist',
	'artist',
	'conductor',
	'orchestra',
	'composer',
	'lyricist',
	'recording_location',
	'during_recording',
	'during_performance',
	'movie_screen',
	'bright_color_fish',
	'illustration',
	'artist_logotype',
	'publisher_logotype'
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
DROP TYPE audio.pict_type;
DROP TYPE audio.entry_status;
DROP TYPE audio.entity;
DROP TYPE audio.ext_db;
-- +goose StatementEnd
