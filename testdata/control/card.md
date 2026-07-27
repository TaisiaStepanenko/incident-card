# Карточка инцидента

__ID инцидента:__ inc\_001

## Краткое резюме ##

Пользователь ***user\_017*** совершил действие ***email\_send*** с файлом ***client\_base.xlsx*** в адрес ***external\_email\_001***.

## Главное событие ##

- __Event ID:__ evt\_12345
- __Action:__ email\_send
## Контекст до события ##

- evt\_12350
- evt\_12351
- evt\_12352

## Контекст после события ##

- evt\_12346
- evt\_12353
- evt\_12347
- evt\_12354

## События того же пользователя ##

- evt\_12345
- evt\_12346
- evt\_12348
- evt\_12351
- evt\_12353
- evt\_12359
- evt\_12365

## События с тем же файлом ##

- evt\_12345
- evt\_12346
- evt\_12348
- evt\_12351
- evt\_12353
- evt\_12359
- evt\_12365

## События с тем же адресатом ##

- evt\_12345
- evt\_12346

## Временная шкала ##

| Время | Событие | Пользователь | Действие | Файл | Адресат | Важность | Роль |
|:---|:---|:---|:---|:---|:---|:---:|:---:|
|2026-06-16T09:45:00Z|evt\_12350|user\_020|copy\_file|employees.xlsx|-|medium|context\_before|
|2026-06-16T10:05:00Z|evt\_12351|user\_017|open\_file|client\_base.xlsx|-|low|context\_before|
|2026-06-16T10:10:00Z|evt\_12352|user\_021|copy\_to\_usb|source\_code.zip|-|high|context\_before|
|2026-06-16T10:15:00Z|evt\_12345|user\_017|email\_send|client\_base.xlsx|external\_email\_001|high|main\_event|
|2026-06-16T10:16:00Z|evt\_12346|user\_017|email\_send|client\_base.xlsx|external\_email\_002|medium|context\_after|
|2026-06-16T10:18:00Z|evt\_12353|user\_017|print\_file|client\_base.xlsx|-|low|context\_after|
|2026-06-16T10:20:00Z|evt\_12347|user\_018|open\_file|report.pdf|-|low|context\_after|
|2026-06-16T10:22:00Z|evt\_12354|user\_022|create\_archive|logs.zip|-|low|context\_after|
|2026-06-16T10:50:00Z|evt\_12359|user\_017|copy\_file|client\_base.xlsx|-|medium|same\_file|
|2026-06-16T11:15:00Z|evt\_12348|user\_017|delete\_file|client\_base.xlsx|-|high|same\_file|
|2026-06-16T12:00:00Z|evt\_12365|user\_017|cloud\_upload|client\_base.xlsx|cloud\_storage\_003|high|same\_file|

## Подозрительные факторы ##

- Внешний адресат
- Клиентские данные
- Персональные данные

## Ссылки на исходные события ##

- ___evt\_12350___: файл __./testdata/control/events.jsonl__ строка __6__
- ___evt\_12351___: файл __./testdata/control/events.jsonl__ строка __7__
- ___evt\_12352___: файл __./testdata/control/events.jsonl__ строка __8__
- ___evt\_12345___: файл __./testdata/control/events.jsonl__ строка __1__
- ___evt\_12346___: файл __./testdata/control/events.jsonl__ строка __2__
- ___evt\_12353___: файл __./testdata/control/events.jsonl__ строка __9__
- ___evt\_12347___: файл __./testdata/control/events.jsonl__ строка __3__
- ___evt\_12354___: файл __./testdata/control/events.jsonl__ строка __10__
- ___evt\_12359___: файл __./testdata/control/events.jsonl__ строка __15__
- ___evt\_12348___: файл __./testdata/control/events.jsonl__ строка __4__
- ___evt\_12365___: файл __./testdata/control/events.jsonl__ строка __21__
