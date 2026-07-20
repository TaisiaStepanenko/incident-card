package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckCondition(t *testing.T) {
	event := &Event{EventID: "evt_12345", TimeStamp: "2026-06-16T10:15:00Z", UserID: "user_017", MachineID: "m_001", Action: "send", Channel: "local",
		Department: strPtr("sales"), FileID: strPtr("file_001"), FileName: strPtr("file.txt"), FileExt: strPtr("txt"), DestinationID: strPtr("dst_001"),
		DestinationType: strPtr("none"), Destination: strPtr("test@gmil.com"), Severity: strPtr("low"), ContentClasses: []string{"personal_data", "finance"}, SizeBytes: int64SizeBytes(204800)}
	emptyEvent := &Event{}
	true_flag := true
	false_flag := false

	// -----EventID------
	// Equels
	assert.True(t, CheckCondition(event, Condition{Field: "event_id", Equals: strPtr("evt_12345")}))
	assert.False(t, CheckCondition(event, Condition{Field: "event_id", Equals: strPtr("evt12345")}))

	// In
	assert.True(t, CheckCondition(event, Condition{Field: "event_id", In: []string{"evt_34567", "evt_12345"}}))
	assert.False(t, CheckCondition(event, Condition{Field: "event_id", In: []string{"evt_34567", "event_12345"}}))

	// Exists
	assert.True(t, CheckCondition(event, Condition{Field: "event_id", Exists: &true_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "event_id", Exists: &false_flag}))
	assert.False(t, CheckCondition(event, Condition{Field: "event_id", Exists: &false_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "event_id", Exists: &true_flag}))

	// -----TimeStamp------
	// Equels
	assert.True(t, CheckCondition(event, Condition{Field: "timestamp", Equals: strPtr("2026-06-16T10:15:00Z")}))
	assert.False(t, CheckCondition(event, Condition{Field: "timestamp", Equals: strPtr("2026-06-16T10:18:00Z")}))

	// In
	assert.True(t, CheckCondition(event, Condition{Field: "timestamp", In: []string{"2026-06-16T10:15:00Z", "2026-06-16T10:18:00Z"}}))
	assert.False(t, CheckCondition(event, Condition{Field: "timestamp", In: []string{"2026-06-16T10:10:00Z", "2026-06-16T10:16:00Z", "2026-06-16T10:15:20Z"}}))

	// Exists
	assert.True(t, CheckCondition(event, Condition{Field: "timestamp", Exists: &true_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "timestamp", Exists: &false_flag}))
	assert.False(t, CheckCondition(event, Condition{Field: "timestamp", Exists: &false_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "timestamp", Exists: &true_flag}))

	// -----UserID------
	// Equels
	assert.True(t, CheckCondition(event, Condition{Field: "user_id", Equals: strPtr("user_017")}))
	assert.False(t, CheckCondition(event, Condition{Field: "user_id", Equals: strPtr("user_027")}))

	// In
	assert.True(t, CheckCondition(event, Condition{Field: "user_id", In: []string{"user_018", "user_017"}}))
	assert.False(t, CheckCondition(event, Condition{Field: "user_id", In: []string{"user_025", "user_019", "user_027"}}))

	// Exists
	assert.True(t, CheckCondition(event, Condition{Field: "user_id", Exists: &true_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "user_id", Exists: &false_flag}))
	assert.False(t, CheckCondition(event, Condition{Field: "user_id", Exists: &false_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "user_id", Exists: &true_flag}))

	// -----MachineID------
	// Equels
	assert.True(t, CheckCondition(event, Condition{Field: "machine_id", Equals: strPtr("m_001")}))
	assert.False(t, CheckCondition(event, Condition{Field: "machine_id", Equals: strPtr("m001")}))

	// In
	assert.True(t, CheckCondition(event, Condition{Field: "machine_id", In: []string{"m_001", "m_101"}}))
	assert.False(t, CheckCondition(event, Condition{Field: "machine_id", In: []string{"m_005", "k_001", "mac_091"}}))

	// Exists
	assert.True(t, CheckCondition(event, Condition{Field: "machine_id", Exists: &true_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "machine_id", Exists: &false_flag}))
	assert.False(t, CheckCondition(event, Condition{Field: "machine_id", Exists: &false_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "machine_id", Exists: &true_flag}))

	// -----Action------
	// Equels
	assert.True(t, CheckCondition(event, Condition{Field: "machine_id", Equals: strPtr("m_001")}))
	assert.False(t, CheckCondition(event, Condition{Field: "machine_id", Equals: strPtr("m001")}))

	// In
	assert.True(t, CheckCondition(event, Condition{Field: "machine_id", In: []string{"m_001", "m_101"}}))
	assert.False(t, CheckCondition(event, Condition{Field: "machine_id", In: []string{"m_005", "k_001", "mac_091"}}))

	// Exists
	assert.True(t, CheckCondition(event, Condition{Field: "machine_id", Exists: &true_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "machine_id", Exists: &false_flag}))
	assert.False(t, CheckCondition(event, Condition{Field: "machine_id", Exists: &false_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "machine_id", Exists: &true_flag}))

	// -----Action------
	// Equels
	assert.True(t, CheckCondition(event, Condition{Field: "action", Equals: strPtr("send")}))
	assert.False(t, CheckCondition(event, Condition{Field: "action", Equals: strPtr("delete")}))

	// In
	assert.True(t, CheckCondition(event, Condition{Field: "action", In: []string{"send", "open"}}))
	assert.False(t, CheckCondition(event, Condition{Field: "action", In: []string{"delete", "delete_all"}}))

	// Exists
	assert.True(t, CheckCondition(event, Condition{Field: "action", Exists: &true_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "action", Exists: &false_flag}))
	assert.False(t, CheckCondition(event, Condition{Field: "action", Exists: &false_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "action", Exists: &true_flag}))

	// -----Channel------
	// Equels
	assert.True(t, CheckCondition(event, Condition{Field: "channel", Equals: strPtr("local")}))
	assert.False(t, CheckCondition(event, Condition{Field: "channel", Equals: strPtr("email")}))

	// In
	assert.True(t, CheckCondition(event, Condition{Field: "channel", In: []string{"local", "usb"}}))
	assert.False(t, CheckCondition(event, Condition{Field: "channel", In: []string{"email", "printer"}}))

	// Exists
	assert.True(t, CheckCondition(event, Condition{Field: "channel", Exists: &true_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "channel", Exists: &false_flag}))
	assert.False(t, CheckCondition(event, Condition{Field: "channel", Exists: &false_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "channel", Exists: &true_flag}))

	// -----Department------
	// Equels
	assert.True(t, CheckCondition(event, Condition{Field: "department", Equals: strPtr("sales")}))
	assert.False(t, CheckCondition(event, Condition{Field: "department", Equals: strPtr("dev")}))

	// In
	assert.True(t, CheckCondition(event, Condition{Field: "department", In: []string{"sales", "hr"}}))
	assert.False(t, CheckCondition(event, Condition{Field: "department", In: []string{"dev", "manager"}}))

	// Exists
	assert.True(t, CheckCondition(event, Condition{Field: "department", Exists: &true_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "department", Exists: &false_flag}))
	assert.False(t, CheckCondition(event, Condition{Field: "department", Exists: &false_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "department", Exists: &true_flag}))

	// -----FileID------
	// Equels
	assert.True(t, CheckCondition(event, Condition{Field: "file_id", Equals: strPtr("file_001")}))
	assert.False(t, CheckCondition(event, Condition{Field: "file_id", Equals: strPtr("file_002")}))

	// In
	assert.True(t, CheckCondition(event, Condition{Field: "file_id", In: []string{"file_001", "file_004"}}))
	assert.False(t, CheckCondition(event, Condition{Field: "file_id", In: []string{"file_002", "file_005"}}))

	// Exists
	assert.True(t, CheckCondition(event, Condition{Field: "file_id", Exists: &true_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "file_id", Exists: &false_flag}))
	assert.False(t, CheckCondition(event, Condition{Field: "file_id", Exists: &false_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "file_id", Exists: &true_flag}))

	// -----FileName------
	// Equels
	assert.True(t, CheckCondition(event, Condition{Field: "file_name", Equals: strPtr("file.txt")}))
	assert.False(t, CheckCondition(event, Condition{Field: "file_name", Equals: strPtr("falseFile.txt")}))

	// In
	assert.True(t, CheckCondition(event, Condition{Field: "file_name", In: []string{"file.txt", "file.png"}}))
	assert.False(t, CheckCondition(event, Condition{Field: "file_name", In: []string{"file.json", "file.jpeg"}}))

	// Exists
	assert.True(t, CheckCondition(event, Condition{Field: "file_name", Exists: &true_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "file_name", Exists: &false_flag}))
	assert.False(t, CheckCondition(event, Condition{Field: "file_name", Exists: &false_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "file_name", Exists: &true_flag}))

	// Contains
	assert.True(t, CheckCondition(event, Condition{Field: "file_name", Contains: strPtr("file")}))
	assert.False(t, CheckCondition(event, Condition{Field: "file_name", Contains: strPtr("pdf")}))

	// -----FileExt------
	// Equels
	assert.True(t, CheckCondition(event, Condition{Field: "file_ext", Equals: strPtr("txt")}))
	assert.False(t, CheckCondition(event, Condition{Field: "file_ext", Equals: strPtr("jpeg")}))

	// In
	assert.True(t, CheckCondition(event, Condition{Field: "file_ext", In: []string{"txt", "pdf"}}))
	assert.False(t, CheckCondition(event, Condition{Field: "file_ext", In: []string{"json", "jpeg"}}))

	// Exists
	assert.True(t, CheckCondition(event, Condition{Field: "file_ext", Exists: &true_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "file_ext", Exists: &false_flag}))
	assert.False(t, CheckCondition(event, Condition{Field: "file_ext", Exists: &false_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "file_ext", Exists: &true_flag}))

	// -----DestinationID------
	// Equels
	assert.True(t, CheckCondition(event, Condition{Field: "destination_id", Equals: strPtr("dst_001")}))
	assert.False(t, CheckCondition(event, Condition{Field: "destination_id", Equals: strPtr("001")}))

	// In
	assert.True(t, CheckCondition(event, Condition{Field: "destination_id", In: []string{"dst_012", "dst_001"}}))
	assert.False(t, CheckCondition(event, Condition{Field: "destination_id", In: []string{"dst_0", "dst_013"}}))

	// Exists
	assert.True(t, CheckCondition(event, Condition{Field: "destination_id", Exists: &true_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "destination_id", Exists: &false_flag}))
	assert.False(t, CheckCondition(event, Condition{Field: "destination_id", Exists: &false_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "destination_id", Exists: &true_flag}))

	// -----DestinationType------
	// Equels
	assert.True(t, CheckCondition(event, Condition{Field: "destination_type", Equals: strPtr("none")}))
	assert.False(t, CheckCondition(event, Condition{Field: "destination_type", Equals: strPtr("internal")}))

	// In
	assert.True(t, CheckCondition(event, Condition{Field: "destination_type", In: []string{"internal", "none"}}))
	assert.False(t, CheckCondition(event, Condition{Field: "destination_type", In: []string{"internal", "usb"}}))

	// Exists
	assert.True(t, CheckCondition(event, Condition{Field: "destination_type", Exists: &true_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "destination_type", Exists: &false_flag}))
	assert.False(t, CheckCondition(event, Condition{Field: "destination_type", Exists: &false_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "destination_type", Exists: &true_flag}))

	// -----Destination------
	// Equels
	assert.True(t, CheckCondition(event, Condition{Field: "destination", Equals: strPtr("test@gmil.com")}))
	assert.False(t, CheckCondition(event, Condition{Field: "destination", Equals: strPtr("test2@gmil.com")}))

	// In
	assert.True(t, CheckCondition(event, Condition{Field: "destination", In: []string{"test@gmil.com", "test2@gmil.com"}}))
	assert.False(t, CheckCondition(event, Condition{Field: "destination", In: []string{"falsetest@gmil.com", "falsetest2@gmil.com"}}))

	// Exists
	assert.True(t, CheckCondition(event, Condition{Field: "destination", Exists: &true_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "destination", Exists: &false_flag}))
	assert.False(t, CheckCondition(event, Condition{Field: "destination", Exists: &false_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "destination", Exists: &true_flag}))

	// Contains
	assert.True(t, CheckCondition(event, Condition{Field: "destination", Contains: strPtr("test")}))
	assert.False(t, CheckCondition(event, Condition{Field: "destination", Contains: strPtr("yandex")}))

	// -----Severity------
	// Equels
	assert.True(t, CheckCondition(event, Condition{Field: "severity", Equals: strPtr("low")}))
	assert.False(t, CheckCondition(event, Condition{Field: "severity", Equals: strPtr("medium")}))

	// In
	assert.True(t, CheckCondition(event, Condition{Field: "severity", In: []string{"medium", "low"}}))
	assert.False(t, CheckCondition(event, Condition{Field: "severity", In: []string{"medium", "critical"}}))

	// Exists
	assert.True(t, CheckCondition(event, Condition{Field: "severity", Exists: &true_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "severity", Exists: &false_flag}))
	assert.False(t, CheckCondition(event, Condition{Field: "severity", Exists: &false_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "severity", Exists: &true_flag}))

	// -----ContentClasses------
	// Equels
	assert.True(t, CheckCondition(event, Condition{Field: "content_classes", Equals: strPtr("personal_data")}))
	assert.False(t, CheckCondition(event, Condition{Field: "content_classes", Equals: strPtr("client_data")}))

	// In
	assert.True(t, CheckCondition(event, Condition{Field: "content_classes", In: []string{"personal_data", "finance"}}))
	assert.False(t, CheckCondition(event, Condition{Field: "content_classes", In: []string{"client_data", "none"}}))

	// Exists
	assert.True(t, CheckCondition(event, Condition{Field: "content_classes", Exists: &true_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "content_classes", Exists: &false_flag}))
	assert.False(t, CheckCondition(event, Condition{Field: "content_classes", Exists: &false_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "content_classes", Exists: &true_flag}))

	// Contains
	assert.True(t, CheckCondition(event, Condition{Field: "content_classes", Contains: strPtr("personal_data")}))
	assert.False(t, CheckCondition(event, Condition{Field: "content_classes", Contains: strPtr("client_data")}))

	// -----SizeBytesq------
	// Equels
	assert.True(t, CheckCondition(event, Condition{Field: "size_bytes", Equals: strPtr("204800")}))
	assert.False(t, CheckCondition(event, Condition{Field: "size_bytes", Equals: strPtr("100")}))

	// In
	assert.True(t, CheckCondition(event, Condition{Field: "size_bytes", In: []string{"100", "204800"}}))
	assert.False(t, CheckCondition(event, Condition{Field: "size_bytes", In: []string{"505", "102400"}}))

	// Exists
	assert.True(t, CheckCondition(event, Condition{Field: "size_bytes", Exists: &true_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "size_bytes", Exists: &false_flag}))
	assert.False(t, CheckCondition(event, Condition{Field: "size_bytes", Exists: &false_flag}))
	assert.False(t, CheckCondition(emptyEvent, Condition{Field: "size_bytes", Exists: &true_flag}))

	// Contains
	assert.False(t, CheckCondition(event, Condition{Field: "size_bytes", Contains: strPtr("2048")}))

	emptyEvent.SizeBytes = int64SizeBytes(0)
	assert.True(t, CheckCondition(event, Condition{Field: "size_bytes", Gt: int64SizeBytes(100000)}))
	assert.True(t, CheckCondition(event, Condition{Field: "size_bytes", Gte: int64SizeBytes(100000)}))
	assert.False(t, CheckCondition(event, Condition{Field: "size_bytes", Gt: int64SizeBytes(300000)}))
	assert.False(t, CheckCondition(event, Condition{Field: "size_bytes", Gte: int64SizeBytes(300000)}))
	assert.True(t, CheckCondition(emptyEvent, Condition{Field: "size_bytes", Lt: int64SizeBytes(100000)}))
	assert.True(t, CheckCondition(emptyEvent, Condition{Field: "size_bytes", Lte: int64SizeBytes(100000)}))
	assert.False(t, CheckCondition(event, Condition{Field: "size_bytes", Lt: int64SizeBytes(500)}))
	assert.False(t, CheckCondition(event, Condition{Field: "size_bytes", Lte: int64SizeBytes(500)}))

}

func TestCheckRules(t *testing.T) {
	event := &Event{
		EventID:         "evt_12345",
		FileName:        strPtr("importantFile.txt"),
		DestinationType: strPtr("external"),
		Severity:        strPtr("high"),
		ContentClasses:  []string{"personal_data", "finance"},
		SizeBytes:       int64SizeBytes(204800),
	}

	rules := []Rule{
		{Title: "Внешний адресат", Condition: Condition{Field: "destination_type", Equals: strPtr("external")}},
		{Title: "Клиентские данные", Condition: Condition{Field: "content_classes", Contains: strPtr("client_data")}},
		{Title: "События высокой важности", Condition: Condition{Field: "severity", Equals: strPtr("high")}},
		{Title: "Файл txt", Condition: Condition{Field: "file_name", Contains: strPtr(".txt")}},
		{Title: "Слишком большой файл", Condition: Condition{Field: "size_bytes", Gte: int64SizeBytes(12827381273)}},
	}

	expectedList := []string{"Внешний адресат", "События высокой важности", "Файл txt"}

	result := CheckRules(event, rules)
	assert.Equal(t, expectedList, result)
}
