package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateDOTGraph_Success(t *testing.T) {

	fileID1 := "file_001"
	fileName1 := "file.xlsx"
	destID1 := "dst_010"
	destName1 := "usb_001"

	fileID2 := "file_002"
	fileName2 := "file.pdf"

	mainEvent := Event{
		EventID:       "evt_12345",
		TimeStamp:     "2026-06-16T10:15:00Z",
		UserID:        "user_017",
		MachineID:     "pc_003",
		Action:        "copy_to_usb",
		Channel:       "usb",
		FileID:        &fileID1,
		FileName:      &fileName1,
		DestinationID: &destID1,
		Destination:   &destName1,
	}

	// Событие до (открытие файла)
	beforeMainEvent := Event{
		EventID:   "evt_12338",
		TimeStamp: "2026-06-16T10:11:00Z",
		UserID:    "user_017",
		MachineID: "pc_003",
		Action:    "open_file",
		Channel:   "local",
		FileID:    &fileID1,
		FileName:  &fileName1,
	}

	// События после (удаление файла)
	afterMainEvent := Event{
		EventID:   "evt_12347",
		TimeStamp: "2026-06-16T10:27:00Z",
		UserID:    "user_017",
		MachineID: "pc_003",
		Action:    "delete_file",
		Channel:   "local",
		FileID:    &fileID1,
		FileName:  &fileName1,
	}

	// событие с другим файлом
	otherEvent := Event{
		EventID:   "evt_12346",
		TimeStamp: "2026-06-16T10:23:00Z",
		UserID:    "user_017",
		MachineID: "pc_003",
		Action:    "open_file",
		Channel:   "local",
		FileID:    &fileID2,
		FileName:  &fileName2,
	}

	events := []Event{mainEvent, beforeMainEvent, afterMainEvent, otherEvent}
	index := BuildIndex(events)

	answer := &Answer{
		MainEvent: MainEvent{
			EventID: mainEvent.EventID,
			Action:  mainEvent.Action,
		},
		TimeLine: []TimelineItem{
			{
				Timestamp:   beforeMainEvent.TimeStamp,
				EventID:     beforeMainEvent.EventID,
				Role:        RoleBeforeContext,
				UserID:      beforeMainEvent.UserID,
				Action:      beforeMainEvent.Action,
				FileName:    *beforeMainEvent.FileName,
				Destination: "",
				Severity:    "low",
			},
			{
				Timestamp:   mainEvent.TimeStamp,
				EventID:     mainEvent.EventID,
				Role:        RoleMain,
				UserID:      mainEvent.UserID,
				Action:      mainEvent.Action,
				FileName:    *mainEvent.FileName,
				Destination: *mainEvent.Destination,
				Severity:    "high",
			},
			{
				Timestamp:   afterMainEvent.TimeStamp,
				EventID:     afterMainEvent.EventID,
				Role:        RoleAfterContext,
				UserID:      afterMainEvent.UserID,
				Action:      afterMainEvent.Action,
				FileName:    *afterMainEvent.FileName,
				Destination: "",
				Severity:    "high",
			},
			{
				Timestamp:   otherEvent.TimeStamp,
				EventID:     otherEvent.EventID,
				Role:        RoleSameUser,
				UserID:      otherEvent.UserID,
				Action:      otherEvent.Action,
				FileName:    *otherEvent.FileName,
				Destination: "",
				Severity:    "low",
			},
		},
	}

	dot, err := GenerateDOTGraph(answer, index)

	require.NoError(t, err)

	// Проверяем содержание обязательных элементов
	assert.Contains(t, dot, "digraph IncidentGraph {\n")
	assert.Contains(t, dot, " rankdir=LR;\n")
	assert.Contains(t, dot, " node[shape=box];\n")

	assert.Contains(t, dot, `"evt_12345" [style=filled, fillcolor=lightblue, label="evt_12345\ncopy_to_usb\n2026-06-16T10:15:00Z"];`)
	assert.Contains(t, dot, `"evt_12338" [label="evt_12338\nopen_file\n2026-06-16T10:11:00Z"];`)
	assert.Contains(t, dot, `"evt_12347" [label="evt_12347\ndelete_file\n2026-06-16T10:27:00Z"];`)
	assert.Contains(t, dot, `"evt_12346" [label="evt_12346\nopen_file\n2026-06-16T10:23:00Z"];`)

	// Узел пользователя
	assert.Contains(t, dot, `"user_user_017" [shape=ellipse, label="user_017"];`)

	// Ребра от событий к пользователю "performed by"
	assert.Contains(t, dot, `"evt_12345" -> "user_user_017" [label="performed by"];`)
	assert.Contains(t, dot, `"evt_12338" -> "user_user_017" [label="performed by"];`)
	assert.Contains(t, dot, `"evt_12347" -> "user_user_017" [label="performed by"];`)
	assert.Contains(t, dot, `"evt_12346" -> "user_user_017" [label="performed by"];`)

	// Узел файла
	assert.Contains(t, dot, `"file_file_001" [shape=note, label="file_001\nfile.xlsx"];`)
	assert.Contains(t, dot, `"file_file_002" [shape=note, label="file_002\nfile.pdf"];`)

	// Ребра от событий к файлу "uses file"
	assert.Contains(t, dot, `"evt_12345" -> "file_file_001" [label="uses file"];`)
	assert.Contains(t, dot, `"evt_12338" -> "file_file_001" [label="uses file"];`)
	assert.Contains(t, dot, `"evt_12347" -> "file_file_001" [label="uses file"];`)
	assert.Contains(t, dot, `"evt_12346" -> "file_file_002" [label="uses file"];`)

	// Узел адресата
	assert.Contains(t, dot, `"dest_dst_010" [shape=cylinder, label="dst_010\nusb_001"];`)

	// Ребра от событий к адресату "sends to"
	assert.Contains(t, dot, `"evt_12345" -> "dest_dst_010" [label="sends to"];`)

}

func TestGenerateDOTGraph_EmptyTimeLine(t *testing.T) {
	answer := &Answer{
		MainEvent: MainEvent{
			EventID: "evt_12345",
			Action:  "open_file",
		},
		TimeLine: []TimelineItem{},
	}
	index := Index{}

	_, err := GenerateDOTGraph(answer, index)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Нет событий для построения графа")
}

func TestGenerateDOTGraph_MainEventNotFound(t *testing.T) {
	answer := &Answer{
		MainEvent: MainEvent{
			EventID: "evt_not_exist",
			Action:  "no",
		},
		TimeLine: []TimelineItem{
			{
				EventID:   "evt_not_exist",
				Timestamp: "2026-06-16T10:15:00Z",
				Role:      RoleMain,
			},
		},
	}

	index := Index{}

	_, err := GenerateDOTGraph(answer, index)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Главное событие не найдено в индексе.")
}

func TestGenerateDOTGraph_TimelineEventNotFound(t *testing.T) {
	mainEvent := Event{
		EventID:   "evt_12345",
		TimeStamp: "2026-06-16T10:15:00Z",
		UserID:    "user_017",
		MachineID: "pc_003",
		Action:    "copy_to_usb",
		Channel:   "usb",
	}
	events := []Event{mainEvent}
	index := BuildIndex(events)

	answer := &Answer{
		MainEvent: MainEvent{
			EventID: mainEvent.EventID,
			Action:  mainEvent.Action,
		},
		TimeLine: []TimelineItem{
			{
				EventID:     mainEvent.EventID,
				Timestamp:   mainEvent.TimeStamp,
				Role:        RoleMain,
				UserID:      mainEvent.UserID,
				Action:      mainEvent.Action,
				FileName:    "file.txt",
				Destination: "usb",
				Severity:    "high",
			},
			{
				EventID:   "evt_12346",
				Timestamp: "2026-06-16T10:10:00Z",
				Role:      RoleBeforeContext,
				UserID:    "user_017",
				Action:    "open_file",
				FileName:  "file.pdf",
				Severity:  "low",
			},
		},
	}

	dot, err := GenerateDOTGraph(answer, index)
	assert.NoError(t, err)

	assert.Contains(t, dot, `"evt_12345"`)
	assert.Contains(t, dot, `label="evt_12345\ncopy_to_usb\n2026-06-16T10:15:00Z"`)

	assert.Contains(t, dot, `"evt_12346" [label="evt_12346\nopen_file\n2026-06-16T10:10:00Z"];`)

	assert.Contains(t, dot, `"evt_12345" -> "user_user_017" [label="performed by"];`)

	// Проверяем, что нет рёбер к файлу или адресату для отсутствующего события
	assert.NotContains(t, dot, `"evt_12346" -> "file_`)
	assert.NotContains(t, dot, `"evt_12346" -> "dest_`)
}
