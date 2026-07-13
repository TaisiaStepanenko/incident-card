package internal

import (
	"log"
	"strconv"
	"strings"
)

func CheckRules(event *Event, rules []Rule) []string {
	var suspicious []string
	for _, factor := range rules {
		if CheckCondition(event, factor.Condition) {
			suspicious = append(suspicious, factor.Title)
		}
	}
	return suspicious
}

func CheckCondition(event *Event, cond Condition) bool {
	switch cond.Field {
	// Обязательные поля типа string
	case "event_id":
		if cond.Equals != nil && event.EventID == *cond.Equals {
			return true
		}
		if len(cond.In) > 0 {
			for _, in := range cond.In {
				if event.EventID == in {
					return true
				}
			}
		}
		if cond.Exists != nil && *cond.Exists && event.EventID != "" {
			return true
		}

	case "timestamp":
		if cond.Equals != nil && event.TimeStamp == *cond.Equals {
			return true
		}
		if len(cond.In) > 0 {
			for _, in := range cond.In {
				if event.TimeStamp == in {
					return true
				}
			}
		}
		if cond.Exists != nil && *cond.Exists && event.TimeStamp != "" {
			return true
		}

	case "user_id":
		if cond.Equals != nil && event.UserID == *cond.Equals {
			return true
		}
		if len(cond.In) > 0 {
			for _, in := range cond.In {
				if event.UserID == in {
					return true
				}
			}
		}
		if cond.Exists != nil && *cond.Exists && event.UserID != "" {
			return true
		}

	case "machine_id":
		if cond.Equals != nil && event.MachineID == *cond.Equals {
			return true
		}
		if len(cond.In) > 0 {
			for _, in := range cond.In {
				if event.MachineID == in {
					return true
				}
			}
		}
		if cond.Exists != nil && *cond.Exists && event.MachineID != "" {
			return true
		}

	case "action":
		if cond.Equals != nil && event.Action == *cond.Equals {
			return true
		}
		if len(cond.In) > 0 {
			for _, in := range cond.In {
				if event.Action == in {
					return true
				}
			}
		}
		if cond.Exists != nil && *cond.Exists && event.Action != "" {
			return true
		}

	case "channel":
		if cond.Equals != nil && event.Channel == *cond.Equals {
			return true
		}
		if len(cond.In) > 0 {
			for _, in := range cond.In {
				if event.Channel == in {
					return true
				}
			}
		}
		if cond.Exists != nil && *cond.Exists && event.Channel != "" {
			return true
		}

	// Необязательные поля типа *string
	case "department":
		// Проверяем задано ли вообще поле (так как оно не обязательное)
		if event.Department == nil {
			if cond.Exists != nil && *cond.Exists {
				return false
			}
			return false
		}
		if cond.Equals != nil && *event.Department == *cond.Equals {
			return true
		}
		if len(cond.In) > 0 {
			for _, in := range cond.In {
				if *event.Department == in {
					return true
				}
			}
		}
		if cond.Exists != nil && *cond.Exists && *event.Department != "" {
			return true
		}

	case "file_id":
		// Проверяем задано ли вообще поле (так как оно не обязательное)
		if event.FileID == nil {
			if cond.Exists != nil && *cond.Exists {
				return false
			}
			return false
		}
		if cond.Equals != nil && *event.FileID == *cond.Equals {
			return true
		}
		if len(cond.In) > 0 {
			for _, in := range cond.In {
				if *event.FileID == in {
					return true
				}
			}
		}
		if cond.Exists != nil && *cond.Exists && *event.FileID != "" {
			return true
		}

	case "file_name":
		// Проверяем задано ли вообще поле (так как оно не обязательное)
		if event.FileName == nil {
			if cond.Exists != nil && *cond.Exists {
				return false
			}
			return false
		}
		if cond.Equals != nil && *event.FileName == *cond.Equals {
			return true
		}
		if len(cond.In) > 0 {
			for _, in := range cond.In {
				if *event.FileName == in {
					return true
				}
			}
		}
		if cond.Exists != nil && *cond.Exists && *event.FileName != "" {
			return true
		}

		// Проверка содержания подстроки
		if (cond.Contains != nil && strings.Contains(*event.FileName, *cond.Contains)){
			return true
		}

	case "file_ext":
		// Проверяем задано ли вообще поле (так как оно не обязательное)
		if event.FileExt == nil {
			if cond.Exists != nil && *cond.Exists {
				return false
			}
			return false
		}
		if cond.Equals != nil && *event.FileExt == *cond.Equals {
			return true
		}
		if len(cond.In) > 0 {
			for _, in := range cond.In {
				if *event.FileExt == in {
					return true
				}
			}
		}
		if cond.Exists != nil && *cond.Exists && *event.FileExt != "" {
			return true
		}

	case "destination_id":
		// Проверяем задано ли вообще поле (так как оно не обязательное)
		if event.DestinationID == nil {
			if cond.Exists != nil && *cond.Exists {
				return false
			}
			return false
		}
		if cond.Equals != nil && *event.DestinationID == *cond.Equals {
			return true
		}
		if len(cond.In) > 0 {
			for _, in := range cond.In {
				if *event.DestinationID == in {
					return true
				}
			}
		}
		if cond.Exists != nil && *cond.Exists && *event.DestinationID != "" {
			return true
		}
	case "destination_type":
		// Проверяем задано ли вообще поле (так как оно не обязательное)
		if event.DestinationType == nil {
			if cond.Exists != nil && *cond.Exists {
				return false
			}
			return false
		}
		if cond.Equals != nil && event.DestinationType != nil && *event.DestinationType == *cond.Equals {
			return true
		}
		if len(cond.In) > 0 {
			for _, in := range cond.In {
				if *event.DestinationType == in {
					return true
				}
			}
		}
		if cond.Exists != nil && *cond.Exists && *event.DestinationType != "" {
			return true
		}
	case "destination":
		// Проверяем задано ли вообще поле (так как оно не обязательное)
		if event.Destination == nil {
			if cond.Exists != nil && *cond.Exists {
				return false
			}
			return false
		}
		if cond.Equals != nil && *event.Destination == *cond.Equals {
			return true
		}
		if len(cond.In) > 0 {
			for _, in := range cond.In {
				if *event.Destination == in {
					return true
				}
			}
		}
		if cond.Exists != nil && *cond.Exists && *event.Destination != "" {
			return true
		}
		// Проверка содержания подстроки
		if (cond.Contains != nil && strings.Contains(*event.Destination, *cond.Contains)){
			return true
		}

	case "severity":
		// Проверяем задано ли вообще поле (так как оно не обязательное)
		if event.Severity == nil {
			if cond.Exists != nil && *cond.Exists {
				return false
			}
			return false
		}
		if cond.Equals != nil && *event.Severity == *cond.Equals {
			return true
		}
		if len(cond.In) > 0 {
			for _, in := range cond.In {
				if *event.Severity == in {
					return true
				}
			}
		}
		if cond.Exists != nil && *cond.Exists && *event.Severity != "" {
			return true
		}

	// Поле типа []string
	case "content_classes":
		if (cond.Contains != nil) {
			for _, class := range event.ContentClasses {
				if (strings.Contains(class, *cond.Contains)) {
					return true
				}
			}
		}

		if (cond.Exists != nil && *cond.Exists && len(event.ContentClasses) > 0) {
			return true
		}

		// Проверка есть ли в массиве значение соответствующее cond.Equals
		if (cond.Equals != nil) {
			for _, class := range event.ContentClasses {
				if (class == *cond.Equals) {
					return true
				}
			}
		}

		if len(cond.In) > 0 {
			for _, class := range event.ContentClasses {
				for _, in := range cond.In {
					if class == in {
						return true
					}
				}
			}
		}

	// Поле типа *int64
	case "size_bytes":
		if (event.SizeBytes == nil) {
			if (cond.Exists != nil && *cond.Exists) {
				return false
			}
			return false
		}
		
		if (cond.Exists != nil && *cond.Exists && *event.SizeBytes >= 0) {
			return true
		}

		if (cond.Equals != nil) {
			size, err  := strconv.ParseInt(*cond.Equals, 10, 64)
			if (err != nil) {
				log.Printf("Ошибка парсинга числа поля equals: %v", err)
			} else if (*event.SizeBytes == size) {
				return true
			}
		}

		if (len(cond.In) > 0) {
			for _, in := range cond.In {
				size, err  := strconv.ParseInt(in, 10, 64)
				if (err != nil) {
				log.Printf("Ошибка парсинга числа поля in: %v", err)
				} else if (*event.SizeBytes == size) {
					return true
				}

			}
		}

		if (cond.Gt != nil && *event.SizeBytes > *cond.Gt) {
			return true
		}

		if (cond.Gte != nil && *event.SizeBytes >= *cond.Gte) {
			return true
		}

		if (cond.Lt != nil && *event.SizeBytes < *cond.Lt) {
			return true
		}

		if (cond.Lte != nil && *event.SizeBytes <= *cond.Lte) {
			return true
		}

		if (cond.Contains != nil) {
			log.Printf("contains is not supported for numeric field size_bytes")
			return  false
		}
	}
	return false
}


