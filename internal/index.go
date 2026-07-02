package internal

func buildIndex(events []Event) map[string]Event {
	eventIndex := make(map[string]Event)

	for _, event := range events {
		_, isExist := eventIndex[event.EventID]
		if isExist {
			continue
		}
		eventIndex[event.EventID] = event
	}

	return eventIndex
}