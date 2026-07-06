package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEventsInTimeRangeValid(t *testing.T) {
	mainTime := "2026-06-16T10:15:00Z"
	events := []Event{
		{EventID: "evt_12345", TimeStamp: "2026-06-16T10:10:00Z"},
		{EventID: "evt_12346", TimeStamp: "2026-06-16T10:15:00Z"},
		{EventID: "evt_12347", TimeStamp: "2026-06-16T10:20:00Z"},
		{EventID: "evt_12348", TimeStamp: "2026-06-16T10:00:00Z"},
		{EventID: "evt_12349", TimeStamp: "2026-06-16T10:30:00Z"},
	}

	eventsInTimeRange, err := GetEventsInTimeRange(events, mainTime, "10m", "5m")
	require.NoError(t, err)
	for _, event := range eventsInTimeRange {
		assert.Contains(t, []string{"evt_12345", "evt_12346", "evt_12347"}, event.EventID)
	}

	assert.Equal(t, "evt_12345", eventsInTimeRange[0].EventID)
	assert.Equal(t, "evt_12346", eventsInTimeRange[1].EventID)
	assert.Equal(t, "evt_12347", eventsInTimeRange[2].EventID)
}

func TestGetEventsInTimeRangeNoEvents(t *testing.T) {
	mainTime := "2026-06-16T10:15:00Z"
	events := []Event{
		{EventID: "evt_12345", TimeStamp: "2026-05-16T10:00:00Z"},
		{EventID: "evt_12346", TimeStamp: "2026-06-16T11:55:00Z"},
	}

	eventsInTimeRange, err := GetEventsInTimeRange(events, mainTime, "20h", "1h")
	require.NoError(t, err)

	assert.Empty(t, eventsInTimeRange)
}

func TestGetEventsInTimeRangeBorders(t *testing.T) {
	mainTime := "2026-06-16T10:15:00Z"
	events := []Event{
		{EventID: "evt_12345", TimeStamp: "2026-06-16T10:10:00Z"},
		{EventID: "evt_12346", TimeStamp: "2026-06-16T10:20:00Z"},
	}

	eventsInTimeRange, err := GetEventsInTimeRange(events, mainTime, "5m", "5m")
	require.NoError(t, err)
	assert.Len(t, eventsInTimeRange, 2)
}

func TestGetEventsInTimeRangeInvalidMainTime(t *testing.T) {
	mainTime := "2026-06-16T10:15:00"
	events := []Event{
		{EventID: "evt_12345", TimeStamp: "2026-06-16T10:10:00Z"},
		{EventID: "evt_12346", TimeStamp: "2026-06-16T10:20:00Z"},
	}

	_, err := GetEventsInTimeRange(events, mainTime, "5m", "5m")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Ошибка при парсинге времени главного события:")
}

func TestGetEventsInTimeRangeInvalidDuration(t *testing.T) {
	mainTime := "2026-06-16T10:15:00Z"
	events := []Event{
		{EventID: "evt_12345", TimeStamp: "2026-06-16T10:10:00Z"},
		{EventID: "evt_12346", TimeStamp: "2026-06-16T10:20:00Z"},
	}

	_, err := GetEventsInTimeRange(events, mainTime, "1day", "5m")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Задан неверный формат --before:")

	_, err = GetEventsInTimeRange(events, mainTime, "5m", "1day")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Задан неверный формат --after:")
}

func TestGetEventsInTimeRangeInvalidTime(t *testing.T) {
	mainTime := "2026-06-16T10:15:00Z"
	events := []Event{
		{EventID: "evt_12345", TimeStamp: "2026-06-16T10:10:00"},
		{EventID: "evt_12346", TimeStamp: "2026-06-16T10:20:00Z"},
	}

	eventsInTimeRange, err := GetEventsInTimeRange(events, mainTime, "5m", "5m")
	require.NoError(t, err)
	assert.Len(t, eventsInTimeRange, 1)
}