package internal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// Чтение событий из JSONL-файла
func ReadEvents(filePath string, buildUserGroup, buildFileGroup, buildDestGroup bool) (Index, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return Index{}, fmt.Errorf("Не удалось открыть файл %s: %w", filePath, err)
	}
	defer file.Close()

	// Оценка количества строк для предварительной аллокации
	fileInfo, err := file.Stat()
	if err != nil {
		return Index{}, fmt.Errorf("Не удалось получить информацию о файле: %w", err)
	}

	estimatedLines := 1000
	if fileInfo.Size() > 0 {
		estimatedLines = int(fileInfo.Size() / 500)
		if estimatedLines < 100 {
			estimatedLines = 100
		}
	}

	// Создаём индекс с предварительной ёмкостью
	index := Index{
		EventsById: make([]*Event, 0, estimatedLines),
		TimeIndex:  make([]*Event, 0, estimatedLines),
	}

	if buildUserGroup {
		index.UserIdGroup = make(map[string][]*Event, estimatedLines)
	}
	if buildFileGroup {
		index.FileIdGroup = make(map[string][]*Event, estimatedLines)
	}
	if buildDestGroup {
		index.DestinationIdGroup = make(map[string][]*Event, estimatedLines)
	}

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	const maxLineLength = 10 * 1024 * 1024
	buffer := make([]byte, 0, maxLineLength+maxLineLength)
	scanner.Buffer(buffer, maxLineLength+maxLineLength)

	// Интернирование строк: пул для часто повторяющихся значений
	stringPool := make(map[string]string)
	intern := func(s string) string {
		if s == "" {
			return ""
		}
		if cached, ok := stringPool[s]; ok {
			return cached
		}
		stringPool[s] = s
		return s
	}

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Удаление BOM
		if lineNumber == 1 && strings.HasPrefix(line, "\xEF\xBB\xBF") {
			line = strings.TrimPrefix(line, "\xEF\xBB\xBF")
		}

		if len(line) > maxLineLength {
			return Index{}, fmt.Errorf("%s:%d: строка слишком длинная\n", filePath, lineNumber)
		}

		var newEvent Event
		if err := json.Unmarshal([]byte(line), &newEvent); err != nil {
			return Index{}, fmt.Errorf("%s:%d: Ошибка парсинга строки: %v\n", filePath, lineNumber, err)
		}

		// Проверка обязательных полей
		if newEvent.EventID == "" || newEvent.TimeStamp == "" || newEvent.UserID == "" ||
			newEvent.MachineID == "" || newEvent.Action == "" || newEvent.Channel == "" {
			return Index{}, fmt.Errorf("%s:%d: Пропущено обязательное поле\n", filePath, lineNumber)
		}

		// Проверка формата времени RFC3339
		if _, err := time.Parse(time.RFC3339, newEvent.TimeStamp); err != nil {
			return Index{}, fmt.Errorf("%s:%d: Неверный формат поля timestamp: %w\n", filePath, lineNumber, err)
		}

		// Проверка отрицательного размера
		if newEvent.SizeBytes < 0 {
			return Index{}, fmt.Errorf("%s:%d: Отрицательное значение поля size_bytes\n", filePath, lineNumber)
		}

		event := &Event{
			EventID:         newEvent.EventID,
			TimeStamp:       newEvent.TimeStamp,
			UserID:          intern(newEvent.UserID),
			MachineID:       intern(newEvent.MachineID),
			Department:      intern(newEvent.Department),
			Action:          intern(newEvent.Action),
			Channel:         intern(newEvent.Channel),
			FileID:          intern(newEvent.FileID),
			FileName:        intern(newEvent.FileName),
			FileExt:         intern(newEvent.FileExt),
			ContentClasses:  internSlice(newEvent.ContentClasses, intern),
			DestinationID:   intern(newEvent.DestinationID),
			DestinationType: intern(newEvent.DestinationType),
			Destination:     intern(newEvent.Destination),
			SizeBytes:       newEvent.SizeBytes,
			Severity:        intern(newEvent.Severity),
			LineNumber:      lineNumber,
		}

		index.EventsById = append(index.EventsById, event)
		if buildUserGroup {
			index.UserIdGroup[event.UserID] = append(index.UserIdGroup[event.UserID], event)
		}
		if buildFileGroup && event.FileID != "" {
			index.FileIdGroup[event.FileID] = append(index.FileIdGroup[event.FileID], event)
		}
		if buildDestGroup && event.DestinationID != "" {
			index.DestinationIdGroup[event.DestinationID] = append(index.DestinationIdGroup[event.DestinationID], event)
		}
		index.TimeIndex = append(index.TimeIndex, event)
	}

	if err := scanner.Err(); err != nil {
		return Index{}, fmt.Errorf("Ошибка при чтении файла %s: %w", filePath, err)
	}

	// Сортировка EventsById по event_id
	sort.Slice(index.EventsById, func(i, j int) bool {
		return index.EventsById[i].EventID < index.EventsById[j].EventID
	})

	for i := 1; i < len(index.EventsById); i++ {
		if (index.EventsById[i].EventID == index.EventsById[i-1].EventID) {
			return Index{}, fmt.Errorf("%s:%d: Дублирование значения event_id %s\n",
				filePath, index.EventsById[i].LineNumber, index.EventsById[i].EventID)
		}
	}

	// Сортировка TimeIndex по времени
	sort.Slice(index.TimeIndex, func(i, j int) bool {
		ti, _ := time.Parse(time.RFC3339, index.TimeIndex[i].TimeStamp)
		tj, _ := time.Parse(time.RFC3339, index.TimeIndex[j].TimeStamp)
		if ti.Equal(tj) {
			return index.TimeIndex[i].EventID < index.TimeIndex[j].EventID
		}
		return ti.Before(tj)
	})

	return index, nil
}

func internSlice(src []string, intern func(string) string) []string {
    if len(src) == 0 {
        return nil
    }
    dst := make([]string, len(src))
    for i, s := range src {
        dst[i] = intern(s)
    }
    return dst
}
