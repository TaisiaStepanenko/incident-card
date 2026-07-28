package internal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// Чтение событий из JSONL-файла
func ReadEvents(filePath string, buildUserGroup, buildFileGroup, buildDestGroup bool) ([]*Event, Index, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, Index{}, fmt.Errorf("Не удалось открыть файл %s: %w", filePath, err)
	}
	defer file.Close()

	// Оценка количества строк для предварительной аллокации
	fileInfo, err := file.Stat() // получаем размер файла
	if (err != nil) {
		return nil, Index{}, fmt.Errorf("Не удалось получить информацию о файле: %w", err)
	}

	estimatedLines := 1000
	if (fileInfo.Size() > 0) {
		estimatedLines = int(fileInfo.Size() / 500) //  500 байт на строку
		if (estimatedLines < 100) {
			estimatedLines = 100
		}
	}

	// Создаём срез указателей на все события с предварительной ёмкостью
	allEvents := make([]*Event, 0, estimatedLines)

	// Создаём индекс с предварительной ёмкостью
	index := Index{
		EventIdIndex: make(map[string]*Event, estimatedLines),
	}

	if (buildUserGroup) {
		index.UserIdGroup = make(map[string][]*Event, estimatedLines)
	}
	if (buildFileGroup) {
		index.FileIdGroup = make(map[string][]*Event, estimatedLines)
	}
	if (buildDestGroup) {
		index.DestinationIdGroup = make(map[string][]*Event, estimatedLines)
	}

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	const maxLineLength = 10 * 1024 * 1024

	buffer := make([]byte, 0, maxLineLength+maxLineLength)
	scanner.Buffer(buffer, maxLineLength+maxLineLength)

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue // пропускаем пустые строки
		}

		// Удаление BOM
		if (lineNumber == 1 && strings.HasPrefix(line, "\xEF\xBB\xBF")) {
			line = strings.TrimPrefix(line, "\xEF\xBB\xBF")
		}

		if len(line) > maxLineLength {
			return nil, Index{}, fmt.Errorf("%s:%d: cтрока слишком длинная\n", filePath, lineNumber) // проверяем слишком длинные значения
		}

		var newEvent Event

		err := json.Unmarshal([]byte(line), &newEvent) // декодирование JSON

		if err != nil {
			return nil, Index{}, fmt.Errorf("%s:%d: Ошибка парсинга строки: %v\n", filePath, lineNumber, err)
		}

		// Проверка обязательных полей
		if (newEvent.EventID == "" || newEvent.TimeStamp == "" || newEvent.UserID == "" || newEvent.MachineID == "" || newEvent.Action == "" || newEvent.Channel == "") {
			return nil, Index{}, fmt.Errorf("%s:%d: Пропущено обязательное поле\n", filePath, lineNumber)
		}

		// Проверка формата времени RFC3339
		_, err = time.Parse(time.RFC3339, newEvent.TimeStamp)
		if (err != nil) {
			return nil, Index{}, fmt.Errorf("%s:%d: Неверный формат поля timestamp: %w\n", filePath, lineNumber, err)
		}

		// Проверка отрицательного размера
		if (newEvent.SizeBytes < 0) {
			return nil, Index{}, fmt.Errorf("%s:%d: Отрицательное значение поля size_bytes\n", filePath, lineNumber)
		}

		// Проверка дубликатов event_id
		_, isExist := index.EventIdIndex[newEvent.EventID]
		if (isExist) {
			return nil, Index{}, fmt.Errorf("%s:%d: Дублирование значения event_id %s\n", filePath, lineNumber, newEvent.EventID)
		}

		event := &Event{
			EventID: newEvent.EventID,
			TimeStamp: newEvent.TimeStamp,
			UserID: newEvent.UserID,
			MachineID: newEvent.MachineID,
			Department: newEvent.Department,
			Action: newEvent.Action,
			Channel: newEvent.Channel,
			FileID: newEvent.FileID,
			FileName: newEvent.FileName,
			FileExt: newEvent.FileExt,
			ContentClasses: newEvent.ContentClasses,
			DestinationID: newEvent.DestinationID,
			DestinationType: newEvent.DestinationType,
			Destination: newEvent.Destination,
			SizeBytes: newEvent.SizeBytes,
			Severity: newEvent.Severity,
			LineNumber: lineNumber,
		}

		// Заполняем индекс
		index.EventIdIndex[event.EventID] = event
		if (buildUserGroup) {
			index.UserIdGroup[event.UserID] = append(index.UserIdGroup[event.UserID], event)
		}
		if (buildFileGroup && event.FileID != "") {
			index.FileIdGroup[event.FileID] = append(index.FileIdGroup[event.FileID], event)
		}
		if (buildDestGroup && event.DestinationID != "") {
			index.DestinationIdGroup[event.DestinationID] = append(index.DestinationIdGroup[event.DestinationID], event)
		}
		allEvents = append(allEvents, event)
	}

	err = scanner.Err()
	if err != nil {
		return nil, Index{}, fmt.Errorf("Ошибка при чтении файла %s: %w", filePath, err) // ошибка при сканировнии
	}

	return allEvents, index, nil

}
