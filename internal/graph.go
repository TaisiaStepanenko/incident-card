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

	graph.WriteString("digraph IncidentGraph {\n")	// объявляем ориентированный граф
	graph.WriteString(" rankdir=LR;\n")		// расположение узлов слева на право (для отображения временного потока)
	graph.WriteString( "node[shape=box];\n")	// форма для всех узлов прямоугольник по умолчанию

	// множества добавленных в граф узлов и рёбер
	nodes := make(map[string]bool)
	edges := make(map[string]bool)

	// Добавляем узел главного события (помечен голубым цветом)
	ev, _ := index.GetEvent(answer.MainEvent.EventID)
	mainEvent := fmt.Sprintf("%s\\n%s\\n%s", answer.MainEvent.EventID, answer.MainEvent.Action, ev.TimeStamp)
	graph.WriteString(fmt.Sprintf(" \"%s\" [style=filled, fillcolor=lightblue, label=\"%s\"];\n", answer.MainEvent.EventID, mainEvent))
	nodes[mainEvent] = true

	// Проходим по всем событиям Timeline и добавляем узлы ещё не добавленных событий
	for _, event := range answer.TimeLine {
		eventID := event.EventID
		if (!nodes[eventID]) {
			action := fmt.Sprintf("%s\\n%s\\n%s", eventID, event.Action, event.Timestamp)
			graph.WriteString(fmt.Sprintf(" \"%s\" [label=\"%s\"];\n", eventID, action))
			nodes[eventID] = true
		}


		// если задан пользователь, то делаем узел пользователя и ребро между его узлом и событием
		if (event.UserID != "") {
			userNode := "user_" + event.UserID
			if (!nodes[userNode]) {
				graph.WriteString(fmt.Sprintf(" \"%s\" [shape=ellipse, label=\"%s\"];\n", userNode, event.UserID))
				nodes[userNode] = true
			}

			edge := fmt.Sprintf("%s->%s", eventID, userNode)
			if (!edges[edge]) {
				graph.WriteString(fmt.Sprintf(" \"%s\" -> \"%s\" [label=\"performed by\"];\n", eventID, userNode))
				edges[edge] = true
			}
		}

		// через index.GetEvent получаем всю информацию о событии для вывода file_id и destination_id
		ev, isExist := index.GetEvent(eventID)
		if (!isExist) {
			continue
		}

		if (ev.FileID != nil && *ev.FileID != "") {
			fileNode := "file_" + *ev.FileID
			if (!nodes[fileNode]) {
				file := *ev.FileID
				if (ev.FileName != nil) {
					file += "\\n" + *ev.FileName
				}
				graph.WriteString(fmt.Sprintf(" \"%s\" [shape=note, label=\"%s\"];\n", fileNode, file))
				nodes[fileNode] = true
			}
			edge := fmt.Sprintf("%s->%s", eventID, fileNode)
			if (!edges[edge]) {
				graph.WriteString(fmt.Sprintf(" \"%s\" -> \"%s\" [label=\"uses file\"];\n", eventID, fileNode))
				edges[edge] = true
			}
		}

		if (ev.DestinationID != nil && *ev.DestinationID != "") {
			destinationNode := "dest_" + *ev.DestinationID
			if (!nodes[destinationNode]) {
				destination := *ev.DestinationID
				if (ev.Destination != nil) {
					destination += "\\n" + *ev.Destination
				}
				graph.WriteString(fmt.Sprintf(" \"%s\" [shape=cylinder, label=\"%s\"];\n", destinationNode, destination))
				nodes[destinationNode] = true
			}
			edge := fmt.Sprintf("%s->%s", eventID, destinationNode)
			if (!edges[edge]) {
				graph.WriteString(fmt.Sprintf(" \"%s\" -> \"%s\" [label=\"sends to\"];\n", eventID, destinationNode))
				edges[edge] = true
			}
		}
	}

	graph.WriteString("}\n")

	return graph.String(), nil
}