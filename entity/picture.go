package entity

// TODO: для описания типов использовать https://github.com/jackc/pgtype

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

	md "github.com/ytsiuryn/ds-audiomd"
)

// Picture описывает изображение объекта метаданных.
type Picture struct {
	EntType  string `sql:"entity_type" json:"entity_type"`
	EntID    int    `sql:"entity_id" json:"entity_id"`
	PictType string `sql:"pict_type" json:"pict_type"` // тип audio.pict_type
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
	Mime     string `json:"mime"`
	Notes    string `json:"notes,omitempty"`
	Data     []byte `json:"data,omitempty"`
}

// NewPicture создает объект.
func NewPicture(entType string, entID int, pict *md.PictureInAudio) *Picture {
	return &Picture{
		EntType:  entType,
		EntID:    entID,
		PictType: pict.PictType.String(),
		Width:    int(pict.Width),
		Height:   int(pict.Height),
		Mime:     pict.MimeType,
		Notes:    pict.Notes,
		Data:     pict.Data}
}

// Pictures возвращает изображения для определенной сущности с ее ID.
func Pictures(ctx context.Context, entType, entID int) ([]*Picture, error) {
	db := ctx.Value("db").(*sqlx.DB)
	if db == nil {
		return nil, errors.Wrap(ErrConnectionInContext, "Pictures() failed")
	}
	ret := []*Picture{}
	qry := `SELECT * FROM audio.picture WHERE entity_type=$1 AND entity_id=$2`
	rows, err := db.Query(qry, entType, entID)
	if err != nil {
		return nil, errors.Wrapf(err, "Pictures() select failed")
	}
	if rows.Scan(&ret); err != nil {
		return nil, errors.Wrap(err, "Pictures() scan field")
	}
	return ret, nil
}

// Create записывает объект в БД.
func (p *Picture) Create(ctx context.Context) error {
	err := InsertFullRec(
		ctx,
		`INSERT INTO audio.picture (entity_type,entity_id,pict_type,width,height,mime,notes,data)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		p.EntType, p.EntID, p.PictType, p.Width, p.Height, p.Mime, p.Notes, p.Data)
	if err != nil {
		err = errors.Wrapf(
			err,
			"Picture.Create() failed: entity_type=%s, entity_id=%d, pict_type=%s",
			p.EntType, p.EntID, p.PictType)
	}
	return err
}

// Update обновляет данные для записи с указанным ID.
func (p *Picture) Update(ctx context.Context) (err error) {
	db := ctx.Value("db").(*sqlx.DB)
	if db == nil {
		return errors.Wrap(ErrConnectionInContext, "Picture.Update() failed")
	}
	_, err = db.Exec(
		`UPDATE audio.picture SET width=$1,height=$2,mime=$3,
		notes=$4,data=$5 WHERE entity_type=$6 AND entity_id=$7 AND pict_type=$8`,
		p.Width, p.Height, p.Mime, p.Notes, p.Data, p.EntType, p.EntID, p.PictType)
	if err != nil {
		err = errors.Wrapf(
			err,
			"Picture.Update() failed: entity_type=%s, entity_id=%d, pict_type=%s",
			p.EntType, p.EntID, p.PictType)
	}
	return err
}

// Delete удаляет объект в БД по ID записи.
func (p *Picture) Delete(ctx context.Context) error {
	err := Delete(
		ctx,
		"DELETE FROM audio.picture WHERE entity_type=$1 AND entity_id=$2 AND pict_type=$3",
		p.EntType, p.EntID, p.PictType)
	if err != nil {
		err = errors.Wrap(err, "Picture.Delete() failed")
	}
	return err
}

// Get ищет объект по значению ключа записи.
func (p *Picture) Get(ctx context.Context) error {
	qry := `SELECT width,height,mime,notes,data
	FROM audio.picture WHERE entity_type=$1 AND entity_id=$2 AND pict_type=$3 LIMIT 1`
	row, err := Get(ctx, qry, p, p.EntType, p.EntID, p.PictType)
	if err != nil && err != pgx.ErrNoRows {
		return errors.Wrap(err, "Picture.Get() select failed")
	}
	err = row.Scan(&p.Width, &p.Height, &p.Mime, &p.Notes, &p.Data)
	if err != nil {
		err = errors.Wrap(err, "Picture.Get() scan failed")
	}
	return err
}

// EntryPictures возвращает список графических объектов для Entry Assumption.
func EntryPictures(ctx context.Context, entryID int) ([]*Picture, error) {
	db := ctx.Value("db").(*pgx.Conn)
	if db == nil {
		return nil, errors.Wrap(ErrConnectionInContext, "EntryPictures() failed")
	}
	qry := "SELECT * FROM audio.picture WHERE entity_type=$1 AND entity_id=$2"
	rows, err := db.Query(ctx, qry, "album_entry", entryID)
	if err != nil && err != pgx.ErrNoRows {
		return nil, errors.Wrapf(err, "EntryPictures() select failed: entry_id=%d", entryID)
	}
	defer rows.Close()

	ret := []*Picture{}
	for rows.Next() {
		var p Picture
		err = rows.Scan(&p.EntType, &p.EntID, &p.PictType, &p.Width, &p.Height,
			&p.Mime, &p.Notes, &p.Data)
		if err != nil {
			return nil, errors.Wrap(err, "EntryPictures() scan failed")
		}
		ret = append(ret, &p)
	}

	return ret, nil
}

// DeleteEntryPictures удаляет все графические объекты Entry, содержащие данные образа.
func DeleteEntryPictures(ctx context.Context, entryID int) error {
	err := Delete(
		ctx,
		"DELETE FROM audio.picture WHERE entity_type='album_entry' AND entity_id=$1",
		entryID)
	if err != nil {
		err = errors.Wrap(err, "DeleteEntryPictures() failed")
	}
	return err
}
