package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/TaisiaStepanenko/incident-card/internal"
)

func main() {  

	args := os.Args
	if (len(args) < 3) {
		fmt.Println("Использование: incident-card --events <file> --event-id <id> --before <dur> --after <dur> --request <json-file> --out <md-file> --json <json-file>")
		return
	}

	var eventsFile, eventId, beforeEvent, afterEvent, outFile, requestFile, jsonFile string

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
		case "--request":
			requestFile = args[i+1]
		case "--json":
			jsonFile = args[i+1]
		}
	}

	var req internal.Request

	if (requestFile != "") {
		reqData, err := os.ReadFile(requestFile)
		if (err != nil) {
			log.Fatalf("Ошибка при чтении файла запроса %s: %v", requestFile, err)
		}
		err = json.Unmarshal(reqData, &req)
		if (err != nil) {
			log.Fatalf("Ошибка парсинга JSON-запроса: %v", err)
		}
	}
	
	// Задаём значения из JSON, но только в том случае, если данные ещё не были заполнены CLI (приоритет) 
	if (req.MainEventID != "" && eventId == "") {
		eventId = req.MainEventID
	}
	if (req.WindowBefore != "" && beforeEvent == "") {
		beforeEvent = req.WindowBefore
		}
	if (req.WindowAfter != "" && afterEvent == "") {
		afterEvent = req.WindowAfter
	}

	if (req.IncludeSameUser == nil) {
		defaultTrue := true
		req.IncludeSameUser = &defaultTrue
	}
	if (req.IncludeSameFile == nil) {
		defaultFalse := false
		req.IncludeSameFile = &defaultFalse
	}
	if (req.IncludeSameDestination == nil) {
		defaultFalse := false
		req.IncludeSameDestination = &defaultFalse
	}
		
	limit := req.MaxEventsPerSection
	if (limit == 0) {
		limit = 50
	}

	// Проверяем передали файл с событиями, без него не можем продолжать работу
	if (eventsFile == "") {
		fmt.Println("Необходимо передать --events")
		return
	}
	if (eventId == "") {
		fmt.Println("Необходимо передать --events-id или передать его в JSON-файле")
		return
	}

	events, eventsLink, err := internal.ReadEvents(eventsFile)
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

	// Создаём запрос в соответствии с структурой Request
	req = internal.Request{
		IncidentID: req.IncidentID,
		MainEventID: eventId,
		WindowBefore: beforeEvent,      
		WindowAfter:  afterEvent,
		IncludeSameUser: req.IncludeSameUser,
		IncludeSameFile: req.IncludeSameFile, 
		IncludeSameDestination: req.IncludeSameDestination,
		MaxEventsPerSection:  limit,  
	}

	answer := internal.BuildAnswer(mainEvent, index, events, eventsLink, req)

	if (outFile != "") {
		markdownCard := internal.GenerateMarkdownCard(mainEvent, &answer, index, limit)

		outFileOpen, err := os.OpenFile(outFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		if (err != nil) {
			log.Fatalf("Не удалось открыть файл %s: %v", outFile, err)
		}
		defer outFileOpen.Close()

		_, err = outFileOpen.WriteString(markdownCard)
		if (err != nil) {
			log.Fatalf("Ошибка записи в файл Markdown-карточки: %v", err)
		}
		fmt.Printf("Markdown записан в файл %s\n", outFile)
	} else {
		fmt.Println("Для сохранения отчёта в виде Markdown-карточки необходимо передать --out")
	}

	if (jsonFile != "") {
		jsonFileOpen, err := os.OpenFile(jsonFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		if (err != nil) {
			log.Fatalf("Не удалось открыть файл %s: %v", jsonFile, err)
		}
		defer jsonFileOpen.Close()

		jsonData, err := json.Marshal(answer)
		if (err != nil) {
			log.Fatalf("Ошибка сериализации JSON: %v", err)
		}
		err = os.WriteFile(jsonFile, jsonData, 0666)
		if (err != nil) {
			log.Fatalf("Ошибка записи JSON: %v", err)
		}
		fmt.Printf("JSON сохранён в файл %s\n", jsonFile)
	} else {
		fmt.Println("Для сохранения отчёта в JSON-файл необходимо передать --json")
	}

}