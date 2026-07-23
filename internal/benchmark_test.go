package internal

import (
	"fmt"
	"testing"
)

// Измеряет время построения индекса
func BenchmarkBuildIndex(b *testing.B) {
	const events_size = 1000000
	events := GenerateStructuredBenchmarkEvents(events_size)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BuildIndex(events)
	}
}

// Измеряет время фильтрации событий по временному окну
func BenchmarkGetEventsInTimeRange(b *testing.B) {
	const events_size = 1000000
	events := GenerateStructuredBenchmarkEvents(events_size)
	mainTime := "2026-06-16T10:15:00Z"
	beforeDur := "30m"
	afterDur := "10m"

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetEventsInTimeRange(events, mainTime, beforeDur, afterDur)
	}
}


// Измеряет полный цикл сборки карточки
func BenchmarkBuildAnswer(b *testing.B) {
	const events_size = 1000000
	events := GenerateStructuredBenchmarkEvents(events_size)
	index := BuildIndex(events)

	// берём первое событие ка гланое (существование гарантированно)
	mainEvent, isExist := index.GetEvent("evt_0000001")
	if (!isExist) {
		b.Fatalf("Главное событие evt_0000001 не найдено.")
	}

	links := []LinkInFile{}

	flagTrue := true
	req := Request{
		IncidentID: "inc_001",
		MainEventID: mainEvent.EventID,
		WindowBefore: "30m",
		WindowAfter: "10m",
		IncludeSameUser: &flagTrue,
		IncludeSameFile: &flagTrue,
		IncludeSameDestination: &flagTrue,
		MaxEventsPerSection: 50,
	}

	// Загружаем реальные правила
	rules := []Rule{
		{FactorID: "external", Title: "External", Condition: Condition{Field: "destination_type", Equals: strPtr("external")}},
		{FactorID: "client", Title: "Client", Condition: Condition{Field: "content_classes", Contains: strPtr("client_data")}},
		{FactorID: "personal", Title: "Personal", Condition: Condition{Field: "content_classes", Contains: strPtr("personal_data")}},
		{FactorID: "finance", Title: "Finance", Condition: Condition{Field: "content_classes", Contains: strPtr("finance")}},
		{FactorID: "large", Title: "Large", Condition: Condition{Field: "size_bytes", Gt: int64SizeBytes(1046801)}},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BuildAnswer(mainEvent, index, events, links, req, rules)
	} 
}

// измеряет проверку таймлайна 
func BenchmarkBuildTimeline(b *testing.B) {
	
	mainEvent := &Event{
		EventID: "evt_12345",
		TimeStamp: "2026-06-16T10:15:00Z",
		UserID: "user_017",
		Action: "send",
	}

	// генерируем события для каждого контекста 
	const events_size = 200000
	contextBefore := make([]*Event, events_size)
	contextAfter := make([]*Event, events_size)
	userEvents := make([]*Event, events_size)
	fileEvents := make([]*Event, events_size)
	destinationEvents := make([]*Event, events_size)

	for i := 0; i < events_size; i++ {
		contextBefore[i] = &Event{EventID: "conextB_" + string(rune(i)), TimeStamp: "2026-06-16T10:10:00Z", UserID: "user_017", Action: "open"}
		contextAfter[i] = &Event{EventID: "conextA_" + string(rune(i)), TimeStamp: "2026-06-16T10:20:00Z", UserID: "user_017", Action: "close"}
		userEvents[i] = &Event{EventID: "userEv_" + string(rune(i)), TimeStamp: "2026-06-16T10:12:00Z", UserID: "user_017", Action: "copy"}
		fileEvents[i] = &Event{EventID: "fileEv_" + string(rune(i)), TimeStamp: "2026-06-16T10:14:00Z", UserID: "user_018", Action: "open"}
		destinationEvents[i] = &Event{EventID: "destEv_" + string(rune(i)), TimeStamp: "2026-06-16T10:16:00Z", UserID: "user_019", Action: "send"}
	}
	
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		BuildTimeline(mainEvent, contextBefore, contextAfter, userEvents, fileEvents, destinationEvents)
	} 

}

func BenchmarkCheckRules(b *testing.B) {
	event := &Event{
		EventID: "evt_12345",
		DestinationType: strPtr("external"),
		ContentClasses: []string{"client_data", "personal_data"},
		SizeBytes: int64SizeBytes(204800),
		Severity: strPtr("high"),
		Action: "send",
		Channel: "email",
		UserID: "user_017",
		MachineID: "pc_003",
		Department: strPtr("sales"),
		FileExt: strPtr("xlsx"),
	}

	// Загружаем реальные правила

	var rules []Rule
	fields := []string{"destination_type", "content_classes", "severity", "file_ext", "action", "channel", "user_id", "machine_id", "department", "size_bytes"}
	operators := []string{"equals", "contains", "gt", "lt", "exists", "in"}

	for i := 0; i< 30; i++ {
		field := fields[i%len(fields)]
		condition := Condition{Field: field}
		operator := operators[i%len(operators)]

		// для size_bytes используем только числовые операторы
		if (field == "size_bytes" && (operator == "contains" || operator == "in")) {
			numOperators := []string{"gt", "lt", "gte", "lte", "equals", "exists"}
			operator = numOperators[i%len(numOperators)]
		}

		// для строковых/массивных полей не используем только числовые операторы gt/lt/gte/lte
		if (field != "size_bytes" && (operator == "gt" || operator == "lt" || operator == "gte" || operator == "lte")) {
			strOperators := []string{"equals", "exists", "contains", "in"}
			operator = strOperators[i%len(strOperators)]
		}

		switch operator {
		case "equals":
			value := fmt.Sprintf("value_%d", i)
			condition.Equals = &value
		case "contains":
			value := fmt.Sprintf("cont_%d", i)
			condition.Contains = &value
		case "exists":
			flag := true
			condition.Exists = &flag
		case "in":
			condition.In = []string{fmt.Sprintf("in_%d", i), fmt.Sprintf("in_%d", i+1)}
		case "gt":
			value := int64((i + 1) * 1000)
			condition.Gt = &value
		case "lt":
			value := int64((i + 1) * 1000)
			condition.Lt = &value
		}
		rules = append(rules, Rule{
			FactorID: fmt.Sprintf("factor_%d", i),
			Title: fmt.Sprintf("Title %d", i),
			Condition: condition,
		})
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		CheckRules(event, rules)
	}
}

// Построение Markdown отчёта
func BenchmarkGenerateMarkdownCard(b *testing.B) {
	const events_size = 1000000
	events := GenerateStructuredBenchmarkEvents(events_size)
	index := BuildIndex(events)
	// берём первое событие ка гланое (существование гарантированно)
	mainEvent, isExist := index.GetEvent("evt_0000001")
	if (!isExist) {
		b.Fatalf("Главное событие evt_0000001 не найдено.")
	}

	flagTrue := true
	req := Request{
		IncidentID: "inc_001",
		MainEventID: mainEvent.EventID,
		WindowBefore: "30m",
		WindowAfter: "10m",
		IncludeSameUser: &flagTrue,
		IncludeSameFile: &flagTrue,
		IncludeSameDestination: &flagTrue,
		MaxEventsPerSection: 50,
	}

	answer, err := BuildAnswer(mainEvent, index, events, []LinkInFile{}, req, []Rule{})
	if (err != nil) {
		b.Fatalf("Ошибка: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i:=0; i<b.N; i++ {
		GenerateMarkdownCard(mainEvent, &answer, index, 50)
	}
}

// Построение JSON отчёта
func BenchmarkGenerateDOTGraph(b *testing.B) {
	const events_size = 1000000
	events := GenerateStructuredBenchmarkEvents(events_size)
	index := BuildIndex(events)
	// берём первое событие ка гланое (существование гарантированно)
	mainEvent, isExist := index.GetEvent("evt_0000001")
	if (!isExist) {
		b.Fatalf("Главное событие evt_0000001 не найдено.")
	}

	flagTrue := true
	req := Request{
		IncidentID: "inc_001",
		MainEventID: mainEvent.EventID,
		WindowBefore: "30m",
		WindowAfter: "10m",
		IncludeSameUser: &flagTrue,
		IncludeSameFile: &flagTrue,
		IncludeSameDestination: &flagTrue,
		MaxEventsPerSection: 50,
	}

	answer, err := BuildAnswer(mainEvent, index, events, []LinkInFile{}, req, []Rule{})
	if (err != nil) {
		b.Fatalf("Ошибка: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i:=0; i<b.N; i++ {
		GenerateDOTGraph(&answer, index)
	}
}
