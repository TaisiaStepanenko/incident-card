package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Создание тестового файла для проведения тестов
func createTestFile(t *testing.T, content string) string {
	t.Helper()
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "events.jsonl")
	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)
	return testFile
}

// Ошибка открытия файла
func TestReadEventsFileOpeningError(t *testing.T) {

	_, _, err := ReadEvents("noFile.json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Не удалось открыть файл")
}

// Успешное чтение событий из файла
func TestReadEventsSuccess(t *testing.T) {
	content := `{"event_id":"evt_12345","timestamp":"2026-06-16T10:00:00Z","user_id":"user_017","machine_id":"pc_003","action":"open","channel":"local"}
{"event_id":"evt_12346","timestamp":"2026-06-16T10:05:00Z","user_id":"user_018","machine_id":"pc_004","action":"email_send","channel":"email"}`
	testFile := createTestFile(t, content)
	defer os.Remove(testFile)

	events, eventLinks, err := ReadEvents(testFile)
	require.NoError(t, err)
	// Проверяем корректность считанных событий
	assert.Len(t, events, 2)
	assert.Equal(t, "evt_12345", events[0].EventID)
	assert.Equal(t, "evt_12346", events[1].EventID)

	// Проверяем корректность записанных ссылок
	assert.Equal(t, "evt_12345", eventLinks[0].EventID)
	assert.Equal(t, testFile, eventLinks[0].FileName)
	assert.Equal(t, 1, eventLinks[0].FileLine)

	assert.Equal(t, "evt_12346", eventLinks[1].EventID)
	assert.Equal(t, testFile, eventLinks[1].FileName)
	assert.Equal(t, 2, eventLinks[1].FileLine)
}

// Проверка на корректную обработку пустых строк в файле
func TestReadEventsEmptyLines(t *testing.T) {
	content := `{"event_id":"evt_12345","timestamp":"2026-06-16T10:00:00Z","user_id":"user_017","machine_id":"pc_003","action":"open","channel":"local"}

	{"event_id":"evt_12346","timestamp":"2026-06-16T10:05:00Z","user_id":"user_018","machine_id":"pc_004","action":"email_send","channel":"email"}`

	testFile := createTestFile(t, content)
	defer os.Remove(testFile)

	events, eventLinks, err := ReadEvents(testFile)
	require.NoError(t, err)
	assert.Len(t, events, 2)
	assert.Equal(t, "evt_12345", events[0].EventID)
	assert.Equal(t, "evt_12346", events[1].EventID)

	assert.Len(t, eventLinks, 2)
	assert.Equal(t, "evt_12345", eventLinks[0].EventID)
	assert.Equal(t, testFile, eventLinks[0].FileName)
	assert.Equal(t, 1, eventLinks[0].FileLine)

	assert.Equal(t, "evt_12346", eventLinks[1].EventID)
	assert.Equal(t, testFile, eventLinks[1].FileName)
	assert.Equal(t, 3, eventLinks[1].FileLine)
}

// Проверка на корректную обработку слишком длинных строк в файле
func TestReadEventsLongLines(t *testing.T) {
	longLine := make([]byte, 10*1024*1024+1)
	for i := range longLine {
		longLine[i] = 'i'
	}

	content := `{"event_id":"evt_12345","timestamp":"2026-06-16T10:00:00Z","user_id":"user_017","machine_id":"pc_003","action":"open","channel":"local"}
{"event_id":"evt_12346","timestamp":"2026-06-16T10:05:00Z","user_id":"user_018","machine_id":"pc_004","action":"email_send","channel":"email"}` + "\n" + string(longLine)

	testFile := createTestFile(t, content)
	defer os.Remove(testFile)

	events, eventLinks, err := ReadEvents(testFile)
	require.NoError(t, err)
	assert.Len(t, events, 2)
	assert.Equal(t, "evt_12345", events[0].EventID)
	assert.Equal(t, "evt_12346", events[1].EventID)

	assert.Len(t, eventLinks, 2)
	assert.Equal(t, "evt_12345", eventLinks[0].EventID)
	assert.Equal(t, testFile, eventLinks[0].FileName)
	assert.Equal(t, 1, eventLinks[0].FileLine)

	assert.Equal(t, "evt_12346", eventLinks[1].EventID)
	assert.Equal(t, testFile, eventLinks[1].FileName)
	assert.Equal(t, 2, eventLinks[1].FileLine)
}

// Проверка на корректную обработку неккоректных объектов JSON в файле
func TestReadEventsInvalidJSON(t *testing.T) {
	longLine := make([]byte, 10*1024*1024+1)
	for i := range longLine {
		longLine[i] = 'i'
	}

	content := `{json}
{"event_id":"evt_12345","timestamp":"2026-06-16T10:00:00Z","user_id":"user_017","machine_id":"pc_003","action":"open","channel":"local"}
{"event_id":"evt_12346","timestamp":"2026-06-16T10:05:00Z","user_id":"user_018","machine_id":"pc_004","action":"email_send","channel":"email"}` + "\n" + string(longLine)

	testFile := createTestFile(t, content)
	defer os.Remove(testFile)

	events, eventLinks, err := ReadEvents(testFile)
	require.NoError(t, err)
	assert.Len(t, events, 2)
	assert.Equal(t, "evt_12345", events[0].EventID)
	assert.Equal(t, "evt_12346", events[1].EventID)

	assert.Len(t, eventLinks, 2)
	assert.Equal(t, "evt_12345", eventLinks[0].EventID)
	assert.Equal(t, testFile, eventLinks[0].FileName)
	assert.Equal(t, 2, eventLinks[0].FileLine)

	assert.Equal(t, "evt_12346", eventLinks[1].EventID)
	assert.Equal(t, testFile, eventLinks[1].FileName)
	assert.Equal(t, 3, eventLinks[1].FileLine)
}
