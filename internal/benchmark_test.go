package internal

import (
	"testing"
)

// Измеряет время построения индекса
func BenchmarkBuildIndex(b *testing.B) {
	const events_size = 1000000
	events := GenerateStructuredBenchmarkEvents(events_size)

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

	rules := []Rule{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BuildAnswer(mainEvent, index, events, links, req, rules)
	} 
}
