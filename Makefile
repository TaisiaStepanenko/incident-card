
## имя бинарника
BINARY_NAME := incident-card
## путь к основному пакету
MAIN_PKG := ./cmd/incident-card/main.go

TESTDATA_DIR := ./testdata/control
DEMO_EVENT_ID := evt_12345
DEMO_BEFORE := 30m
DEMO_AFTER := 10m
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
	go test ./... -bench=.


## Демо запуск на тестовом наборе данных
## зависимость build (перед demo сначала выполняется build)
demo: build
	./$(BINARY_NAME) build \
		--events $(TESTDATA_DIR)/events.jsonl --event-id $(DEMO_EVENT_ID) \
		--before $(DEMO_BEFORE) --after $(DEMO_AFTER) \
		--out $(TESTDATA_DIR)/card.md --json $(TESTDATA_DIR)/card.json \
		--factors $(TESTDATA_DIR)/factors.yaml --request $(TESTDATA_DIR)/request.json
	@echo "Демо-запуск завешён, результаты сохранены в файлы."


## Очистка сгенерированных файлов
## использовано del, так как запускается на Windows, -Q принудительное удаление
clean: 
	del /Q $(BINARY_NAME)
	del /Q $(TESTDATA_DIR)/card.md $(TESTDATA_DIR)/card.json