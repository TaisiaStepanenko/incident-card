package main

import (
	"fmt"
	"log"
	"os"

	"github.com/TaisiaStepanenko/incident-card/internal"
)

func main() {  

	args := os.Args
	if (len(args) < 5) {
		fmt.Println("Использование: incident-card --events <file> --event-id <id> --before <dur> --after <dur> --out <md-file>")
		return
	}

	var eventsFile, eventId, beforeEvent, afterEvent, outFile string

	for i := 1; i < len(args); i += 2 {
		switch args[i] {
		case "--events":
			eventsFile = args[i+1]
		case "--event-id":
			eventId = args[i+1]
		case "--before":
			beforeEvent = args[i+1]
		case "--after":
			afterEvent = args[i+1]
		case "--out":
			outFile = args[i+1]
		}
	}

	if (eventsFile == "" || eventId == "") {
		fmt.Println("Необходимо передать --events и --events-id")
		return
	}

	events, err := internal.ReadEvents(eventsFile)
	if err != nil {
		log.Fatalf("Ошибка: %v\n", err)
	}
	fmt.Printf("Прочитано %d событий\n", len(events)) 

	index := internal.BuildIndex(events)

	mainEvent, isExist := index.GetEvent(eventId)
	if (!isExist) {
		fmt.Printf("Событие с данным event_id (event_id = %s) не существует\n", eventId)
		return 
	}

	bool_true := true
	true_ptr := &bool_true
	bool_false := false
	false_ptr := &bool_false
	limit := 50
	// Создаём запрос в соответствии с структурой Request, некоторые поля пока заполняем вручную
	req := internal.Request{
		IncidentID: "inc_1",
		MainEventID: eventId,
		WindowBefore: beforeEvent,      
		WindowAfter:  afterEvent,
		IncludeSameUser: true_ptr,
		IncludeSameFile: false_ptr, 
		IncludeSameDestination: true_ptr,
		MaxEventsPerSection:  limit,  
	}

	answer := internal.BuildAnswer(mainEvent, index, events, req)

	

}