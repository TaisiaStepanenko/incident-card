package internal

import (
	"math/rand"
	"testing"
	"time"
)

func GenerateBenchmarkEvents(count int) []Event {
	rng := rand.New(rand.NewSource(42))
	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	events := make([]Event, count)
	for i := 0; i < count; i++ {
		events[i] = GenerateRandomEvent(rng, baseTime)
	}
	return events
}

// Измеряет время построения индекса
func BenchmarkBuildIndex(b *testing.B) {
	const events_size = 1000000
	events := GenerateBenchmarkEvents(events_size)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BuildIndex(events)
	}
}

// Измеряет время фильтрации событий по временному окну
func BenchmarkGetEventsInTimeRange(b *testing.B) {
	const events_size = 1000000
	events := GenerateBenchmarkEvents(events_size)
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
	events := GenerateBenchmarkEvents(events_size)
	index := BuildIndex(events)

	// берём первое событие ка гланое (существование гарантированно)
	mainEvent := &events[0]

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