// Менеджер БД тегов для аудио репозитория.
// Следит за целостностью БД самостоятельно.

package dbm

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"

	"github.com/ytsiuryn/ds-audiodbm/entity"
	srv "github.com/ytsiuryn/ds-microservice"
	"github.com/ytsiuryn/go-collection"
)

// Описание сервиса
const ServiceName = "dbmaudio"

// Dbm описывает внутреннее состояние клиента Discogs.
type Dbm struct {
	*srv.Service
	ctx  context.Context
	conn *pgx.Conn
}

// New создает объект менеджера БД для аудио.
func New(dbURL string) *Dbm {
	dbm := &Dbm{Service: srv.NewService(ServiceName)}

	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		dbm.Log.Fatalln(err)
	}

	dbm.conn = conn
	dbm.ctx = context.WithValue(context.Background(), "db", dbm.conn)

	return dbm
}

// AnswerWithError заполняет структуру ответа информацией об ошибке.
func (m *Dbm) AnswerWithError(delivery *amqp.Delivery, err error, context string) {
	m.LogOnError(err, context)
	req := &AudioDBRequest{
		Error: srv.ErrorResponse{
			Error:   err.Error(),
			Context: context,
		},
	}
	data, err := json.Marshal(req)
	if err != nil {
		srv.FailOnError(err, "Answer marshalling error")
	}
	m.Answer(delivery, data)
}

// Контролирует сигнал завершения цикла и последующего освобождения ресурсов микросервиса.
func (m *Dbm) Start(msgs <-chan amqp.Delivery) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		for delivery := range msgs {
			var req AudioDBRequest
			if err := json.Unmarshal(delivery.Body, &req); err != nil {
				m.AnswerWithError(&delivery, err, "Message dispatcher")
				continue
			}
			m.logRequest(&req)
			m.RunCmd(&req, &delivery)
		}
	}()

	m.Log.Info("Awaiting RPC requests")
	<-c

	m.cleanup()
}

func (m *Dbm) cleanup() {
	m.conn.Close(m.ctx)
	m.Service.Cleanup()
}

// Отображение сведений о выполняемом запросе.
func (m *Dbm) logRequest(req *AudioDBRequest) {
	if req.Entry != nil && req.Entry.ID != 0 {
		m.Log.Infof(req.Cmd+"(%d)", req.Entry.ID)
	} else {
		m.Log.Info(req.Cmd + "()")
	}
}

// RunCmd выполняет команды и возвращает результат клиенту в виде JSON-сообщения.
func (m *Dbm) RunCmd(req *AudioDBRequest, delivery *amqp.Delivery) {
	var data []byte
	var err error

	switch req.Cmd {
	case "get_entry":
		data, err = m.getEntry(req)
	case "set_entry":
		data, err = m.setEntry(req)
	case "delete_entry":
		data, err = m.deleteEntry(req)
	case "finalyze_entry":
		data, err = m.finalyzeEntry(req)
	case "rename_entry":
		data, err = m.renameEntry(req)
	default:
		m.Service.RunCmd(req.Cmd, delivery)
		return
	}

	if err != nil {
		m.AnswerWithError(delivery, err, req.Cmd)
	} else {
		m.Answer(delivery, data)
	}
}

// Чтение информации по Entry ID или его пути.
// Заполняются таблицы запроса `AudioDBRequest` и для него формируется JSON.
func (m *Dbm) getEntry(req *AudioDBRequest) (_ []byte, err error) {
	if err = req.Entry.Get(m.ctx); err != nil {
		return
	}
	req.Actors, err = entity.EntryActors(m.ctx, req.Entry.ID)
	if err != nil {
		return
	}
	req.Pictures, err = entity.EntryPictures(m.ctx, req.Entry.ID)
	if err != nil {
		return
	}
	req.Suggestions, err = entity.EntrySuggestions(m.ctx, req.Entry.ID)
	if err != nil {
		return
	}
	req.BadSuggestions, err = entity.EntryBadSuggestions(m.ctx, req.Entry.ID)
	if err != nil {
		return
	}
	return json.Marshal(req)
}

// Создание записи или изменение существующих данных по каталогу.
// В случае успеха возвращает ID записи Entry.
func (m *Dbm) setEntry(req *AudioDBRequest) (_ []byte, err error) {
	var tx pgx.Tx
	tx, err = m.conn.Begin(m.ctx)
	if err != nil {
		return
	}
	defer m.completeTx(tx, err)
	txctx := context.WithValue(m.ctx, "tx", tx)
	req.Entry.LastModified = req.Entry.LastModified.UTC()
	if req.Entry.ID == 0 {
		err = req.Entry.Create(txctx)
	} else {
		err = req.Entry.Update(txctx)
	}
	if err != nil {
		return
	}
	if err = syncEntryPictures(txctx, req); err != nil {
		return
	}
	if err = syncEntryActors(txctx, req); err != nil {
		return
	}
	if err = syncSuggestions(txctx, req); err != nil {
		return
	}
	if err = syncBadSuggestions(txctx, req); err != nil {
		return
	}
	return json.Marshal(req)
}

// Удаление информации по каталогу по его ID.
// В случае успеха возвращает пустую байтовую последовательность.
func (m *Dbm) deleteEntry(req *AudioDBRequest) (_ []byte, err error) {

	var tx pgx.Tx
	tx, err = m.conn.Begin(m.ctx)
	if err != nil {
		return
	}
	defer m.completeTx(tx, err)
	if req.Entry.ID == 0 {
		err = req.Entry.Get(m.ctx)
		if err != nil && errors.Cause(err) != pgx.ErrNoRows {
			return
		}
	}
	txctx := context.WithValue(m.ctx, "tx", tx)
	if err = entity.DeleteEntryPictures(txctx, req.Entry.ID); err != nil {
		return
	}
	if err = entity.DeleteEntryActors(txctx, req.Entry.ID); err != nil {
		return
	}
	if err = entity.DeleteEntryBadSuggestions(txctx, req.Entry.ID); err != nil {
		return
	}
	if err = entity.DeleteEntrySuggestions(txctx, req.Entry.ID); err != nil {
		return
	}
	if err = req.Entry.Delete(txctx); err != nil {
		return
	}
	return json.Marshal(req)
}

// finalyze закрывает Entry для дальнейшего редактирования.
// При этом удаляются данные по Entry из таблиц:
//
// * audio.suggestion
//
// * audio.bad_suggestion
//
// * audio.actor для уникальных акторов audio.suggestion
//
// В таблице audio.album_entry устанавливается статус "finalyzed".
// В случае успеха возвращает пустую байтовую последовательность.
func (m *Dbm) finalyzeEntry(req *AudioDBRequest) (_ []byte, err error) {
	var tx pgx.Tx
	tx, err = m.conn.Begin(m.ctx)
	if err != nil {
		return
	}
	defer m.completeTx(tx, err)
	txctx := context.WithValue(m.ctx, "tx", tx)

	if err = entity.DeleteEntrySuggestions(txctx, req.Entry.ID); err != nil {
		return
	}

	if err = entity.DeleteEntryBadSuggestions(txctx, req.Entry.ID); err != nil {
		return
	}

	if err = req.Entry.Get(m.ctx); err != nil {
		return
	}
	req.Entry.Status = "finalyzed"
	if err = req.Entry.Update(txctx); err != nil {
		return
	}

	return json.Marshal(req)
}

// renameEntry переименовывает наименование каталога альбома.
// Возвращает эхо-ответ в случае успеха.
func (m *Dbm) renameEntry(req *AudioDBRequest) (_ []byte, err error) {
	tx, err := m.conn.Begin(m.ctx)
	if err != nil {
		return
	}
	defer m.completeTx(tx, err)

	entry := *req.Entry
	if err = entry.Get(m.ctx); err != nil {
		return
	}

	entry.Path = req.NewPath
	err = entry.Update(context.WithValue(m.ctx, "tx", tx))
	if err != nil {
		return
	}

	return json.Marshal(req)
}

func (m *Dbm) completeTx(tx pgx.Tx, err error) {
	if err != nil {
		tx.Rollback(m.ctx)
	} else {
		tx.Commit(m.ctx)
	}
}

// Добавляет или заменяет графические объекты альбома.
func syncEntryPictures(ctx context.Context, req *AudioDBRequest) error {
	oldPictures, err := entity.EntryPictures(ctx, req.Entry.ID)
	if err != nil {
		return err
	}
	for _, pict := range req.Pictures {
		pict.EntID = req.Entry.ID
	}
	for _, pict := range oldPictures {
		if !collection.Contains(pict, req.Pictures) {
			if err = pict.Delete(ctx); err != nil {
				return err
			}
		}
	}
	for _, pict := range req.Pictures {
		if !collection.Contains(pict, oldPictures) {
			if err = pict.Create(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

// Добавляет или заменяет идентификаторы акторов во внешних БД.
func syncEntryActors(ctx context.Context, req *AudioDBRequest) error {
	oldActors, err := entity.EntryActors(ctx, req.Entry.ID)
	if err != nil {
		return err
	}
	for _, actor := range req.Actors {
		actor.EntryID = req.Entry.ID
	}
	for _, actor := range oldActors {
		if !collection.Contains(actor, req.Actors) {
			if err := actor.Delete(ctx); err != nil {
				return err
			}
		}
	}
	for _, actor := range req.Actors {
		if !collection.Contains(actor, oldActors) {
			if err := actor.Create(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

// Добавляет или удаляет online-предложения.
func syncSuggestions(ctx context.Context, req *AudioDBRequest) error {
	for _, suggestion := range req.Suggestions {
		suggestion.EntryID = req.Entry.ID
	}
	oldSuggestions, err := entity.EntrySuggestions(ctx, req.Entry.ID)
	if err != nil {
		return err
	}
	for _, suggestion := range oldSuggestions {
		if !collection.Contains(suggestion, req.Suggestions) {
			if err := suggestion.Delete(ctx); err != nil {
				return err
			}
		}
	}
	for _, suggestion := range req.Suggestions {
		if !collection.Contains(suggestion, oldSuggestions) {
			if err := suggestion.Create(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

// Добавляет или удаляет исключения для online-предложений.
func syncBadSuggestions(ctx context.Context, req *AudioDBRequest) error {
	for _, badSuggestion := range req.BadSuggestions {
		badSuggestion.EntryID = req.Entry.ID
	}
	oldBadSuggestions, err := entity.EntryBadSuggestions(ctx, req.Entry.ID)
	if err != nil {
		return err
	}
	for _, badSuggestion := range oldBadSuggestions {
		if !collection.Contains(badSuggestion, req.BadSuggestions) {
			if err := badSuggestion.Delete(ctx); err != nil {
				return err
			}
		}
	}
	for _, badSuggestion := range req.BadSuggestions {
		if !collection.Contains(badSuggestion, oldBadSuggestions) {
			if err := badSuggestion.Create(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}
