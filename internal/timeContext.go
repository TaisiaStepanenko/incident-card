package internal

import (
	"fmt"
	"sort"
	"time"
)

func GetEventsInTimeRange(events []Event, mainEventTime, beforeEvent, afterEvent string) ([]*Event, error) {
	var eventsInRange []*Event
	var beforeMainTime, afterMainTime time.Duration
	var err error

	if (beforeEvent != "") {
		beforeMainTime, err = time.ParseDuration(beforeEvent) 
		if (err != nil) {
			return nil, fmt.Errorf("Задан неверный формат --before: %v", err)
		}
	}

	if (afterEvent != "") {
		afterMainTime, err = time.ParseDuration(afterEvent)
		if (err != nil) {
			return nil, fmt.Errorf("Задан неверный формат --after: %v", err)
		}
	}

	mainTime, err := time.Parse(time.RFC3339, mainEventTime)
	if (err != nil) {
		return nil, fmt.Errorf("Ошибка при парсинге времени главного события: %v", err)
	}
	startTime := mainTime.Add(-beforeMainTime)
	endTime := mainTime.Add(afterMainTime)


	for i := range events {
		event := &events[i]
		eventTime, err := time.Parse(time.RFC3339, event.TimeStamp)
		if (err != nil) {
			continue
		}

		if ((eventTime.Equal(startTime) || eventTime.After(startTime)) && 
			(eventTime.Equal(endTime) || eventTime.Before(endTime))) {
				eventsInRange = append(eventsInRange, event)
			}
	}

	// сортировка по времени, от более ранних событий к более поздним
	sort.Slice(eventsInRange, func(i, j int) bool {
		time_i, _ := time.Parse(time.RFC3339, eventsInRange[i].TimeStamp)
		time_j, _ := time.Parse(time.RFC3339, eventsInRange[j].TimeStamp)
		return time_i.Before(time_j)
	})

	return eventsInRange, nil
}