package main

import (
	"fmt"
	"log"
	"os"

	"github.com/TaisiaStepanenko/incident-card/internal"
)

func main() {
	events, err := internal.ReadEvents("testdata/eventsList.jsonl")
	if err != nil {
		log.Fatalf("Ошибка: %v\n", err)
	}
	fmt.Printf("Прочитано %d событий\n", len(events))   

	args := os.Args
	if (len(args) < 5) {
		fmt.Println("Использование: incident-card --events <file> --event-id <id>")
		return
	}

	var eventsFile, eventId string

	for i := 1; i < len(args); i += 2 {
		if (args[i] == "--events") {
			eventsFile = args[i+1]
		} else if (args[i] == "--event-id") {
			eventId = args[i+1]
		}
	}

	if (eventsFile == "" || eventId == "") {
		fmt.Println("Необходимо передать --events и --events-id")
		return
	}

	index := internal.BuildIndex(events)

	mainEvent, isExist := index.GetEvent(eventId)
	if (!isExist) {
		fmt.Printf("Событие с данным event_id (event_id = %s) не существует\n", eventId)
		return 
	}

	userEvents := index.GetEventByUser(mainEvent.UserID)
	fmt.Printf("Найдено %d событий пользователя\n", len(userEvents))
	for _, event := range userEvents {
		fmt.Printf("Событие %s, Action: %s\n", event.EventID, event.Action)
	}

	if (mainEvent.FileID != nil) {
		countEvents := len(index.GetEventByFile(*mainEvent.FileID))
		fmt.Printf("Найдено %d событий с файлом %s\n", countEvents, *mainEvent.FileID)
	}


}