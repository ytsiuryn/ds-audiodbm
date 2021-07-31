package entity

// TODO: для описания типов использовать https://github.com/jackc/pgtype

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
)

// Suggestion описывает предложения.
type Suggestion struct {
	EntryID int     `sql:"entry_id" json:"entry_id"`
	ExtDB   string  `sql:"ext_db" json:"ext_db"`
	ExtID   string  `sql:"ext_id" json:"ext_id"`
	Json    []byte  `json:"json"`
	Score   float64 `json:"score"`
}

// Create записывает объект в БД.
func (r *Suggestion) Create(ctx context.Context) error {
	err := InsertFullRec(
		ctx,
		`INSERT INTO audio.suggestion (entry_id,ext_db,ext_id,json,score)`,
		r.EntryID, r.ExtDB, r.ExtID, r.Json, r.Score)
	if err != nil {
		err = errors.Wrapf(
			err, "Suggestion.Create() failed: entry_id=%d, ext_db=%s, ext_id=%s",
			r.EntryID, r.ExtDB, r.ExtID)
	}
	return err
}

// Delete удаляет объект в БД по ID записи.
func (r *Suggestion) Delete(ctx context.Context) error {
	err := Delete(
		ctx,
		"DELETE FROM audio.release WHERE entry_id=$1 AND ext_db=$2 AND ext_id=$3",
		r.EntryID, r.ExtDB, r.ExtID)
	if err != nil {
		err = errors.Wrap(err, "Suggestion.Delete() failed")
	}
	return err
}

// Get ищет объект по значению ключа записи.
func (r *Suggestion) Get(ctx context.Context) error {
	qry := `SELECT json,score FROM audio.release
	WHERE entry_id=$1 AND ext_db=$2 AND ext_id=$3 LIMIT 1`
	row, err := Get(ctx, qry, r, r.EntryID, r.ExtDB, r.ExtID)
	if err != nil {
		return errors.Wrap(err, "Suggestion.Get() select failed")
	}
	err = row.Scan(&r.Json, &r.Score)
	if err != nil && err != pgx.ErrNoRows {
		err = errors.Wrap(err, "Suggestion.Get() scan failed")
	}
	return err
}

// EntrySuggestions возвращает список рекомендованных релизов для данного album_entry.
func EntrySuggestions(ctx context.Context, entryID int) ([]*Suggestion, error) {
	db := ctx.Value("db").(*pgx.Conn)
	if db == nil {
		return nil, errors.Wrap(ErrConnectionInContext, "EntrySuggestions() failed")
	}

	rows, err := db.Query(ctx, "SELECT * FROM audio.suggestion WHERE entry_id=$1", entryID)
	if err != nil {
		return nil, errors.Wrap(err, "EntrySuggestions() select failed")
	}
	defer rows.Close()

	ret := []*Suggestion{}
	for rows.Next() {
		var s Suggestion
		if err = rows.Scan(&s.EntryID, &s.ExtDB, &s.ExtID, &s.Json, &s.Score); err != nil {
			return nil, errors.Wrap(err, "EntrySuggestions() scan failed")
		}
		ret = append(ret, &s)
	}

	return ret, nil
}

func DeleteEntrySuggestions(ctx context.Context, entryID int) error {
	err := Delete(ctx, "DELETE FROM audio.suggestion WHERE entry_id=$1", entryID)
	if err != nil {
		err = errors.Wrap(err, "DeleteEntrySuggestions() failed")
	}
	return err
}
