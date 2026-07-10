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

func BuildAnswer(mainEvent *Event, index Index, events []Event, eventsLink []LinkInFile, req Request, rules []Rule) Answer {

	// Сбор событий по временному контексту 
	timeEvents, err := GetEventsInTimeRange(events, mainEvent.TimeStamp, req.WindowBefore, req.WindowAfter)
	if err != nil {
		log.Fatalf("Ошибка при получении временного контекста событий: %v", err)
	}

	// События пользователя главного события (если есть в запросе)
	var userEvents []*Event
	if (req.IncludeSameUser != nil && *req.IncludeSameUser) {
		userEvents = index.GetEventByUser(mainEvent.UserID)
	}

	// События с файлом главного события (если есть в запросе)
	var fileEvents []*Event
	if (req.IncludeSameFile != nil && *req.IncludeSameFile && mainEvent.FileID != nil) {
		fileEvents = index.GetEventByFile(*mainEvent.FileID)
	}

	// События адресата главного события (если есть в запросе)
	var destinationEvents []*Event
	if (req.IncludeSameDestination != nil && *req.IncludeSameDestination && mainEvent.DestinationID != nil) {
		destinationEvents = index.GetEventByDestination(*mainEvent.DestinationID)
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


	timelineItems, linksTotimelineItems := BuildTimeline(mainEvent, contextBefore, contextAfter, userEvents, fileEvents, destinationEvents, eventsLink)
	
	if (len(timelineItems) > limit) {
		timelineItems = timelineItems[:limit]
		linksTotimelineItems = linksTotimelineItems[:limit]
	}


	summary := BuildSummary(mainEvent)

	contextBeforeIds := FindIDs(contextBefore)
	contextAfterIds := FindIDs(contextAfter)
	userEventsIds := FindIDs(userEvents)
	fileEventsIds := FindIDs(fileEvents)
	destinationEventsIds := FindIDs(destinationEvents)

	suspicious := CheckRules(mainEvent, rules)

	return Answer{
		IncidentID: req.IncidentID,
		MainEvent: MainEvent{
			EventID: mainEvent.EventID,
			Action: mainEvent.Action,
		},
		Summary: summary,
		ContextBefore: contextBeforeIds,
		ContextAfter: contextAfterIds,
		SameUserEvents: userEventsIds,
		SameFileEvents: fileEventsIds,
		SameDestinationEvents: destinationEventsIds,
		TimeLine: timelineItems,
		SuspiciousFactors: suspicious,
		LinksToTheOriginalEvents: linksTotimelineItems,
	}

}

func BuildTimeline(mainEvent *Event, contextBefore, contextAfter, userEvents, fileEvents, destinationEvents []*Event, eventsLink []LinkInFile) ([]TimelineItem, []LinkInFile) {
	
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
	linksTotimelineItems := make([]LinkInFile, 0, len(allUniqueEventsMap))
	for _, event := range allUniqueEventsMap {
		var fileName, destination, severity string
		if (event.FileName != nil) {
			fileName = *event.FileName
		}
		if (event.Destination != nil) {
			destination = *event.Destination
		}
		if (event.Severity != nil) {
			severity = *event.Severity
		}

		timelineItems = append(timelineItems, TimelineItem{
			Timestamp:   event.TimeStamp,
			EventID:     event.EventID,
			Role:        roleMap[event.EventID],
			UserID:      event.UserID,
			Action:      event.Action,
			FileName:    fileName,
			Destination: destination,
			Severity:    severity,	
		})

		for _, link := range eventsLink {
			if (link.EventID == event.EventID) {
				linksTotimelineItems = append(linksTotimelineItems, LinkInFile{
					EventID: event.EventID,
					FileName: link.FileName,
					FileLine: link.FileLine,
				})
			}
		}
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

	// Сортировка ссылок по event_id
	sort.Slice(linksTotimelineItems, func(i, j int) bool {
		return linksTotimelineItems[i].EventID < linksTotimelineItems[j].EventID
	})

	return timelineItems, linksTotimelineItems

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
	summary.WriteString(fmt.Sprintf("***%s***", mainEvent.UserID))
	summary.WriteString(" совершил действие ")
	summary.WriteString(fmt.Sprintf("***%s***", mainEvent.Action))
	if (mainEvent.FileName != nil) {
		summary.WriteString(" с файлом ")
		summary.WriteString(fmt.Sprintf("***%s***", *mainEvent.FileName))
	}
	if (mainEvent.Destination != nil) {
		summary.WriteString(" в адрес ")
		summary.WriteString(fmt.Sprintf("***%s***", *mainEvent.Destination))
	}
	summary.WriteString(".\n\n")
	return summary.String()
}

func GenerateMarkdownCard(mainEvent *Event, answer *Answer, index Index, maxEventsPerSection int) string {
	var markdownnContent strings.Builder
	markdownnContent.WriteString("# Карточка инцидента\n\n")
	markdownnContent.WriteString(fmt.Sprintf("__ID инцидента:__ %s\n\n", answer.IncidentID))

	markdownnContent.WriteString("## Краткое резюме ##\n\n")
	markdownnContent.WriteString(WriteSummaryText(mainEvent))

	markdownnContent.WriteString("## Главное событие ##\n\n")
	markdownnContent.WriteString(fmt.Sprintf("- __Event ID:__ %s\n", answer.MainEvent.EventID))
	markdownnContent.WriteString(fmt.Sprintf("- __Action:__ %s\n", answer.MainEvent.Action))

	markdownnContent.WriteString("## Контекст до события ##\n\n")
	PrintSectionEvents(answer.ContextBefore, &markdownnContent)

	markdownnContent.WriteString("## Контекст после события ##\n\n")
	PrintSectionEvents(answer.ContextAfter, &markdownnContent)

	markdownnContent.WriteString("## События того же пользователя ##\n\n")
	PrintSectionEvents(answer.SameUserEvents, &markdownnContent)

	markdownnContent.WriteString("## События с тем же файлом ##\n\n")
	PrintSectionEvents(answer.SameFileEvents, &markdownnContent)

	markdownnContent.WriteString("## События с тем же адресатом ##\n\n")
	PrintSectionEvents(answer.SameDestinationEvents, &markdownnContent)

	markdownnContent.WriteString("## Временная шкала ##\n\n")
	if (len(answer.TimeLine) == 0) {
		markdownnContent.WriteString("Подходящих для данного раздела событий не найдено\n\n")
	} else {
		if (len(answer.TimeLine) > maxEventsPerSection) {
			markdownnContent.WriteString(fmt.Sprintf("Количество записей превысило максимально возможное значение. В таблице приведены первые %d событий из %d.\n\n", maxEventsPerSection, len(answer.TimeLine)))
		}
		markdownnContent.WriteString("| Время | Событие | Пользователь | Действие | Файл | Адресат | Важность | Роль |\n")
		markdownnContent.WriteString("|:---|:---|:---|:---|:---|:---|:---:|:---:|\n")
		for i, timelineItem := range answer.TimeLine {
			if (i < maxEventsPerSection) {
				WriteTableRaw(&timelineItem, &markdownnContent)
			} else {
				break
			}
		}
		markdownnContent.WriteString("\n")
	}

	markdownnContent.WriteString("## Подозрительные факторы ##\n\n")
	PrintSectionEvents(answer.SuspiciousFactors, &markdownnContent)

	markdownnContent.WriteString("## Ссылки на исходные события ##\n\n")
	if (len(answer.LinksToTheOriginalEvents) == 0) {
		markdownnContent.WriteString("Подходящих для данного раздела событий не найдено\n\n")
	} else {
		for _, link := range answer.LinksToTheOriginalEvents {
			markdownnContent.WriteString(fmt.Sprintf("- ___%s___: файл __%s__ строка __%d__\n", link.EventID, link.FileName, link.FileLine))
		}
	}

	return markdownnContent.String()
}

func PrintSectionEvents(ids []string, markdownnContent *strings.Builder) {
	if (len(ids) == 0) {
		markdownnContent.WriteString("Подходящих для данного раздела событий не найдено.\n\n")
		return
	} else {
		for _, id := range ids {
			markdownnContent.WriteString(fmt.Sprintf("- %s\n", id)) 
		}
		markdownnContent.WriteString("\n")
	}
}

func WriteTableRaw(item *TimelineItem, markdownnContent *strings.Builder) {
	for i := 0; i <= 7; i++ {
		switch i {
			case 0:
				if(item.Timestamp != "") {
					markdownnContent.WriteString(fmt.Sprintf("|%s", item.Timestamp))
				} else {
					markdownnContent.WriteString("|-")
				}
			case 1:
				if(item.EventID != "") {
					markdownnContent.WriteString(fmt.Sprintf("|%s", item.EventID))
				} else {
					markdownnContent.WriteString("|-")
				}
			case 2:
				if(item.UserID != "") {
					markdownnContent.WriteString(fmt.Sprintf("|%s", item.UserID))
				} else {
					markdownnContent.WriteString("|-")
				}
			case 3:
				if(item.Action != "") {
					markdownnContent.WriteString(fmt.Sprintf("|%s", item.Action))
				} else {
					markdownnContent.WriteString("|-")
				}
			case 4:
				if(item.FileName != "") {
					markdownnContent.WriteString(fmt.Sprintf("|%s", item.FileName))
				} else {
					markdownnContent.WriteString("|-")
				}
			case 5:
				if(item.Destination != "") {
					markdownnContent.WriteString(fmt.Sprintf("|%s", item.Destination))
				} else {
					markdownnContent.WriteString("|-")
				}
			case 6:
				if(item.Severity != "") {
					markdownnContent.WriteString(fmt.Sprintf("|%s", item.Severity))
				} else {
					markdownnContent.WriteString("|-")
				}
			case 7:
				if(item.Role != "") {
					markdownnContent.WriteString(fmt.Sprintf("|%s", item.Role))
				} else {
					markdownnContent.WriteString("|-")
				}
		}
	}

	markdownnContent.WriteString("|\n")

}
