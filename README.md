# ds-audiodbm #

Микросервис работы с метаданными аудио-объектов, представленных в БД. Обмен сообщениями с микросервисом реализован с использованием [RabbitMQ](https://www.rabbitmq.com).

## Команды микросервиса

---
|     Команда      |             Описание             |Запрос|Ответ|
|------------------|----------------------------------|----------------|-----|
|ping              |проверка работы микросервиса      |{"cmd":"ping"}|{}|
|get_entry         |чтение данных каталога            |{"cmd":"get_entry","entry":{"id":123}}|{"cmd":"get_entry","entry":<...>[,"suggestions":<...>][,"actors":<...>][,"pictures":<...>]}|
|set_entry         |создание/изменение данных каталога|{"cmd":"set_entry","entry":{["id":123,]["path":"The Darkside Of the Moon"]}[,"actors":<...>][,"pictures":<...>"]}|{"cmd":"set_entry,"entry":{"id":123}}|
|delete_entry      |удаление данных о каталоге        |{"cmd":"delete_entry","entry":{"id":123}}|эхо-ответ|
|finalyze_entry    |финализация каталога              |{"cmd":"finalyze_entry","entry":{"id":123}}|{"cmd":"finalyze_entry","entry":{"id":123,"status":"finalyzed"}}|
|rename_entry      |переименование каталога альбома   |{"cmd":"rename_entry","new_path":<new_path>,"entry":{"path":<old_path>}}|эхо-ответ
---

## Системные переменные для проведения тестов

---
|Переменная|Значение|
|----------|--------|
|DS_DB_URL |строка подключения к БД|
---