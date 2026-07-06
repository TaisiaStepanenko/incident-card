package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIndexEventId(t *testing.T) {

	events := []Event{
		{EventID: "evt_12345", UserID: "user_017", Action: "open"},
		{EventID: "evt_12346", UserID: "user_018", Action: "email_send"},
		{EventID: "evt_12345", UserID: "user_018", Action: "email_send"},
	}

	index := BuildIndex(events)
	event, isExist := index.GetEvent("evt_12345")
	assert.True(t, isExist)
	assert.Equal(t, "evt_12345", event.EventID)

	event, isExist = index.GetEvent("evt_12346")
	assert.True(t, isExist)
	assert.Equal(t, "evt_12346", event.EventID)

	event, isExist = index.GetEvent("evt_12345")
	assert.True(t, isExist)
	assert.NotEqual(t, "user_018", event.UserID)

	event, isExist = index.GetEvent("evt_12347")
	assert.False(t, isExist)
}

func TestIndexUserIdGroup(t *testing.T) {

	events := []Event{
		{EventID: "evt_12345", UserID: "user_017", Action: "open"},
		{EventID: "evt_12346", UserID: "user_018", Action: "email_send"},
		{EventID: "evt_12347", UserID: "user_018", Action: "open"},
	}

	index := BuildIndex(events)

	userEvents := index.GetEventByUser("user_018")
	assert.Len(t, userEvents, 2)
	for _, event := range userEvents {
		assert.Contains(t, []string{"evt_12346", "evt_12347"}, event.EventID)
	}

	userEvents = index.GetEventByUser("user_017")
	assert.Len(t, userEvents, 1)
	assert.Equal(t, userEvents[0].EventID, "evt_12345")

	userEvents = index.GetEventByUser("user_020")
	assert.Len(t, userEvents, 0)
	assert.Empty(t, userEvents)
}

func TestIndexFileIdGroup(t *testing.T) {
	file1 := "file_001"
	file2 := "file_002"

	events := []Event{
		{EventID: "evt_12345", FileID: &file1, Action: "open"},
		{EventID: "evt_12346", FileID: &file1, Action: "email_send"},
		{EventID: "evt_12348", FileID: nil, Action: "email_send"},
		{EventID: "evt_12347", FileID: &file2, Action: "open"},
	}

	index := BuildIndex(events)

	fileEvents := index.GetEventByFile(file1)
	assert.Len(t, fileEvents, 2)
	for _, event := range fileEvents {
		assert.Contains(t, []string{"evt_12345", "evt_12346"}, event.EventID)
	}

	fileEvents = index.GetEventByFile(file2)
	assert.Len(t, fileEvents, 1)
	assert.Equal(t, fileEvents[0].EventID, "evt_12347")

	nonExistentFile := index.GetEventByFile("nonExistentFile")
	assert.Len(t, nonExistentFile, 0)
	assert.Empty(t, nonExistentFile)
}

func TestIndexDestinationIdGroup(t *testing.T) {
	dest1 := "dest_001"
	dest2 := "dest_002"

	events := []Event{
		{EventID: "evt_12345", DestinationID: &dest1, Action: "open"},
		{EventID: "evt_12346", DestinationID: &dest1, Action: "email_send"},
		{EventID: "evt_12348", DestinationID: nil, Action: "email_send"},
		{EventID: "evt_12347", DestinationID: &dest2, Action: "open"},
	}

	index := BuildIndex(events)

	destinationEvents := index.GetEventByDestination(dest1)
	assert.Len(t, destinationEvents, 2)
	for _, event := range destinationEvents {
		assert.Contains(t, []string{"evt_12345", "evt_12346"}, event.EventID)
	}

	destinationEvents = index.GetEventByDestination(dest2)
	assert.Len(t, destinationEvents, 1)
	assert.Equal(t, destinationEvents[0].EventID, "evt_12347")

	nonExistentDestination := index.GetEventByDestination("nonExistentDestination")
	assert.Len(t, nonExistentDestination, 0)
	assert.Empty(t, nonExistentDestination)
}