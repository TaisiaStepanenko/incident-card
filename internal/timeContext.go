package internal

import (
	"fmt"
	"sort"
	"time"
)

func GetEventsInTimeRange(index Index, mainEventTime, beforeEvent, afterEvent string) ([]*Event, error) {
	var beforeMainTime, afterMainTime time.Duration
	var err error

	if beforeEvent != "" {
		beforeMainTime, err = time.ParseDuration(beforeEvent)
		if err != nil {
			return nil, fmt.Errorf("Задан неверный формат --before: %v", err)
		}
	}

	if afterEvent != "" {
		afterMainTime, err = time.ParseDuration(afterEvent)
		if err != nil {
			return nil, fmt.Errorf("Задан неверный формат --after: %v", err)
		}
	}

	mainTime, err := time.Parse(time.RFC3339, mainEventTime)
	if err != nil {
		return nil, fmt.Errorf("Ошибка при парсинге времени главного события: %v", err)
	}
	startTime := mainTime.Add(-beforeMainTime)
	endTime := mainTime.Add(afterMainTime)

	// Поиск первого события >= startTime
	left := sort.Search(len(index.TimeIndex), func(i int) bool {
		t, _ := time.Parse(time.RFC3339, index.TimeIndex[i].TimeStamp)
		return !t.Before(startTime)
	})

	// Поиск первого события < endTime
	right := sort.Search(len(index.TimeIndex), func(i int) bool {
		t, _ := time.Parse(time.RFC3339, index.TimeIndex[i].TimeStamp)
		return t.After(endTime)
	})

	if (left > right) {
		return []*Event{}, nil
	}

	return index.TimeIndex[left:right], nil
}
