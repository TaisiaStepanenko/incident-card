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
func ReadEvents(filePath string) ([]Event, Index, error) {
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

	// Создаём срез событий с предварительной ёмкостью
	events := make([]Event, 0, estimatedLines)

	// Создаём индекс с предварительной ёмкостью
	index := Index{
		EventIdIndex: make(map[string]*Event, estimatedLines),
		UserIdGroup: make(map[string][]*Event, estimatedLines),
		FileIdGroup: make(map[string][]*Event, estimatedLines),
		DestinationIdGroup: make(map[string][]*Event, estimatedLines),
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

		// Проверка отрицаткльного размера
		if (newEvent.SizeBytes != nil && *newEvent.SizeBytes < 0) {
			return nil, Index{}, fmt.Errorf("%s:%d: Отрицательное значение поля size_bytes\n", filePath, lineNumber)
		}

		// Проверка дубликатов event_id
		_, isExist := index.EventIdIndex[newEvent.EventID]
		if (isExist) {
			return nil, Index{}, fmt.Errorf("%s:%d: Дублирование значения event_id %s\n", filePath, lineNumber, newEvent.EventID)
		}

		events = append(events, newEvent) // добавляем новое событие в список событий
		eventPtr := &events[len(events) - 1]
		eventPtr.LineNumber = lineNumber

		// Заполняем индекс
		index.EventIdIndex[eventPtr.EventID] = eventPtr
		index.UserIdGroup[eventPtr.UserID] = append(index.UserIdGroup[eventPtr.UserID], eventPtr)
		if (eventPtr.FileID != nil) {
			index.FileIdGroup[*eventPtr.FileID] = append(index.FileIdGroup[*eventPtr.FileID], eventPtr)
		}
		if (eventPtr.DestinationID != nil) {
			index.DestinationIdGroup[*eventPtr.DestinationID] = append(index.DestinationIdGroup[*eventPtr.DestinationID], eventPtr)
		}
	}

	err = scanner.Err()
	if err != nil {
		return nil, Index{}, fmt.Errorf("Ошибка при чтении файла %s: %w", filePath, err) // ошибка при сканировнии
	}

	return events, index, nil

}
