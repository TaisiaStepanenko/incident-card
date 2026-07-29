package internal

import (
	"sort"
	"time"
)

type Index struct {
	EventsById []*Event
	UserIdGroup        map[string][]*Event
	FileIdGroup        map[string][]*Event
	DestinationIdGroup map[string][]*Event
	TimeIndex 		   []*Event		// отсортированный по времени
}

func BuildIndex(events []Event) Index {

	idx := Index{
		EventsById:       make([]*Event, 0, len(events)),
		UserIdGroup:        make(map[string][]*Event, len(events)),
		FileIdGroup:        make(map[string][]*Event, len(events)),
		DestinationIdGroup: make(map[string][]*Event, len(events)),
		TimeIndex: 			make([]*Event, 0, len(events)),	
	}

	for i := range events {
		event := &events[i]
		
		idx.EventsById = append(idx.EventsById, event)

		// Группировка по user_id
		idx.UserIdGroup[event.UserID] = append(idx.UserIdGroup[event.UserID], event)

		// Группировка по file_id
		if event.FileID != "" {
			idx.FileIdGroup[event.FileID] = append(idx.FileIdGroup[event.FileID], event)
		}

		// Группировка по destination_id
		if event.DestinationID != "" {
			idx.DestinationIdGroup[event.DestinationID] = append(idx.DestinationIdGroup[event.DestinationID], event)
		}

		idx.TimeIndex = append(idx.TimeIndex, event)
	}

	sort.Slice(idx.EventsById, func(i, j int) bool {
		return idx.EventsById[i].EventID < idx.EventsById[j].EventID
	})

	sort.Slice(idx.TimeIndex, func(i, j int) bool {
		time_i, _ := time.Parse(time.RFC3339, idx.TimeIndex[i].TimeStamp)
		time_j, _ := time.Parse(time.RFC3339, idx.TimeIndex[j].TimeStamp)

		if time_i.Equal(time_j) {
			return idx.TimeIndex[i].EventID < idx.TimeIndex[j].EventID
		}
		return time_i.Before(time_j)
	})

	return idx
}

// функция возвращает Event по event_id
func (idx Index) GetEvent(eventId string) (*Event, bool) {
	i := sort.Search(len(idx.EventsById), func(i int) bool {
		return idx.EventsById[i].EventID >= eventId
	})

	if (i < len(idx.EventsById) && idx.EventsById[i].EventID == eventId) {
		return idx.EventsById[i], true
	}
	return nil, false
}

// функция возвращает все Event пользователя с user_id
func (idx Index) GetEventByUser(userId string) []*Event {
	if (idx.UserIdGroup == nil) {
		return nil
	} 
	return idx.UserIdGroup[userId]
}

// функция возвращает все Event пользователя с file_id
func (idx Index) GetEventByFile(fileId string) []*Event {
	if (idx.FileIdGroup == nil) {
		return nil
	} 
	return idx.FileIdGroup[fileId]
}

// функция возвращает все Event пользователя с destination_id
func (idx Index) GetEventByDestination(destinationId string) []*Event {
	if (idx.DestinationIdGroup == nil) {
		return nil
	} 
	return idx.DestinationIdGroup[destinationId]
}
