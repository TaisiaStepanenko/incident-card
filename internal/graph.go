package internal

import (
	"fmt"
	"strings"
)

func GenerateDOTGraph(answer *Answer, index Index) (string, error) {

	if len(answer.TimeLine) == 0 {
		return "", fmt.Errorf("Нет событий для построения графа")
	}

	var graph strings.Builder

	graph.WriteString("digraph IncidentGraph {\n") // объявляем ориентированный граф
	graph.WriteString(" rankdir=LR;\n")            // расположение узлов слева на право (для отображения временного потока)
	graph.WriteString(" node[shape=box];\n")       // форма для всех узлов прямоугольник по умолчанию

	// множества добавленных в граф узлов и рёбер
	nodes := make(map[string]bool)
	edges := make(map[string]bool)

	// Добавляем узел главного события (помечен голубым цветом)
	mainEvent, _ := index.GetEvent(answer.MainEvent.EventID)
	if mainEvent == nil {
		return "", fmt.Errorf("Главное событие не найдено в индексе.")
	}
	mainEventLabel := fmt.Sprintf("%s\\n%s\\n%s", answer.MainEvent.EventID, answer.MainEvent.Action, mainEvent.TimeStamp)
	graph.WriteString(fmt.Sprintf(" %s [style=filled, fillcolor=lightblue, label=%s];\n", 	escapeDOTString(answer.MainEvent.EventID), escapeDOTString(mainEventLabel)))
	
	nodes[answer.MainEvent.EventID] = true

	addEventEdges(&graph, mainEvent, nodes, edges)

	// Проходим по всем событиям Timeline и добавляем узлы ещё не добавленных событий
	for _, event := range answer.TimeLine {
		if event.EventID == answer.MainEvent.EventID {
			continue
		}

		eventID := event.EventID
		if !nodes[eventID] {
			action := fmt.Sprintf("%s\\n%s\\n%s", eventID, event.Action, event.Timestamp)
			graph.WriteString(fmt.Sprintf(" %s [label=%s];\n", escapeDOTString(eventID), escapeDOTString(action)))
			nodes[eventID] = true
		}

		// через index.GetEvent получаем всю информацию о событии для вывода file_id и destination_id
		ev, isExist := index.GetEvent(eventID)
		if !isExist {
			continue
		}

		addEventEdges(&graph, ev, nodes, edges)
	}

	graph.WriteString("}\n")

	return graph.String(), nil
}

func addEventEdges(graph *strings.Builder, event *Event, nodes, edges map[string]bool) {

	// узел и ребро пользователя
	if event.UserID != "" {
		userNode := event.UserID
		if !nodes[userNode] {
			graph.WriteString(fmt.Sprintf(" %s [shape=ellipse, label=%s];\n", escapeDOTString(userNode), escapeDOTString(event.UserID)))
			nodes[userNode] = true
		}
		edge := fmt.Sprintf("%s->%s", event.EventID, userNode)
		if !edges[edge] {
			graph.WriteString(fmt.Sprintf(" %s -> %s [label=%s];\n", escapeDOTString(event.EventID), escapeDOTString(userNode), escapeDOTString("performed by")))
			edges[edge] = true
		}
	}

	// узел и ребро файла
	if event.FileID != "" {
		fileNode := event.FileID
		if !nodes[fileNode] {
			file := event.FileID
			if event.FileName != "" {
				file += "\\n" + event.FileName
			}
			graph.WriteString(fmt.Sprintf(" %s [shape=note, label=%s];\n", escapeDOTString(fileNode), escapeDOTString(file)))
			nodes[fileNode] = true
		}
		edge := fmt.Sprintf("%s->%s", event.EventID, fileNode)
		if !edges[edge] {
			graph.WriteString(fmt.Sprintf(" %s -> %s [label=%s];\n", escapeDOTString(event.EventID), escapeDOTString(fileNode), escapeDOTString("uses file")))
			edges[edge] = true
		}
	}

	// узел и ребро адресата
	if event.DestinationID != "" {
		destinationNode := event.DestinationID
		if !nodes[destinationNode] {
			destination := event.DestinationID
			if event.Destination != "" {
				destination += "\\n" + event.Destination
			}
			graph.WriteString(fmt.Sprintf(" %s [shape=cylinder, label=%s];\n", escapeDOTString(destinationNode), escapeDOTString(destination)))
			nodes[destinationNode] = true
		}
		edge := fmt.Sprintf("%s->%s", event.EventID, destinationNode)
		if !edges[edge] {
			graph.WriteString(fmt.Sprintf(" %s -> %s [label=%s];\n", escapeDOTString(event.EventID), escapeDOTString(destinationNode), escapeDOTString("sends to")))
			edges[edge] = true
		}
	}
}

// Экранирование строки для использования в DOT-формате
func escapeDOTString(s string) string {
	// Заменяем кавычки 
	s = strings.ReplaceAll(s, "\"", "\\\"")

	return "\"" + s + "\""
}
