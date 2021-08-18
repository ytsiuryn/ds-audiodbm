package entity

// TODO: для описания типов использовать https://github.com/jackc/pgtype

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
)

type BadSuggestion struct {
	EntryID int    `sql:"entry_id"`
	ExtDB   string `sql:"ext_db"`
	ExtID   string `sql:"ext_id"`
}

// Create записывает объект в БД.
func (bs *BadSuggestion) Create(ctx context.Context) error {
	err := InsertFullRec(
		ctx,
		`INSERT INTO audio.bad_suggestion (entry_id, ext_db, ext_id) VALUES($1,$2,$3)`,
		bs.EntryID, bs.ExtDB, bs.ExtID)
	if err != nil {
		err = errors.Wrapf(
			err, "BadSuggestion.Create() failed: entry_id=%d, ext_db=%s, ext_id=%s",
			bs.EntryID, bs.ExtDB, bs.ExtID)
	}
	return err
}

// Delete удаляет объект в БД по ID записи.
func (bs *BadSuggestion) Delete(ctx context.Context) error {
	err := Delete(
		ctx,
		"DELETE FROM audio.bad_suggestion WHERE entry_id=$1 AND ext_db=$2 AND ext_id=$3",
		bs.EntryID, bs.ExtDB, bs.ExtID)
	if err != nil && err != pgx.ErrNoRows {
		err = errors.Wrap(err, "BadSuggestion.Delete() failed")
	}
	return err
}

// Get ищет объект в БД по ID записи.
func (bs *BadSuggestion) Get(ctx context.Context) error {
	row, err := Get(
		ctx,
		"SELECT * WHERE entry_id=$1 AND ext_db=$2 AND ext_id=$3 LIMIT 1",
		bs.EntryID, bs.ExtDB, bs.ExtID)
	if err != nil && err != pgx.ErrNoRows {
		return errors.Wrapf(
			err,
			"BadSuggestion.Get() failed: entry_id=%d, ext_db=%s, ext_id=%s",
			bs.EntryID, bs.ExtDB, bs.ExtID)
	}
	err = row.Scan()
	if err != nil && err != pgx.ErrNoRows {
		err = errors.Wrap(err, "BadSuggestion.Get() scan failed")
	}
	return err
}

// EntryBadSuggestions возвращает список ID релизов, которые не следует использовать при поиске
// новых предложений.
func EntryBadSuggestions(ctx context.Context, entryID int) ([]*BadSuggestion, error) {
	db := ctx.Value("db").(*pgx.Conn)
	if db == nil {
		return nil, errors.Wrap(ErrConnectionInContext, "EntryBadSuggestions() failed")
	}

	rows, err := db.Query(ctx, "SELECT * FROM audio.bad_suggestion WHERE entry_id=$1", entryID)
	if err != nil {
		return nil, errors.Wrap(err, "EntryBadSuggestions() select failed")
	}
	defer rows.Close()

	ret := []*BadSuggestion{}
	for rows.Next() {
		var bs BadSuggestion
		if err = rows.Scan(&bs.EntryID, &bs.ExtDB, &bs.ExtID); err != nil {
			return nil, errors.Wrap(err, "EntryBadSuggestions() scan field")
		}
		ret = append(ret, &bs)
	}

	return ret, nil
}

// DeleteEntryBadSuggestions удаляет все отвергнутые online-предложения.
func DeleteEntryBadSuggestions(ctx context.Context, entryID int) error {
	err := Delete(ctx, "DELETE FROM audio.bad_suggestion WHERE entry_id=$1", entryID)
	if err != nil {
		err = errors.Wrap(err, "DeleteEntryBadSuggestions() failed")
	}
	return err
}
