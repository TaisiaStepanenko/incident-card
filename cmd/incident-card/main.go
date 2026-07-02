package main

import (
	"fmt"
	"log"

	"github.com/TaisiaStepanenko/incident-card/internal"
)

func main() {
	events, err := internal.ReadEvents("testdata/eventsList.jsonl")
	if err != nil {
		log.Fatalf("Ошибка: %v", err)
	}
	fmt.Printf("Прочитано %d событий\n", len(events))
	for _, ev := range events {
		fmt.Printf("EventID: %s, User: %s, Action: %s\n", ev.EventID, ev.UserID, ev.Action)
	}
}