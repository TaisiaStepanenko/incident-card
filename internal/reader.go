package internal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

// Чтение событий из JSONL-файла
func ReadEvents(filePath string) ([]Event, error) {
	file, err := os.Open(filePath)
	if (err != nil) {
		return nil, fmt.Errorf("Не удалось открыть файл %s: %w", filePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	
	var events []Event
	lineNumber := 0
	const maxLineLength = 10 * 1024 * 1024

	buffer := make([]byte, 0, maxLineLength + maxLineLength)
	scanner.Buffer(buffer, maxLineLength + maxLineLength)

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		if (line == "") {
			continue   // пропускаем пустые строки
		}

		if (len(line) > maxLineLength) {
			fmt.Fprintf(os.Stderr, "Строка %d слишком длинная\n", lineNumber) // проверяем слишком длинные значения
			continue
		}

		var newEvent Event

		err := json.Unmarshal([]byte(line), &newEvent); // декодирование JSON

		if (err != nil) {
			fmt.Fprintf(os.Stderr, "Ошибка парсинга строки %d: %v\n", lineNumber, err)
			continue
		}

		events = append(events, newEvent) // добавляем новое событие в список событий
	}

	err = scanner.Err()
	if (err != nil) {
		return nil, fmt.Errorf("Ошибка при чтении файла %s: %w", filePath, err)  // ошибка при сканировнии
	}

	return events, err

}