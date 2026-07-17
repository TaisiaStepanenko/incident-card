# Карточка инцидента

__ID инцидента:__ 

## Краткое резюме ##

Пользователь ***user_017*** совершил действие ***copy_to_usb*** с файлом ***report_finance.xlsx*** в адрес ***usb_device_001***.

## Главное событие ##

- __Event ID:__ evt_12345
- __Action:__ copy_to_usb
## Контекст до события ##

Подходящих для данного раздела событий не найдено.

## Контекст после события ##

Подходящих для данного раздела событий не найдено.

## События того же пользователя ##

- evt_12345
- evt_12338
- evt_12347
- evt_12346
- evt_046974
- evt_020504
- evt_028179
- evt_109025
- evt_040593
- evt_015691
- evt_086856
- evt_039196
- evt_068434

## События с тем же файлом ##

Подходящих для данного раздела событий не найдено.

## События с тем же адресатом ##

Подходящих для данного раздела событий не найдено.

## Временная шкала ##

| Время | Событие | Пользователь | Действие | Файл | Адресат | Важность | Роль |
|:---|:---|:---|:---|:---|:---|:---:|:---:|
|2026-07-14T17:32:07+03:00|evt_12338|user_017|open_file|report_finance.xlsx|-|low|same_user|
|2026-07-14T17:49:07+03:00|evt_12345|user_017|copy_to_usb|report_finance.xlsx|usb_device_001|high|main_event|
|2026-07-14T18:05:07+03:00|evt_12346|user_017|open_file|info.pdf|-|low|same_user|
|2026-07-14T18:18:30+03:00|evt_068434|user_017|email_send|user_019|-|low|same_user|
|2026-07-14T18:29:07+03:00|evt_12347|user_017|delete_file|report_finance.xlsx|-|high|same_user|
|2026-07-14T20:51:42+03:00|evt_020504|user_017|email_send|user_019|-|high|same_user|
|2026-07-15T01:44:51+03:00|evt_015691|user_017|cloud_upload|user_020|-|medium|same_user|
|2026-07-15T03:39:16+03:00|evt_039196|user_017|messenger_send|user_019|-|low|same_user|
|2026-07-15T05:16:16+03:00|evt_086856|user_017|print_file|user_018|-|high|same_user|
|2026-07-15T06:59:26+03:00|evt_109025|user_017|messenger_send|user_020|-|critical|same_user|
|2026-07-15T09:57:04+03:00|evt_028179|user_017|print_file|user_016|-|critical|same_user|
|2026-07-16T06:50:19+03:00|evt_046974|user_017|create_archive|user_017|-|medium|same_user|
|2026-07-16T07:43:26+03:00|evt_040593|user_017|messenger_send|user_016|-|medium|same_user|

## Подозрительные факторы ##

Подходящих для данного раздела событий не найдено.

## Ссылки на исходные события ##

- ___evt_015691___: файл __testdata/control/test.jsonl__ строка __27__
- ___evt_020504___: файл __testdata/control/test.jsonl__ строка __12__
- ___evt_028179___: файл __testdata/control/test.jsonl__ строка __13__
- ___evt_039196___: файл __testdata/control/test.jsonl__ строка __38__
- ___evt_040593___: файл __testdata/control/test.jsonl__ строка __17__
- ___evt_046974___: файл __testdata/control/test.jsonl__ строка __7__
- ___evt_068434___: файл __testdata/control/test.jsonl__ строка __43__
- ___evt_086856___: файл __testdata/control/test.jsonl__ строка __35__
- ___evt_109025___: файл __testdata/control/test.jsonl__ строка __15__
- ___evt_12338___: файл __testdata/control/test.jsonl__ строка __2__
- ___evt_12345___: файл __testdata/control/test.jsonl__ строка __1__
- ___evt_12346___: файл __testdata/control/test.jsonl__ строка __4__
- ___evt_12347___: файл __testdata/control/test.jsonl__ строка __3__
