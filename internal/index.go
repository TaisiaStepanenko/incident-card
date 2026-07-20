package internal

type Index struct {
	EventIdIndex       map[string]*Event
	UserIdGroup        map[string][]*Event
	FileIdGroup        map[string][]*Event
	DestinationIdGroup map[string][]*Event
}

func BuildIndex(events []Event) Index {

	idx := Index{
		EventIdIndex:       make(map[string]*Event),
		UserIdGroup:        make(map[string][]*Event),
		FileIdGroup:        make(map[string][]*Event),
		DestinationIdGroup: make(map[string][]*Event),
	}

	for i := range events {
		event := &events[i]
		// Индекс по event_id
		_, isExist := idx.EventIdIndex[event.EventID] // проверка на дубликат
		if isExist {
			continue
		}
		idx.EventIdIndex[event.EventID] = event

		// Группировка по user_id
		idx.UserIdGroup[event.UserID] = append(idx.UserIdGroup[event.UserID], event)

		// Группировка по file_id
		if event.FileID != nil {
			idx.FileIdGroup[*event.FileID] = append(idx.FileIdGroup[*event.FileID], event)
		}

		// Группировка по destination_id
		if event.DestinationID != nil {
			idx.DestinationIdGroup[*event.DestinationID] = append(idx.DestinationIdGroup[*event.DestinationID], event)
		}
	}

	return idx
}

// функция возвращает Event по event_id
func (idx Index) GetEvent(eventId string) (*Event, bool) {
	event, isExist := idx.EventIdIndex[eventId]
	return event, isExist
}

// функция возвращает все Event пользователя с user_id
func (idx Index) GetEventByUser(userId string) []*Event {
	return idx.UserIdGroup[userId]
}

// функция возвращает все Event пользователя с file_id
func (idx Index) GetEventByFile(fileId string) []*Event {
	return idx.FileIdGroup[fileId]
}

// функция возвращает все Event пользователя с destination_id
func (idx Index) GetEventByDestination(destinationId string) []*Event {
	return idx.DestinationIdGroup[destinationId]
}
