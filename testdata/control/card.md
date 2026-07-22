# Карточка инцидента

__ID инцидента:__ inc_001

## Краткое резюме ##

Пользователь ***user_017*** совершил действие ***email_send*** с файлом ***client_base.xlsx*** в адрес ***external_email_001***.

## Главное событие ##

- __Event ID:__ evt_12345
- __Action:__ email_send
## Контекст до события ##

- evt_12350
- evt_12351
- evt_12352

## Контекст после события ##

- evt_12346
- evt_12353
- evt_12347
- evt_12354

## События того же пользователя ##

- evt_12345
- evt_12346
- evt_12348
- evt_12351
- evt_12353
- evt_12359
- evt_12365

## События с тем же файлом ##

- evt_12345
- evt_12346
- evt_12348
- evt_12351
- evt_12353
- evt_12359
- evt_12365

## События с тем же адресатом ##

- evt_12345
- evt_12346

## Временная шкала ##

| Время | Событие | Пользователь | Действие | Файл | Адресат | Важность | Роль |
|:---|:---|:---|:---|:---|:---|:---:|:---:|
|2026-06-16T09:45:00Z|evt_12350|user_020|copy_file|employees.xlsx|-|medium|context_before|
|2026-06-16T10:05:00Z|evt_12351|user_017|open_file|client_base.xlsx|-|low|context_before|
|2026-06-16T10:10:00Z|evt_12352|user_021|copy_to_usb|source_code.zip|-|high|context_before|
|2026-06-16T10:15:00Z|evt_12345|user_017|email_send|client_base.xlsx|external_email_001|high|main_event|
|2026-06-16T10:16:00Z|evt_12346|user_017|email_send|client_base.xlsx|external_email_002|medium|context_after|
|2026-06-16T10:18:00Z|evt_12353|user_017|print_file|client_base.xlsx|-|low|context_after|
|2026-06-16T10:20:00Z|evt_12347|user_018|open_file|report.pdf|-|low|context_after|
|2026-06-16T10:22:00Z|evt_12354|user_022|create_archive|logs.zip|-|low|context_after|
|2026-06-16T10:50:00Z|evt_12359|user_017|copy_file|client_base.xlsx|-|medium|same_file|

## Подозрительные факторы ##

- Внешний адресат
- Клиентские данные
- Персональные данные

## Ссылки на исходные события ##

- ___evt_12350___: файл __./testdata/control/events.jsonl__ строка __6__
- ___evt_12351___: файл __./testdata/control/events.jsonl__ строка __7__
- ___evt_12352___: файл __./testdata/control/events.jsonl__ строка __8__
- ___evt_12345___: файл __./testdata/control/events.jsonl__ строка __1__
- ___evt_12346___: файл __./testdata/control/events.jsonl__ строка __2__
- ___evt_12353___: файл __./testdata/control/events.jsonl__ строка __9__
- ___evt_12347___: файл __./testdata/control/events.jsonl__ строка __3__
- ___evt_12354___: файл __./testdata/control/events.jsonl__ строка __10__
- ___evt_12359___: файл __./testdata/control/events.jsonl__ строка __15__
