package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Вспомогательная функция MakeLimitSlice
func TestMakeLimitSlise(t *testing.T) {
	events := []*Event{
		{EventID: "evt_12345", TimeStamp: "2026-06-16T10:10:00Z"},
		{EventID: "evt_12346", TimeStamp: "2026-06-16T10:15:00Z"},
		{EventID: "evt_12347", TimeStamp: "2026-06-16T10:20:00Z"},
		{EventID: "evt_12348", TimeStamp: "2026-06-16T10:00:00Z"},
		{EventID: "evt_12349", TimeStamp: "2026-06-16T10:30:00Z"},
	}

	// Лимит больше количества событий
	limitSlice := MakeLimitSlice(events, 10)
	assert.Len(t, limitSlice, 5)
	assert.Equal(t, events, limitSlice)

	// Лимит равен количества событий
	limitSlice = MakeLimitSlice(events, 5)
	assert.Len(t, limitSlice, 5)
	assert.Equal(t, events, limitSlice)

	// Лимит меньше количества событий
	limitSlice = MakeLimitSlice(events, 2)
	assert.Len(t, limitSlice, 2)
	assert.Equal(t, "evt_12345", limitSlice[0].EventID)
	assert.Equal(t, "evt_12346", limitSlice[1].EventID)
}

// Функция BuildAnswer
func TestBuildAnswer(t *testing.T) {
	fileID := "file_001"
	fileName := "file.txt"
	destinationID := "dst_001"
	events := []Event{
		{EventID: "evt_12345", TimeStamp: "2026-06-16T10:15:00Z", UserID: "user_001", Action: "send", FileID: &fileID, FileName: &fileName, DestinationID: &destinationID},
		{EventID: "evt_12346", TimeStamp: "2026-06-16T10:10:00Z", UserID: "user_001", Action: "open", FileID: &fileID, FileName: &fileName},
		{EventID: "evt_12347", TimeStamp: "2026-06-16T10:17:00Z", UserID: "user_001", Action: "close", FileID: &fileID, FileName: &fileName},
		{EventID: "evt_12348", TimeStamp: "2026-06-16T10:12:00Z", UserID: "user_002", Action: "open", FileID: &fileID, FileName: &fileName},
		{EventID: "evt_12349", TimeStamp: "2026-06-16T10:14:00Z", UserID: "user_003", Action: "open", FileID: &fileID, FileName: &fileName},
		{EventID: "evt_12350", TimeStamp: "2026-06-16T10:16:00Z", UserID: "user_004", Action: "send", DestinationID: &destinationID},
	}

	links := []LinkInFile{
		{EventID: "evt_12345", FileName: "eventsList.jsonl", FileLine: 1},
		{EventID: "evt_12346", FileName: "eventsList.jsonl", FileLine: 2},
		{EventID: "evt_12347", FileName: "eventsList.jsonl", FileLine: 3},
		{EventID: "evt_12348", FileName: "eventsList.jsonl", FileLine: 4},
		{EventID: "evt_12349", FileName: "eventsList.jsonl", FileLine: 5},
		{EventID: "evt_12350", FileName: "eventsList.jsonl", FileLine: 6},
	}

	flag_true := true

	req := Request{
		IncidentID:             "inc_001",
		MainEventID:            "evt_12345",
		WindowBefore:           "10m",
		WindowAfter:            "10m",
		IncludeSameUser:        &flag_true,
		IncludeSameFile:        &flag_true,
		IncludeSameDestination: &flag_true,
		MaxEventsPerSection:    50,
	}

	index := BuildIndex(events)

	// Проверка на существование события с event_id главного события
	mainEvent, isExist := index.GetEvent(req.MainEventID)
	require.True(t, isExist)

	answer, _ := BuildAnswer(mainEvent, index, events, links, req, []Rule{})

	// Проверяем успешно ли записаны данные при вызове BuildAnswer
	assert.Equal(t, "inc_001", answer.IncidentID)
	assert.Equal(t, "evt_12345", answer.MainEvent.EventID)
	assert.Equal(t, "user_001 file.txt", answer.Summary)
	assert.Equal(t, "send", answer.MainEvent.Action)

	// События контекста до
	assert.Contains(t, answer.ContextBefore, "evt_12346")
	assert.Contains(t, answer.ContextBefore, "evt_12348")
	assert.Contains(t, answer.ContextBefore, "evt_12349")

	// События контекста после
	assert.Contains(t, answer.ContextAfter, "evt_12347")
	assert.Contains(t, answer.ContextAfter, "evt_12350")

	// События того же пользователя
	assert.Contains(t, answer.SameUserEvents, "evt_12346")
	assert.Contains(t, answer.SameUserEvents, "evt_12347")

	// События того же файла
	assert.Contains(t, answer.SameFileEvents, "evt_12346")
	assert.Contains(t, answer.SameFileEvents, "evt_12347")
	assert.Contains(t, answer.SameFileEvents, "evt_12348")
	assert.Contains(t, answer.SameFileEvents, "evt_12349")

	// Событие с тем же адресатом
	assert.Contains(t, answer.SameDestinationEvents, "evt_12350")

	//
	expectedTimelineItems := []string{"evt_12346", "evt_12348", "evt_12349", "evt_12345", "evt_12350", "evt_12347"}

	realTimelineItems := make([]string, len(answer.TimeLine))
	for i, timeLineItem := range answer.TimeLine {
		realTimelineItems[i] = timeLineItem.EventID
	}
	assert.Equal(t, expectedTimelineItems, realTimelineItems)

	assert.Len(t, answer.LinksToTheOriginalEvents, len(realTimelineItems))
	for i, link := range answer.LinksToTheOriginalEvents {
		assert.Equal(t, realTimelineItems[i], link.EventID)
		assert.Equal(t, "eventsList.jsonl", link.FileName)
	}

	// Устанавливаем лимит меньше количесва записей
	req.MaxEventsPerSection = 4
	answer, _ = BuildAnswer(mainEvent, index, events, links, req, []Rule{})

	assert.Len(t, answer.TimeLine, 4)
	assert.Len(t, answer.LinksToTheOriginalEvents, 4)

	// Устанавливаем лимит 0 (должен автоматически установить 50 и вывести все 6 строк)
	req.MaxEventsPerSection = 0
	answer, _ = BuildAnswer(mainEvent, index, events, links, req, []Rule{})

	assert.Len(t, answer.TimeLine, 6)
	assert.Len(t, answer.LinksToTheOriginalEvents, 6)
}

func TestBuildTimeLine(t *testing.T) {

	mainEvent := &Event{EventID: "evt_12345", TimeStamp: "2026-06-16T10:15:00Z", UserID: "user_001", Action: "send"}
	contextBefore := []*Event{{EventID: "evt_12346", TimeStamp: "2026-06-16T10:10:00Z", UserID: "user_001", Action: "open"}}
	contextAfter := []*Event{{EventID: "evt_12347", TimeStamp: "2026-06-16T10:17:00Z", UserID: "user_001", Action: "close"}}
	userEvents := []*Event{{EventID: "evt_12348", TimeStamp: "2026-06-16T10:12:00Z", UserID: "user_001", Action: "close"}}
	fileEvents := []*Event{{EventID: "evt_12349", TimeStamp: "2026-06-16T10:14:00Z", UserID: "user_002", Action: "send"}}
	destinationEvents := []*Event{{EventID: "evt_12350", TimeStamp: "2026-06-16T10:16:00Z", UserID: "user_003", Action: "close"}}

	timelineItems := BuildTimeline(mainEvent, contextBefore, contextAfter, userEvents, fileEvents, destinationEvents)

	assert.Len(t, timelineItems, 6)

	assert.Equal(t, RoleMain, timelineItems[3].Role)
	assert.Equal(t, RoleBeforeContext, timelineItems[0].Role)
	assert.Equal(t, RoleAfterContext, timelineItems[5].Role)
	assert.Equal(t, RoleSameUser, timelineItems[1].Role)
	assert.Equal(t, RoleSameFile, timelineItems[2].Role)
	assert.Equal(t, RoleSameDestination, timelineItems[4].Role)

	// Проверка на соблюдение приоритета ролей (когда одно и тоже событие подходит под 2 роли)
	userEvents = []*Event{{EventID: "evt_12351", TimeStamp: "2026-06-16T10:14:00Z", UserID: "user_005", Action: "open"}}
	fileEvents = []*Event{{EventID: "evt_12351", TimeStamp: "2026-06-16T10:14:00Z", UserID: "user_005", Action: "open"}}

	timelineItems = BuildTimeline(mainEvent, contextBefore, contextAfter, userEvents, fileEvents, destinationEvents)

	assert.Equal(t, RoleSameFile, timelineItems[1].Role)

	// Только с главным событием
	timelineItems = BuildTimeline(mainEvent, nil, nil, nil, nil, nil)
	assert.Len(t, timelineItems, 1)

	// Пустые поля
	eventWithoutFields := &Event{
		EventID:   "evt_12340",
		TimeStamp: "2026-06-16T10:14:00Z",
		UserID:    "user_001",
		Action:    "open",
	}

	timelineItems = BuildTimeline(mainEvent, []*Event{eventWithoutFields}, nil, nil, nil, nil)
	assert.Len(t, timelineItems, 2)

	// проверяем, что поля пустые
	assert.Equal(t, "", timelineItems[0].FileName)
	assert.Equal(t, "", timelineItems[0].Destination)
	assert.Equal(t, "", timelineItems[0].Severity)

}

// Вспомогательная функция BuildSummary
func TestBuildSummary(t *testing.T) {
	fileName := "file.txt"

	// Полные данные
	event := &Event{
		UserID:   "user_001",
		FileName: &fileName,
	}

	summary := BuildSummary(event)

	assert.Equal(t, "user_001 file.txt", summary)

	// В event есть только user_id
	event = &Event{
		UserID: "user_002",
	}

	summary = BuildSummary(event)

	assert.Equal(t, "user_002", summary)
}

// Вспомогательная функция FindIDs
func TestFindIDs(t *testing.T) {
	events := []*Event{{EventID: "evt_001"}, {EventID: "evt_002"}, {EventID: "evt_003"}}

	ids := FindIDs(events)

	expectedIds := []string{"evt_001", "evt_002", "evt_003"}
	assert.Equal(t, expectedIds, ids)

	// При отсутствии событий
	ids = FindIDs([]*Event{})
	assert.Empty(t, ids)
}

// Вспомогательная функция WriteSummaryText
func TestWriteSummaryText(t *testing.T) {
	fileName := "file.txt"
	destination := "dst_001"

	// Полные данные
	event := &Event{
		UserID:      "user_001",
		FileName:    &fileName,
		Action:      "send",
		Destination: &destination,
	}

	summary := WriteSummaryText(event)

	assert.Equal(t, "Пользователь ***user_001*** совершил действие ***send*** с файлом ***file.txt*** в адрес ***dst_001***.\n\n", summary)

	// В event есть только user_id и action
	event = &Event{
		UserID: "user_001",
		Action: "send",
	}

	summary = WriteSummaryText(event)

	assert.Equal(t, "Пользователь ***user_001*** совершил действие ***send***.\n\n", summary)

	// В event есть только user_id, action и file_name
	event = &Event{
		UserID:   "user_001",
		FileName: &fileName,
		Action:   "send",
	}

	summary = WriteSummaryText(event)

	assert.Equal(t, "Пользователь ***user_001*** совершил действие ***send*** с файлом ***file.txt***.\n\n", summary)

	// В event есть только user_id, action и destination
	event = &Event{
		UserID:      "user_001",
		Action:      "send",
		Destination: &destination,
	}

	summary = WriteSummaryText(event)

	assert.Equal(t, "Пользователь ***user_001*** совершил действие ***send*** в адрес ***dst_001***.\n\n", summary)
}

func TestGenerateMarkdownCard(t *testing.T) {
	fileName := "file.txt"
	destination := "dst_001"
	mainEvent := &Event{EventID: "evt_12345", TimeStamp: "2026-06-16T10:15:00Z", UserID: "user_001", Action: "send", FileName: &fileName, Destination: &destination}

	answer := &Answer{
		IncidentID: "inc_001",
		MainEvent: MainEvent{
			EventID: "evt_12345",
			Action:  "send",
		},
		Summary:               "user_001 file.txt",
		ContextBefore:         []string{"evt_12346"},
		ContextAfter:          []string{"evt_12347"},
		SameUserEvents:        []string{"evt_12346", "evt_12347"},
		SameFileEvents:        []string{"evt_12346"},
		SameDestinationEvents: []string{"evt_12348"},
		TimeLine: []TimelineItem{
			{Timestamp: "2026-06-16T10:10:00Z", EventID: "evt_12346", Role: RoleBeforeContext, UserID: "user_001", Action: "open", FileName: "file.txt", Destination: "", Severity: "low"},
			{Timestamp: "2026-06-16T10:15:00Z", EventID: "evt_12345", Role: RoleMain, UserID: "user_001", Action: "send", FileName: "file.txt", Destination: "dst_001", Severity: "high"},
			{Timestamp: "2026-06-16T10:20:00Z", EventID: "evt_12347", Role: RoleAfterContext, UserID: "user_001", Action: "delete", FileName: "file.txt", Destination: "", Severity: "medium"},
			{Timestamp: "2026-06-16T10:14:00Z", EventID: "evt_12348", Role: RoleSameDestination, UserID: "user_002", Action: "send", FileName: "file.txt", Destination: "dst_002", Severity: "medium"},
		},
		LinksToTheOriginalEvents: []LinkInFile{
			{EventID: "evt_12345", FileName: "eventsList.jsonl", FileLine: 1},
			{EventID: "evt_12346", FileName: "eventsList.jsonl", FileLine: 2},
			{EventID: "evt_12347", FileName: "eventsList.jsonl", FileLine: 3},
			{EventID: "evt_12348", FileName: "eventsList.jsonl", FileLine: 4},
		},
		SuspiciousFactors: []string{"external_destination", "client_data"},
	}

	markdownCard := GenerateMarkdownCard(mainEvent, answer, Index{}, 2)

	assert.Contains(t, markdownCard, "# Карточка инцидента\n\n")
	assert.Contains(t, markdownCard, "__ID инцидента:__ inc_001\n\n")
	assert.Contains(t, markdownCard, "Пользователь ***user_001*** совершил действие ***send*** с файлом ***file.txt*** в адрес ***dst_001***.\n\n")
	assert.Contains(t, markdownCard, "## Главное событие ##\n\n")
	assert.Contains(t, markdownCard, "- __Event ID:__ evt_12345\n")
	assert.Contains(t, markdownCard, "- __Action:__ send\n")
	assert.Contains(t, markdownCard, "## Контекст до события ##\n\n- evt_12346\n")
	assert.Contains(t, markdownCard, "## Контекст после события ##\n\n- evt_12347\n")
	assert.Contains(t, markdownCard, "## События того же пользователя ##\n\n- evt_12346\n- evt_12347\n")
	assert.Contains(t, markdownCard, "## События с тем же файлом ##\n\n- evt_12346\n")
	assert.Contains(t, markdownCard, "## События с тем же адресатом ##\n\n- evt_12348\n")
	assert.Contains(t, markdownCard, "## Временная шкала ##\n\n")
	assert.Contains(t, markdownCard, "Количество записей превысило максимально возможное значение (truncated). В таблице приведены первые 2 событий из 4.\n\n")
	assert.Contains(t, markdownCard, "| Время | Событие | Пользователь | Действие | Файл | Адресат | Важность | Роль |\n")
	assert.Contains(t, markdownCard, "evt_12346")
	assert.Contains(t, markdownCard, "evt_12345")
	assert.Contains(t, markdownCard, "evt_12347")
	assert.Contains(t, markdownCard, "evt_12348")
	assert.Contains(t, markdownCard, "## Подозрительные факторы ##\n\n- external_destination\n- client_data\n")
	assert.Contains(t, markdownCard, "## Ссылки на исходные события ##\n\n- ___evt_12345___: файл __eventsList.jsonl__ строка __1__\n- ___evt_12346___: файл __eventsList.jsonl__ строка __2__\n- ___evt_12347___: файл __eventsList.jsonl__ строка __3__\n- ___evt_12348___: файл __eventsList.jsonl__ строка __4__\n")

	// Пустые разделы и timeline
	answer = &Answer{
		IncidentID: "inc_001",
		MainEvent: MainEvent{
			EventID: "evt_12345",
			Action:  "send",
		},
		Summary:                  "user_001",
		ContextBefore:            []string{},
		ContextAfter:             []string{},
		SameUserEvents:           []string{},
		SameFileEvents:           []string{},
		SameDestinationEvents:    []string{},
		TimeLine:                 []TimelineItem{},
		LinksToTheOriginalEvents: []LinkInFile{},
		SuspiciousFactors:        []string{},
	}

	markdownCard = GenerateMarkdownCard(mainEvent, answer, Index{}, 50)

	assert.Contains(t, markdownCard, "Подходящих для данного раздела событий не найдено.\n\n")

}
