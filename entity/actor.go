package entity

// TODO: для описания типов использовать https://github.com/jackc/pgtype

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
)

type EntityMask uint8

const (
	AlbumEntryEntity EntityMask = 1
	SuggestionEntity EntityMask = 2
)

// Actor хранит идентификаторы ссылок на объекты во внешних БД.
type Actor struct {
	EntryID    int         `sql:"entry_id" json:"entry_id"`
	Name       string      `json:"name"`
	IDs        [][2]string `json:"ids"`
	EntityMask EntityMask  `sql:"entity_mask" json:"entity_mask"`
}

// Create записывает объект в БД.
func (a *Actor) Create(ctx context.Context) (err error) {
	err = InsertFullRec(
		ctx,
		`INSERT INTO audio.actor (entry_id,name,ids,entity_mask) VALUES($1,$2,$3,$4)`,
		a.EntryID, a.Name, a.IDs, a.EntityMask)

	if err != nil {
		errors.Wrapf(err, "Actor.Create() failed: entry_id=%d, name=%s", a.EntryID, a.Name)
	}
	return
}

// Delete удаляет объект в БД по ID записи.
func (a *Actor) Delete(ctx context.Context) (err error) {
	err = Delete(
		ctx,
		`DELETE FROM audio.actor WHERE entry_id=$1 AND name=$2`,
		a.EntryID, a.Name)

	if err != nil {
		errors.Wrapf(err, "Actor.Delete() failed")
	}
	return
}

// Get ищет объект Release по значению ключа записи.
func (a *Actor) Get(ctx context.Context) error {
	qry := `SELECT ids,entity_mask FROM audio.actor WHERE entry_id=$1 AND name=$2 LIMIT 1`
	row, err := Get(ctx, qry, a.EntryID, a.Name)
	if err != nil && err != pgx.ErrNoRows {
		return errors.Wrap(err, "Actor.Get() failed")
	}
	err = row.Scan(&a.IDs, &a.EntityMask)
	if err != nil {
		return errors.Wrapf(
			err, "Actor.Get() scan failed: entry_id=%d, name=%s", a.EntryID, a.Name)
	}
	return err
}

// EntryActors возвращает список акторов для указанного Entry.
func EntryActors(ctx context.Context, entryID int) ([]*Actor, error) {
	db := ctx.Value("db").(*pgx.Conn)
	if db == nil {
		return nil, errors.Wrap(ErrConnectionInContext, "EntryActors() failed")
	}

	rows, err := db.Query(ctx, "SELECT * FROM audio.actor WHERE entry_id=$1", entryID)
	if err != nil {
		return nil, errors.Wrap(err, "EntryActors() select failed")
	}
	defer rows.Close()

	var ret []*Actor
	for rows.Next() {
		var actor Actor
		err = rows.Scan(&actor.EntryID, &actor.Name, &actor.IDs, &actor.EntityMask)
		if err != nil {
			return nil, errors.Wrap(err, "EntryActors() scan failed")
		}
		ret = append(ret, &actor)
	}

	return ret, nil
}

// DeleteEntryActors удаляеи всех акторов для указанного Entry.
func DeleteEntryActors(ctx context.Context, entryID int) error {
	err := Delete(ctx, "DELETE FROM audio.actor WHERE entry_id=$1", entryID)
	if err != nil {
		err = errors.Wrap(err, "DeleteEntryActors() failed")
	}
	return err
}
