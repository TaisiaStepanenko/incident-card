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
		{EventID: "evt_12346", TimeStamp: "2026-06-16T10:10:00Z", UserID: "user_001", Action: "open", FileID: &fileID, FileName: &fileName,},
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
		IncidentID: "inc_001",
		MainEventID: "evt_12345",
		WindowBefore: "10m",      
		WindowAfter:  "10m",
		IncludeSameUser: &flag_true,
		IncludeSameFile: &flag_true, 
		IncludeSameDestination: &flag_true,
		MaxEventsPerSection:  50,  
	}

	index := BuildIndex(events)

	// Проверка на существование события с event_id главного события
	mainEvent, isExist := index.GetEvent(req.MainEventID)
	require.True(t, isExist)

	answer := BuildAnswer(mainEvent, index, events, links, req)

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
	expectedItemsIds := []string{"evt_12345", "evt_12346", "evt_12347", "evt_12348", "evt_12349", "evt_12350"}
	for i, link := range answer.LinksToTheOriginalEvents {
		assert.Equal(t, expectedItemsIds[i], link.EventID)
		assert.Equal(t, "eventsList.jsonl", link.FileName)
	}
}

func TestBuildTimeLineRoles(t *testing.T) {

	mainEvent := &Event{EventID: "evt_12345", TimeStamp: "2026-06-16T10:15:00Z", UserID: "user_001", Action: "send"}
	contextBefore := []*Event{{EventID: "evt_12346", TimeStamp: "2026-06-16T10:10:00Z", UserID: "user_001", Action: "open"}}
	contextAfter := []*Event{{EventID: "evt_12347", TimeStamp: "2026-06-16T10:17:00Z", UserID: "user_001", Action: "close"}}
	userEvents := []*Event{{EventID: "evt_12348", TimeStamp: "2026-06-16T10:12:00Z", UserID: "user_001", Action: "close"}}
	fileEvents := []*Event{{EventID: "evt_12349", TimeStamp: "2026-06-16T10:14:00Z", UserID: "user_002", Action: "send"}}
	destinationEvents := []*Event{{EventID: "evt_12350", TimeStamp: "2026-06-16T10:16:00Z", UserID: "user_003", Action: "close"}}

	links := []LinkInFile{
		{EventID: "evt_12345", FileName: "eventsList.jsonl", FileLine: 1},
		{EventID: "evt_12346", FileName: "eventsList.jsonl", FileLine: 2},
		{EventID: "evt_12347", FileName: "eventsList.jsonl", FileLine: 3},
		{EventID: "evt_12348", FileName: "eventsList.jsonl", FileLine: 4},
		{EventID: "evt_12349", FileName: "eventsList.jsonl", FileLine: 5},
		{EventID: "evt_12350", FileName: "eventsList.jsonl", FileLine: 6},
	}

	timelineItems, _ := BuildTimeline(mainEvent, contextBefore,contextAfter, userEvents, fileEvents, destinationEvents, links)

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

	timelineItems, _ = BuildTimeline(mainEvent, contextBefore,contextAfter, userEvents, fileEvents, destinationEvents, links)

	assert.Equal(t, RoleSameFile, timelineItems[1].Role)
}

// Вспомогательная функция BuildSummary
func TestBuildSummary(t *testing.T) {
	fileName := "file.txt"
	
	// Полные данные
	event := &Event{
		UserID: "user_001",
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