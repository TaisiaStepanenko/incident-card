package internal

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func MakeLimitSlice(events []*Event, limit int) []*Event {
	if len(events) > limit {
		return events[:limit]
	}
	return events
}

func BuildAnswer(mainEvent *Event, index Index, fileName string, req Request, rules []Rule) (Answer, error) {

	if (req.MaxEventsPerSection < 1 || req.MaxEventsPerSection > 1000) {
		return Answer{}, fmt.Errorf("max_events_per_section должен быть в диапозоне [1, 1000]")
	}
	
	// Сбор событий по временному контексту
	timeEvents, err := GetEventsInTimeRange(index, mainEvent.TimeStamp, req.WindowBefore, req.WindowAfter)
	if err != nil {
		return Answer{}, fmt.Errorf("Ошибка при получении временного контекста событий: %v", err)
	}

	// События пользователя главного события (если есть в запросе)
	var userEvents []*Event
	if req.IncludeSameUser != nil && *req.IncludeSameUser {
		userEvents = index.GetEventByUser(mainEvent.UserID)
		sortEventsByTime(userEvents)
	}

	// События с файлом главного события (если есть в запросе)
	var fileEvents []*Event
	if req.IncludeSameFile != nil && *req.IncludeSameFile && mainEvent.FileID != "" {
		fileEvents = index.GetEventByFile(mainEvent.FileID)
		sortEventsByTime(fileEvents)
	}

	// События адресата главного события (если есть в запросе)
	var destinationEvents []*Event
	if req.IncludeSameDestination != nil && *req.IncludeSameDestination && mainEvent.DestinationID != "" {
		destinationEvents = index.GetEventByDestination(mainEvent.DestinationID)
		sortEventsByTime(destinationEvents)
	}

	// Устанавливаем ограничение размера разделов (по умолчанию 50)
	limit := req.MaxEventsPerSection
	if limit == 0 {
		limit = 50
	}

	var contextBefore, contextAfter []*Event
	mainTime, _ := time.Parse(time.RFC3339, mainEvent.TimeStamp)
	for _, event := range timeEvents {
		eventTime, _ := time.Parse(time.RFC3339, event.TimeStamp)
		if eventTime.Before(mainTime) {
			contextBefore = append(contextBefore, event)
		} else if eventTime.After(mainTime) {
			contextAfter = append(contextAfter, event)
		}
	}

	timelineItems := BuildTimeline(mainEvent, contextBefore, contextAfter, userEvents, fileEvents, destinationEvents)
	totalTimelineEvents := len(timelineItems) 	// полное количество до учечения

	if (len(timelineItems) > limit) {
		// Проверяем, есть ли главное событие в первых limit элементах
		mainEventIncluded := false
		var mainIndex int
		for i := 0; i < limit; i++ {
			if (timelineItems[i].EventID == mainEvent.EventID) {
				mainEventIncluded = true
				mainIndex = i
				break
			}
		}

		// Если главное событие не входит в первые limit, ищем его индекс в оставшейся части
		if (!mainEventIncluded) {
			for i := limit; i < len(timelineItems); i++ {
				if (timelineItems[i].EventID == mainEvent.EventID) {
					mainIndex = i
					break
				}
			}

			// Если главное событие найдено, меняем его с последним элементом среза
			if (mainIndex >= limit) {
				timelineItems[limit-1], timelineItems[mainIndex] = timelineItems[mainIndex], timelineItems[limit-1] 
			}
		}

		// Обрезаем до limit
		timelineItems = timelineItems[:limit]
		
		sort.Slice(timelineItems, func(i, j int) bool {
		time_i, _ := time.Parse(time.RFC3339, timelineItems[i].Timestamp)
		time_j, _ := time.Parse(time.RFC3339, timelineItems[j].Timestamp)
		if time_i.Equal(time_j) {
			return timelineItems[i].EventID < timelineItems[j].EventID
		}
		return time_i.Before(time_j)
	})
		
	}


	linksTotimeline := make([]LinkInFile, 0, len(timelineItems))
	for _, item := range timelineItems {
		event, isExist := index.GetEvent(item.EventID)
		if (isExist && event.LineNumber > 0) {
			linksTotimeline = append(linksTotimeline, LinkInFile{
				EventID: item.EventID,
				FileName: fileName,
				FileLine: event.LineNumber,
			})
		}
	}

	summary := BuildSummary(mainEvent)

	contextBefore = LimitClosestBefore(contextBefore, limit)
	contextAfter = LimitClosestAfter(contextAfter, limit)
	userEvents = MakeLimitSlice(userEvents, limit)
	fileEvents = MakeLimitSlice(fileEvents, limit)
	destinationEvents = MakeLimitSlice(destinationEvents, limit)

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
			Action:  mainEvent.Action,
		},
		Summary:                  summary,
		ContextBefore:            contextBeforeIds,
		ContextAfter:             contextAfterIds,
		SameUserEvents:           userEventsIds,
		SameFileEvents:           fileEventsIds,
		SameDestinationEvents:    destinationEventsIds,
		TimeLine:                 timelineItems,
		TotalTimelineEvents:	  totalTimelineEvents,
		SuspiciousFactors:        suspicious,
		LinksToTheOriginalEvents: linksTotimeline,
	}, nil

}

func BuildTimeline(mainEvent *Event, contextBefore, contextAfter, userEvents, fileEvents, destinationEvents []*Event) []TimelineItem {

	// Собираем все события в один срез
	allEvents := make([]*Event, 0, len(contextBefore)+len(contextAfter)+len(userEvents)+len(fileEvents)+len(destinationEvents)+1)
	allEvents = append(allEvents, mainEvent)
	allEvents = append(allEvents, contextBefore...)
	allEvents = append(allEvents, contextAfter...)
	allEvents = append(allEvents, userEvents...)
	allEvents = append(allEvents, fileEvents...)
	allEvents = append(allEvents, destinationEvents...)

	// Сортировка по времени, при равенстве времени сортируем по event_id
	sort.Slice(allEvents, func(i, j int) bool {
		time_i, _ := time.Parse(time.RFC3339, allEvents[i].TimeStamp)
		time_j, _ := time.Parse(time.RFC3339, allEvents[j].TimeStamp)
		if time_i.Equal(time_j) {
			return allEvents[i].EventID < allEvents[j].EventID
		}
		return time_i.Before(time_j)
	})

	// Удаляем дубликаты
	seen := make(map[string]bool)
	uniqueEvents := make([]*Event, 0, len(allEvents))
	for _, event := range allEvents {
		if (!seen[event.EventID]) {
			seen[event.EventID] = true
			uniqueEvents = append(uniqueEvents, event)
		}
	}

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

	for _, event := range contextAfter {
		roleMap[event.EventID] = RoleAfterContext
	}

	roleMap[mainEvent.EventID] = RoleMain // устанавливаем main_event главному событию


	// собираем срез []TimelineItem
	timelineItems := make([]TimelineItem, 0, len(uniqueEvents))
	for _, event := range uniqueEvents {
		var fileName, destination, severity string
		if event.FileName != "" {
			fileName = event.FileName
		}
		if event.Destination != "" {
			destination = event.Destination
		}
		if event.Severity != "" {
			severity = event.Severity
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
	}

	return timelineItems

}

// Оставляет последние limit событий (ближайшие к главному по времени)
func LimitClosestBefore(events []*Event, limit int) []*Event {
	if (len(events) <= limit) {
		return events
	}
	return events[len(events)-limit:]
}

// Оставляет первые limit событий (ближайшие к главному по времени)
func LimitClosestAfter(events []*Event, limit int) []*Event {
	if (len(events) <= limit) {
		return events
	}
	return events[:limit]
}


func BuildSummary(event *Event) string {
	var summary strings.Builder
	summary.WriteString(event.UserID)
	if event.FileName != "" {
		summary.WriteString(" ")
		summary.WriteString(event.FileName)
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
	summary.WriteString(fmt.Sprintf("***%s***", escapeMarkdownText(mainEvent.UserID)))
	summary.WriteString(" совершил действие ")
	summary.WriteString(fmt.Sprintf("***%s***", escapeMarkdownText(mainEvent.Action)))
	if mainEvent.FileName != "" {
		summary.WriteString(" с файлом ")
		summary.WriteString(fmt.Sprintf("***%s***", escapeMarkdownText(mainEvent.FileName)))
	}
	if mainEvent.Destination != "" {
		summary.WriteString(" в адрес ")
		summary.WriteString(fmt.Sprintf("***%s***", escapeMarkdownText(mainEvent.Destination)))
	}
	summary.WriteString(".\n\n")
	return summary.String()
}

func GenerateMarkdownCard(mainEvent *Event, answer *Answer, index Index, maxEventsPerSection int) string {
	var markdownnContent strings.Builder
	markdownnContent.WriteString("# Карточка инцидента\n\n")
	markdownnContent.WriteString(fmt.Sprintf("__ID инцидента:__ %s\n\n", escapeMarkdownText(answer.IncidentID)))

	markdownnContent.WriteString("## Краткое резюме ##\n\n")
	markdownnContent.WriteString(WriteSummaryText(mainEvent))

	markdownnContent.WriteString("## Главное событие ##\n\n")
	markdownnContent.WriteString(fmt.Sprintf("- __Event ID:__ %s\n", escapeMarkdownText(answer.MainEvent.EventID)))
	markdownnContent.WriteString(fmt.Sprintf("- __Action:__ %s\n", escapeMarkdownText(answer.MainEvent.Action)))

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
	if len(answer.TimeLine) == 0 {
		markdownnContent.WriteString("Подходящих для данного раздела событий не найдено\n\n")
	} else {
		if answer.TotalTimelineEvents > maxEventsPerSection {
			markdownnContent.WriteString(fmt.Sprintf("Количество записей превысило максимально возможное значение (truncated). В таблице приведены первые %d событий из %d.\n\n", maxEventsPerSection, answer.TotalTimelineEvents))
		}
		markdownnContent.WriteString("| Время | Событие | Пользователь | Действие | Файл | Адресат | Важность | Роль |\n")
		markdownnContent.WriteString("|:---|:---|:---|:---|:---|:---|:---:|:---:|\n")
		for i, timelineItem := range answer.TimeLine {
			if i < maxEventsPerSection {
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
	if len(answer.LinksToTheOriginalEvents) == 0 {
		markdownnContent.WriteString("Подходящих для данного раздела событий не найдено\n\n")
	} else {
		for _, link := range answer.LinksToTheOriginalEvents {
			markdownnContent.WriteString(fmt.Sprintf("- ___%s___: файл __%s__ строка __%d__\n", escapeMarkdownText(link.EventID), escapeMarkdownText(link.FileName), link.FileLine))
		}
	}

	return markdownnContent.String()
}

func PrintSectionEvents(ids []string, markdownnContent *strings.Builder) {
	if len(ids) == 0 {
		markdownnContent.WriteString("Подходящих для данного раздела событий не найдено.\n\n")
		return
	} else {
		for _, id := range ids {
			markdownnContent.WriteString(fmt.Sprintf("- %s\n", escapeMarkdownText(id)))
		}
		markdownnContent.WriteString("\n")
	}
}

func WriteTableRaw(item *TimelineItem, markdownnContent *strings.Builder) {

	escapeCell := func(s string) string {
		return escapeMarkdownCell(escapeMarkdownText(s))
	}
	for i := 0; i <= 7; i++ {
		switch i {
		case 0:
			if item.Timestamp != "" {
				markdownnContent.WriteString("|")
				markdownnContent.WriteString(escapeCell(item.Timestamp))
			} else {
				markdownnContent.WriteString("|-")
			}
		case 1:
			if item.EventID != "" {
				markdownnContent.WriteString("|")
				markdownnContent.WriteString(escapeCell(item.EventID))
			} else {
				markdownnContent.WriteString("|-")
			}
		case 2:
			if item.UserID != "" {
				markdownnContent.WriteString("|")
				markdownnContent.WriteString(escapeCell(item.UserID))
			} else {
				markdownnContent.WriteString("|-")
			}
		case 3:
			if item.Action != "" {
				markdownnContent.WriteString("|")
				markdownnContent.WriteString(escapeCell(item.Action))
			} else {
				markdownnContent.WriteString("|-")
			}
		case 4:
			if item.FileName != "" {
				markdownnContent.WriteString("|")
				markdownnContent.WriteString(escapeCell(item.FileName))
			} else {
				markdownnContent.WriteString("|-")
			}
		case 5:
			if item.Destination != "" {
				markdownnContent.WriteString("|")
				markdownnContent.WriteString(escapeCell(item.Destination))
			} else {
				markdownnContent.WriteString("|-")
			}
		case 6:
			if item.Severity != "" {
				markdownnContent.WriteString("|")
				markdownnContent.WriteString(escapeCell(item.Severity))
			} else {
				markdownnContent.WriteString("|-")
			}
		case 7:
			if item.Role != "" {
				markdownnContent.WriteString("|")
				markdownnContent.WriteString(escapeCell(string(item.Role)))
			} else {
				markdownnContent.WriteString("|-")
			}
		}
	}

	markdownnContent.WriteString("|\n")

}

// Экранирование специальных символов для Markdown
func escapeMarkdownText(s string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
        "*", "\\*",
        "#", "\\#",
        "[", "\\[",
        "]", "\\]",
        "(", "\\(",
        ")", "\\)",
	)
	return replacer.Replace(s)
}

// Экранирование символов для ячеек таблицы
func escapeMarkdownCell (s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}

func sortEventsByTime(events []*Event) {
	if (len(events) == 0) {
		return
	}

	sort.Slice(events, func(i, j int) bool {
		time_i, _ := time.Parse(time.RFC3339, events[i].TimeStamp)
		time_j, _ := time.Parse(time.RFC3339, events[j].TimeStamp)	
		if time_i.Equal(time_j) {
			return events[i].EventID < events[j].EventID
		}
		return time_i.Before(time_j)
	})
}