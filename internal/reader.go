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
func ReadEvents(filePath string) ([]Event, []LinkInFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("Не удалось открыть файл %s: %w", filePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	var events []Event
	var eventsLinkInFile []LinkInFile
	lineNumber := 0
	const maxLineLength = 10 * 1024 * 1024

	buffer := make([]byte, 0, maxLineLength+maxLineLength)
	scanner.Buffer(buffer, maxLineLength+maxLineLength)
	existId := make(map[string]bool)

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue // пропускаем пустые строки
		}

		if (lineNumber == 1 && strings.HasPrefix(line, "\xEF\xBB\xBF")) {
			line = strings.TrimPrefix(line, "\xEF\xBB\xBF")
		}

		if len(line) > maxLineLength {
			return nil, nil, fmt.Errorf("%s:%d: cтрока слишком длинная\n", filePath, lineNumber) // проверяем слишком длинные значения
		}

		var newEvent Event

		err := json.Unmarshal([]byte(line), &newEvent) // декодирование JSON

		if err != nil {
			return nil, nil, fmt.Errorf("%s:%d: Ошибка парсинга строки: %v\n", filePath, lineNumber, err)
		}

		// Проверка обязательных полей
		if (newEvent.EventID == "" || newEvent.TimeStamp == "" || newEvent.UserID == "" || newEvent.MachineID == "" || newEvent.Action == "" || newEvent.Channel == "") {
			return nil, nil, fmt.Errorf("%s:%d: Пропущено обязательное поле\n", filePath, lineNumber)
		}

		// Проверка формата времени RFC3339
		_, err = time.Parse(time.RFC3339, newEvent.TimeStamp)
		if (err != nil) {
			return nil, nil, fmt.Errorf("%s:%d: Неверный формат поля timestamp: %w\n", filePath, lineNumber, err)
		}

		// Проверка отрицаткльного размера
		if (newEvent.SizeBytes != nil && *newEvent.SizeBytes < 0) {
			return nil, nil, fmt.Errorf("%s:%d: Отрицательное значение поля size_bytes\n", filePath, lineNumber)
		}

		// Проверка дубликатов event_id
		if (existId[newEvent.EventID]) {
			return nil, nil, fmt.Errorf("%s:%d: Дублирование значения event_id %s\n", filePath, lineNumber, newEvent.EventID)
		}
		existId[newEvent.EventID] = true

		// Добавляем новую ссылку на событие
		eventsLinkInFile = append(eventsLinkInFile, LinkInFile{
			EventID:  newEvent.EventID,
			FileName: filePath,
			FileLine: lineNumber,
		})

		events = append(events, newEvent) // добавляем новое событие в список событий
	}

	err = scanner.Err()
	if err != nil {
		return nil, nil, fmt.Errorf("Ошибка при чтении файла %s: %w", filePath, err) // ошибка при сканировнии
	}

	return events, eventsLinkInFile, nil

}
