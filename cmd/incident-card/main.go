package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/TaisiaStepanenko/incident-card/internal"
	"gopkg.in/yaml.v3"
)

// Использовала gopkg.in/yaml.v3 для парсинга YAML-файлов с правилами подозрительных факторов.
// Стандартная библиотека Go не содержит пакета для работы с YAML, поэтому было принято решение использовать внешнюю библиотеку.

func main() {  

	// Определение подкоманд
	buildCommand := flag.NewFlagSet("build", flag.ExitOnError)
	generateCommand := flag.NewFlagSet("generate", flag.ExitOnError)

	// Флаги команды build
	eventsFile := buildCommand.String("events", "", "JSONL-файл, содержащий информацию о событиях")
	eventId := buildCommand.String("event-id", "", "ID главного события")
	beforeEvent := buildCommand.String("before", "", "временное окно до главного события")
	afterEvent := buildCommand.String("after", "", "временное окно после главного события")
	outFile := buildCommand.String("out", "", "выходной Markdown-файл (отчёт)")
	jsonFile := buildCommand.String("json", "", "выходной JSON-файл (отчёт)")
	requestFile := buildCommand.String("request", "", "JSON-файл, содержащий параметры запроса")
	factorsFile := buildCommand.String("factors", "", "YAML-файл, содержащий правила подозрительных факторов")
	
	// Флаги команды generate (пока без переменных)
	count := generateCommand.Int("count", 100000, "количество событий")
	scenario := generateCommand.String("scenario", "external_send", "сценарий")
	outGenFile := generateCommand.String("out", "", "выходной файл")
	seed := generateCommand.Int64("seed", 42, "seed генератора")


	args := os.Args
	if (len(args) < 2) {
		fmt.Println("Использование: incident-card build или incident-card generate")
		return
	}

	var req internal.Request
	var rules []internal.Rule

	if (os.Args[1] == "build") {
		buildCommand.Parse(os.Args[2:])

		// Проверяем передали файл с событиями, без него не можем продолжать работу
		if (*eventsFile == "") {
			log.Fatalf("Необходимо передать --events")
		}
		if (*eventId == "") {
			log.Fatalf("Необходимо передать --events-id или передать его в JSON-файле")
		}

		if (*requestFile != "") {
			reqData, err := os.ReadFile(*requestFile)
			if (err != nil) {
				log.Fatalf("Ошибка при чтении файла запроса %s: %v", *requestFile, err)
			}
			err = json.Unmarshal(reqData, &req)
			if (err != nil) {
				log.Fatalf("Ошибка парсинга JSON-запроса: %v", err)
			}
		}
		
		// Задаём значения из JSON, но только в том случае, если данные ещё не были заполнены CLI (приоритет) 
		if (req.MainEventID != "" && *eventId == "") {
			*eventId = req.MainEventID
		}
		if (req.WindowBefore != "" && *beforeEvent == "") {
			*beforeEvent = req.WindowBefore
		}
		if (req.WindowAfter != "" && *afterEvent == "") {
			*afterEvent = req.WindowAfter
		}

		// Если не заданы IncludeSameUser, IncludeSameFile, IncludeSameDestination и MaxEventsPerSection, ставим значение по умолчанию
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

		events, eventsLink, err := internal.ReadEvents(*eventsFile)
		if err != nil {
			log.Fatalf("Ошибка: %v\n", err)
		}
		fmt.Printf("Прочитано %d событий\n", len(events)) 

		index := internal.BuildIndex(events)


		mainEvent, isExist := index.GetEvent(*eventId)
		if (!isExist) {
			fmt.Printf("Событие с данным event_id (event_id = %s) не существует\n", *eventId)
			return 
		}

		// Создаём запрос в соответствии с структурой Request
		req = internal.Request{
			IncidentID: req.IncidentID,
			MainEventID: *eventId,
			WindowBefore: *beforeEvent,      
			WindowAfter:  *afterEvent,
			IncludeSameUser: req.IncludeSameUser,
			IncludeSameFile: req.IncludeSameFile, 
			IncludeSameDestination: req.IncludeSameDestination,
			MaxEventsPerSection:  limit,  
		}

		if (*factorsFile != "") {
			factData, err := os.ReadFile(*factorsFile)
			if err != nil {
				log.Fatalf("Ошибка при чтении файла, содержащего правила %s: %v", *factorsFile, err)
			}

			var factArr struct {
				Factors []internal.Rule `yaml:"factors"`
			}

			err = yaml.Unmarshal(factData, &factArr)
			if err != nil {
				log.Fatalf("Ошибка парсинга YAML: %v", err)
			}
			rules = factArr.Factors
		}

		answer := internal.BuildAnswer(mainEvent, index, events, eventsLink, req, rules)

		if (*outFile != "") {
			markdownCard := internal.GenerateMarkdownCard(mainEvent, &answer, index, limit)

			outFileOpen, err := os.OpenFile(*outFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
			if (err != nil) {
				log.Fatalf("Не удалось открыть файл %s: %v", *outFile, err)
			}
			defer outFileOpen.Close()

			_, err = outFileOpen.WriteString(markdownCard)
			if (err != nil) {
				log.Fatalf("Ошибка записи в файл Markdown-карточки: %v", err)
			}
			fmt.Printf("Markdown записан в файл %s\n", *outFile)
		} else {
			fmt.Println("Для сохранения отчёта в виде Markdown-карточки необходимо передать --out")
		}

		if (*jsonFile != "") {
			jsonFileOpen, err := os.OpenFile(*jsonFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
			if (err != nil) {
				log.Fatalf("Не удалось открыть файл %s: %v", *jsonFile, err)
			}
			defer jsonFileOpen.Close()

			jsonData, err := json.Marshal(answer)
			if (err != nil) {
				log.Fatalf("Ошибка сериализации JSON: %v", err)
			}
			err = os.WriteFile(*jsonFile, jsonData, 0666)
			if (err != nil) {
				log.Fatalf("Ошибка записи JSON: %v", err)
			}
			fmt.Printf("JSON сохранён в файл %s\n", *jsonFile)
		} else {
			fmt.Println("Для сохранения отчёта в JSON-файл необходимо передать --json")
		}
	} else if (os.Args[1] == "generate") {
		generateCommand.Parse(os.Args[2:])

		if (*outGenFile == "") {
			log.Fatalf("Необходимо указать файл --out, в который необходимо записать сгенерированные события")
		}

		events, err := internal.GenerateEvents(*count, *scenario, *seed)
		if (err != nil) {
			log.Fatalf("Ошибка генерации: %v", err)
		}

	
		jsonlFileOpen, err := os.OpenFile(*outGenFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		if (err != nil) {
			log.Fatalf("Не удалось открыть файл %s: %v", *outGenFile, err)
		}
		defer jsonlFileOpen.Close()
		
		for _, event := range events {
			data, err := json.Marshal(event)
			if (err != nil) {
				log.Fatalf("Ошибка сериализации JSONL: %v", err)
			}
			_, err = jsonlFileOpen.Write(data)
			if (err != nil) {
				log.Fatalf("Ошибка записи в файл: %v", err)
			}
			_, err = jsonlFileOpen.Write([]byte("\n"))
			if (err != nil) {
				log.Fatalf("Ошибка записи перевода строки в файл: %v", err)
			}
		}
		fmt.Printf("JSON сохранён в файл %s\n", *outGenFile)
	} else {
		log.Fatalf("Команда %s не поддерживается программой. Попробуйте заменить её на build или generate", os.Args[1])
	}

}