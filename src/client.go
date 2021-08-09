package dbm

import (
	"encoding/json"
	"errors"

	"github.com/gofrs/uuid"

	// "entity"
	"github.com/ytsiuryn/ds-audiodbm/src/entity"
	md "github.com/ytsiuryn/ds-audiomd"
	srv "github.com/ytsiuryn/ds-microservice"
)

// AudioDBRequest описывает структуру пакета запроса к сервису.
// Обязательным для заполнения является команда `Cmd` и `Entry.ID`
// (кроме случая создания новой записи).
// Объект этого типа может использоваться клиентом сервиса как "долгоиграющий"
// с динамическим обновлением исходных метаданных.
type AudioDBRequest struct {
	Cmd            string                  `json:"cmd"`
	NewPath        string                  `json:"new_path,omitempty"`
	Entry          *entity.AlbumEntry      `json:"entry,omitempty"`
	Suggestions    []*entity.Suggestion    `json:"suggestions,omitempty"`
	BadSuggestions []*entity.BadSuggestion `json:"bad_suggestions,omitempty"`
	Actors         []*entity.Actor         `json:"actors,omitempty"`
	Pictures       []*entity.Picture       `json:"pictures,omitempty"`
}

// AudioDBResponse описывает структуру ответа
type AudioDBResponse struct {
	*AudioDBRequest
	Error *srv.ErrorResponse `json:"error,omitempty"`
}

// NewAudioDBRequest создает объект запроса.
func NewAudioDBRequest(cmd string, entry *entity.AlbumEntry) *AudioDBRequest {
	return &AudioDBRequest{Cmd: cmd, Entry: entry}
}

// ImportAssumption импортирует метаданные объекта `md.Assumption` в запрос.
// Возвращает jsonData типа `md.Realease` и CorrelationID для запроса к RabbitMQ.
func (req *AudioDBRequest) ImportAssumption(assumption *md.Assumption) (err error) {

	req.clearAssumptionMetadata()

	req.Entry.Json, err = json.Marshal(assumption.Release)
	if err != nil {
		return
	}

	req.mergeActors(assumption.Actors, entity.AlbumEntryEntity)

	for _, pict := range assumption.Pictures {
		req.Pictures = append(req.Pictures, entity.NewPicture("album_entry", req.Entry.ID, pict))
	}

	return
}

// ImportSuggestions обрабатывает предложения по Entry, формируя из них данные для запроса.
func (req *AudioDBRequest) ImportSuggestions(set *md.SuggestionSet) error {
	req.clearSuggestionsMetadata()

	for _, suggestion := range set.Suggestions {
		data, err := json.Marshal(suggestion.Release)
		if err != nil {
			return err
		}
		req.Suggestions = append(
			req.Suggestions,
			&entity.Suggestion{
				EntryID: req.Entry.ID,
				ExtDB:   suggestion.ServiceName,
				ExtID:   suggestion.Release.IDs[suggestion.ServiceName],
				Json:    data,
				Score:   suggestion.SourceSimilarity})
	}

	req.mergeActors(set.Actors, entity.SuggestionEntity)

	return nil
}

// Create генерирует CorrelationID и дамп данных для запроса.
func (req *AudioDBRequest) Create() (_ string, data []byte, err error) {
	correlationID, err := uuid.NewV4()
	if err != nil {
		return
	}

	data, err = json.Marshal(&req)
	if err != nil {
		return
	}

	return correlationID.String(), data, nil
}

// ClearMetaData очистка полей с метаданными для повторного использования запроса
// и сокращения его размера.
// Зачищаются Entry.Json, Actors, Suggestions, BadSuggestion и Pictures.
func (req *AudioDBRequest) ClearMetaData() {
	req.Entry.Json = nil
	req.Actors = nil
	req.BadSuggestions = nil
	req.Pictures = nil
	req.Suggestions = nil
}

func (req *AudioDBRequest) clearAssumptionMetadata() {
	req.Entry.Json = nil
	req.clearActorWithMask(entity.AlbumEntryEntity)
	req.Pictures = nil
}

func (req *AudioDBRequest) clearSuggestionsMetadata() {
	req.clearActorWithMask(entity.SuggestionEntity)
	req.Suggestions = nil
	req.BadSuggestions = nil
}

func (req *AudioDBRequest) mergeActors(actors md.ActorIDs, mask entity.EntityMask) {
	m := map[string]*entity.Actor{}
	for _, actor := range req.Actors {
		m[actor.Name] = actor
	}
	for name, actorIDs := range actors {
		actor, ok := m[name]
		if !ok {
			actor = &entity.Actor{
				EntryID:    req.Entry.ID,
				Name:       name,
				EntityMask: mask}
			req.Actors = append(req.Actors, actor)
		}
		ids := actor.IDs
		for extDB, id := range actorIDs {
			var found bool
			for _, pair := range actor.IDs {
				if extDB == pair[0] {
					found = true
					break
				}
			}
			if !found {
				ids = append(ids, [2]string{extDB, id})
			}
		}
		actor.IDs = ids
	}
}

func (req *AudioDBRequest) clearActorWithMask(mask entity.EntityMask) {
	for i := len(req.Actors) - 1; i >= 0; i-- {
		if req.Actors[i].EntityMask == mask {
			req.Actors = append(req.Actors[:i], req.Actors[i+1:]...)
		} else {
			req.Actors[i].EntityMask ^= mask
		}
	}
}

// MustBeOK контроллирует значение ответа микросервиса, и, в случае ошибки,
// печатает сведения об ошибке и останавливает процесс с запущенным клиентом.
func (resp *AudioDBResponse) Unwrap() *AudioDBRequest {
	if resp.Error != nil {
		srv.FailOnError(errors.New(resp.Error.Error), resp.Error.Context)
	}
	return resp.AudioDBRequest
}

// ParseAnswer разбирает JSON ответа и импортирует данные в объект `AudioDBRequest`.
func ParseAnswer(data []byte) (_ *AudioDBResponse, err error) {
	entry := AudioDBResponse{}
	if err = json.Unmarshal(data, &entry); err != nil {
		return
	}
	return &entry, nil
}
