package entity

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
)

// Ошибки работы с БД.
var (
	ErrConnectionInContext = errors.New("could not get database connection pool from context")
)

// Insert создает объект в БД и возвращает ID новой записи.
func Insert(ctx context.Context, cmd string, args ...interface{}) (id int, err error) {
	tx := ctx.Value("tx").(pgx.Tx)
	if tx == nil {
		return 0, ErrConnectionInContext
	}
	if err = tx.QueryRow(ctx, cmd, args...).Scan(&id); err != nil {
		return 0, errors.Wrapf(err, "Insert() error: %v", ArgsAsStrings(args))
	}
	return
}

// InsertFullRec - общая функция создания объекта в БД "с нуля".
func InsertFullRec(ctx context.Context, cmd string, args ...interface{}) (err error) {
	tx := ctx.Value("tx").(pgx.Tx)
	if tx == nil {
		return ErrConnectionInContext
	}
	if _, err = tx.Exec(ctx, cmd, args...); err != nil {
		return errors.Wrapf(err, fmt.Sprint("InsertFullRec() error:", ArgsAsStrings(args)))
	}
	return
}

// Delete - общая функция удаления объекта из таблицы tblName по первичному ключу,
// представленному полями `pkFldValues`.
func Delete(ctx context.Context, cmd string, pkFldValues ...interface{}) error {
	tx := ctx.Value("tx").(pgx.Tx)
	if tx == nil {
		return ErrConnectionInContext
	}
	_, err := tx.Exec(ctx, cmd, pkFldValues...)
	return err
}

// Get - общая функция для получения объекта в таблице tblName по первичному ключу,
// представленному полями `pkFldValues`.
// Если запись не найдена, на выходе pgx.Row = nil.
func Get(ctx context.Context, qry string, pkFldValues ...interface{}) (pgx.Row, error) {
	db := ctx.Value("db").(*pgx.Conn)
	if db == nil {
		return nil, ErrConnectionInContext
	}
	return db.QueryRow(ctx, qry, pkFldValues...), nil
}

// ArgsAsStrings фильтрует аргументы запросов, удаляя из списка параметры []byte
// и заменяя их на фразу "...long field skipped...".
func ArgsAsStrings(args []interface{}) []string {
	ret := make([]string, 0, len(args))
	for _, v := range args {
		fmt.Println(fmt.Sprintf("%T", v))
		switch v.(type) {
		case int8, int16, int32, int64, int:
			ret = append(ret, fmt.Sprintf("%d", v.(int)))
		case float32, float64:
			ret = append(ret, fmt.Sprintf("%f", v.(float64)))
		case []uint8:
			ret = append(ret, "...long field skipped...")
		default:
			ret = append(ret, fmt.Sprintf("%s", v))
		}
	}
	return ret
}
