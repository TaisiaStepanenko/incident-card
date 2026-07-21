
## имя бинарника
BINARY_NAME := incident-card.exe
## путь к основному пакету
MAIN_PKG := ./cmd/incident-card/main.go

TESTDATA_DIR := ./testdata/control
TESTDATA_DIR_FOR_CLEAN := .\testdata\control
.PHONY: build test bench demo clean

## Сборка бинарного файла
## -o задаёт имя выходного файла
build:
	go build -o ./$(BINARY_NAME) $(MAIN_PKG)

## Запуск тестов
## -cover для вывода показателя покрытия кода
test: 
	go test ./... -cover

## Запуск бенчмарков
bench:
	go test ./... -bench=. -benchmem


## Демо запуск на тестовом наборе данных
## зависимость build (перед demo сначала выполняется build)
demo: build
	./$(BINARY_NAME) build \
		--events $(TESTDATA_DIR)/events.jsonl \
		--request $(TESTDATA_DIR)/request.json \
		--factors $(TESTDATA_DIR)/factors.yaml \
		--out $(TESTDATA_DIR)/card.md \
		--json $(TESTDATA_DIR)/card.json \
		--dot $(TESTDATA_DIR)/graph.dot
	@echo "Демо-запуск завешён, результаты сохранены в файлы."


## Очистка сгенерированных файлов
## использовано del, так как запускается на Windows, -Q принудительное удаление
clean: 
	del /Q $(BINARY_NAME)
	del /Q $(TESTDATA_DIR_FOR_CLEAN)\card.md
	del /Q $(TESTDATA_DIR_FOR_CLEAN)\card.json
	del /Q $(TESTDATA_DIR_FOR_CLEAN)\graph.dot
	del /Q $(TESTDATA_DIR_FOR_CLEAN)\graph.png