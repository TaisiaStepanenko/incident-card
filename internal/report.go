package internal

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
)

func MakeLimitSlice(events []*Event, limit int) []*Event {
	if (len(events) > limit) {
		return events[:limit]
	}
	return events
}

func BuildAnswer(mainEvent *Event, index Index, events []Event, req Request) Answer {

	// Сбор событий по временному контексту 
	timeEvents, err := GetEventsInTimeRange(events, mainEvent.TimeStamp, req.WindowBefore, req.WindowBefore)
	if err != nil {
		log.Fatalf("Ошибка при получении временного контекста событий: %v", err)
	}

	// События пользователя главного события (если есть в запросе)
	var userEvents []*Event
	if (req.IncludeSameUser != nil) {
		userEvents = index.GetEventByUser(mainEvent.UserID)
	}

	// События с файлом главного события (если есть в запросе)
	var fileEvents []*Event
	if (req.IncludeSameFile != nil) {
		fileEvents = index.GetEventByFile(*mainEvent.FileID)
	}

	// События адресата главного события (если есть в запросе)
	var destinationEvents []*Event
	if (req.IncludeSameDestination != nil) {
		destinationEvents = index.GetEventByFile(*mainEvent.DestinationID)
	}

	// Устанавливаем ограничение размера разделов (по умолчанию 50)
	limit := req.MaxEventsPerSection
	if (limit == 0) {
		limit = 50
	}

	var contextBefore, contextAfter []*Event
	mainTime, _ := time.Parse(time.RFC3339, mainEvent.TimeStamp)
	for _, event := range timeEvents {
		eventTime, _ := time.Parse(time.RFC3339, event.TimeStamp)
		if (eventTime.Before(mainTime)) {
			contextBefore = append(contextBefore, event)
		} else if (eventTime.After(mainTime)){
			contextAfter = append(contextAfter, event)
		}
	}

	contextBefore = MakeLimitSlice(contextBefore, limit)
	contextAfter = MakeLimitSlice(contextAfter, limit)
	userEvents = MakeLimitSlice(userEvents, limit)
	fileEvents = MakeLimitSlice(fileEvents, limit)
	destinationEvents = MakeLimitSlice(destinationEvents, limit)


	timelineItems := BuildTimeline(mainEvent, contextBefore, contextAfter, userEvents, fileEvents, destinationEvents)
	
	if (len(timelineItems) > limit) {
		timelineItems = timelineItems[:limit]
	}

	summary := BuildSummary(mainEvent)

	contextBeforeIds := FindIDs(contextBefore)
	contextAfterIds := FindIDs(contextAfter)
	userEventsIds := FindIDs(userEvents)
	fileEventsIds := FindIDs(fileEvents)
	destinationEventsIds := FindIDs(destinationEvents)

	

	return Answer{
		IncidentID: req.IncidentID,
		MainEvent: MainEvent{
			EventID: mainEvent.EventID,
			Action: mainEvent.Action,
		},
		Summary: summary,
		ContextBefore: contextBeforeIds,
		ContextAfter: contextAfterIds,
		SameUserEvents: &userEventsIds,
		SameFileEvents: &fileEventsIds,
		SameDestinationEvents: &destinationEventsIds,
		TimeLine: timelineItems,
	}

}

func BuildTimeline(mainEvent *Event, contextBefore, contextAfter, userEvents, fileEvents, destinationEvents []*Event) []TimelineItem {
	
	roleMap := make(map[string]Role) // соответствие события и его роли

	// поэтапно устанавливаем роли для всех списков, начиная от менее приоритетного,
	// чтобы при попадании события в несколько списков роль перезаписывалась на более приоритетную 
	for _, event := range userEvents {
		roleMap[event.EventID] = RoleSameUser
	}

	for _, event := range destinationEvents {
		roleMap[event.EventID] = RoleSameDestination
	}

	for _, event := range fileEvents {
		roleMap[event.EventID] = RoleSameFile
	}

	for _, event := range contextBefore {
		roleMap[event.EventID] = RoleBeforeContext
	}

	for _, event := range contextAfter{
		roleMap[event.EventID] = RoleAfterContext
	}

	roleMap[mainEvent.EventID] = RoleMain // устанавливаем main_event главному событию

	// map уникальных событий (по event_id)
	allUniqueEventsMap := make(map[string]*Event)
	allUniqueEventsMap[mainEvent.EventID] = mainEvent

	for _, event := range userEvents {
		allUniqueEventsMap[event.EventID] = event
	}

	for _, event := range destinationEvents {
		allUniqueEventsMap[event.EventID] = event
	}

	for _, event := range fileEvents {
		allUniqueEventsMap[event.EventID] = event
	}

	for _, event := range contextBefore {
		allUniqueEventsMap[event.EventID] = event
	}

	for _, event := range contextAfter {
		allUniqueEventsMap[event.EventID] = event
	}

	// собираем срез []TimelineItem
	timelineItems := make([]TimelineItem, 0, len(allUniqueEventsMap))
	for _, event := range allUniqueEventsMap {
		var fileName, destination, severity *string
		if (event.FileName != nil) {
			fileName = event.FileName
		}
		if (event.Destination != nil) {
			destination = event.Destination
		}
		if (event.Severity != nil) {
			severity = event.Severity
		}

		timelineItems = append(timelineItems, TimelineItem{
			Timestamp: event.TimeStamp,
			EventID:     event.EventID,
			Role:        roleMap[event.EventID],
			UserID:      event.UserID,
			Action:      event.Action,
			FileName:    fileName,
			Destination: destination,
			Severity:    severity,	
		})
	}

	// Сортировка по времени, при равенстве времени сортируем по event_id
	sort.Slice(timelineItems, func(i, j int) bool {
		time_i, _ := time.Parse(time.RFC3339, timelineItems[i].Timestamp)
		time_j, _ := time.Parse(time.RFC3339, timelineItems[j].Timestamp)
		if (time_i.Equal(time_j)) {
			return timelineItems[i].EventID < timelineItems[j].EventID
		}
		return time_i.Before(time_j)
	})

	return timelineItems

}

func BuildSummary(event *Event) string {
	var summary strings.Builder
	summary.WriteString(event.UserID)
	if (event.FileName != nil) {
		summary.WriteString(" ")
		summary.WriteString(*event.FileName)
	}
	return summary.String()
}

func FindIDs(events []*Event) []string {
	var ids []string
	for _, event := range events {
		ids = append(ids, event.EventID)
	}
	return ids
}

func WriteSummaryText(mainEvent *Event) string {
	var summary strings.Builder
	summary.WriteString("Пользователь ")
	summary.WriteString(fmt.Sprintf("*%s*", mainEvent.UserID))
	summary.WriteString("совершил действие ")
	summary.WriteString(fmt.Sprintf("*%s*", mainEvent.Action))
	if (mainEvent.FileName != nil) {
		summary.WriteString("с файлом ")
		summary.WriteString(fmt.Sprintf("*%s*", *mainEvent.FileName))
	}
	if (mainEvent.FileName != nil) {
		summary.WriteString("в адрес ")
		summary.WriteString(fmt.Sprintf("*%s*", *mainEvent.Destination))
	}
	summary.WriteString("\n\n")
	return summary.String()
}

func GenerateMarkdownCard(mainEvent *Event, answer *Answer, index Index) string {
	var markdownnContent strings.Builder
	markdownnContent.WriteString("# Карточка инцидента\n\n")
	markdownnContent.WriteString(fmt.Sprintf("__ID инцидента:__ %s\n\n", answer.IncidentID))

	markdownnContent.WriteString("## Краткое резюме ##\n\n")
	markdownnContent.WriteString(WriteSummaryText(mainEvent))

	return markdownnContent.String()
}