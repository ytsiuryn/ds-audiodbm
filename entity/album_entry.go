package entity

// TODO: для описания типов использовать https://github.com/jackc/pgtype

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
)

// AlbumEntry описывает каталог репозитория с релизом.
type AlbumEntry struct {
	ID           int       `json:"id,omitempty"`
	Path         string    `json:"path,omitempty"`
	Json         []byte    `json:"json,omitempty"`
	Status       string    `json:"status,omitempty"` // тип audio.entry_status
	LastModified time.Time `json:"last_modified"`
}

// Create записывает объект в БД.
func (ent *AlbumEntry) Create(ctx context.Context) (err error) {
	ent.ID, err = Insert(
		ctx,
		`INSERT INTO audio.album_entry (path,json,status,last_modified) VALUES($1,$2,$3,$4)
		RETURNING id`,
		ent.Path, ent.Json, ent.Status, ent.LastModified)
	if err != nil {
		err = errors.Wrapf(err, "AlbumEntry.Create() failed: path=%s", ent.Path)
	}
	return err
}

// Update обновляет данные для записи с указанным ID.
func (ent *AlbumEntry) Update(ctx context.Context) error {
	tx := ctx.Value("tx").(pgx.Tx)
	if tx == nil {
		return errors.Wrapf(
			ErrConnectionInContext, "AlbumEntry.Update() failed: path=%s", ent.Path)
	}
	_, err := tx.Exec(
		ctx,
		"UPDATE audio.album_entry SET path=$1,json=$2,status=$3,last_modified=$4 WHERE id=$5",
		ent.Path, ent.Json, ent.Status, ent.LastModified, ent.ID)
	if err != nil {
		err = errors.Wrapf(err, "AlbumEntry.Update() failed: id=%d", ent.ID)
	}
	return err
}

// Delete удаляет объект в БД по ID записи или пути к каталогу.
func (ent *AlbumEntry) Delete(ctx context.Context) (err error) {
	if ent.ID != 0 {
		err = Delete(ctx, "DELETE FROM audio.album_entry WHERE id=$1", ent.ID)
	} else {
		err = Delete(ctx, "DELETE FROM audio.album_entry WHERE path=$1", ent.Path)
	}
	if err != nil {
		err = errors.Wrapf(err, "AlbumEntry.Delete() failed: id=%d", ent.ID)
	}
	return err
}

// ActorNames выполняет выборку всех имен акторов из JSON.
// func (ent *AlbumEntry) ActorNames(ctx context.Context) (names []string, err error) {
// 	actors, err := EntryActors(ctx, ent.ID)
// 	if err != nil {
// 		return
// 	}
// 	for _, actor := range actors {
// 		names = append(names, actor.Name)
// 	}
// 	return
// }

// Get ищет объект в БД по ID записи.
func (ent *AlbumEntry) Get(ctx context.Context) (err error) {
	var row pgx.Row
	if ent.ID != 0 {
		row, err = Get(
			ctx,
			"SELECT * FROM audio.album_entry WHERE id=$1 LIMIT 1", ent.ID)
	} else {
		row, err = Get(
			ctx,
			"SELECT * FROM audio.album_entry WHERE path=$1 LIMIT 1", ent.Path)
	}
	if err != nil && err != pgx.ErrNoRows {
		return errors.Wrapf(err, "AlbumEntry.Get() select failed: id=%d", ent.ID)
	}
	err = row.Scan(&ent.ID, &ent.Path, &ent.Json, &ent.Status, &ent.LastModified)
	if err != nil {
		err = errors.Wrapf(err, "AlbumEntry.Get() scan failed: id=%d", ent.ID)
	}
	return err
}
