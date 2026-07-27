package internal

import (
	"fmt"
	"math/rand"
	"time"
)

func GenerateEvents(count int, scenario string, seed int64) ([]Event, error) {
	if count <= 0 {
		return nil, fmt.Errorf("Некорректное значение count: %d", count)
	}

	var events []Event
	var err error
	rng := rand.New(rand.NewSource(seed))
	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(seed) * time.Hour)

	switch scenario {
	case "external_send":
		events, err = GenerateExternalSend(rng, count, baseTime)
	case "usb_copy":
		events, err = GenerateUSBCopy(rng, count, baseTime)
	case "cloud_upload":
		events, err = GenerateCloudUpload(rng, count, baseTime)
	default:
		return nil, fmt.Errorf("Неизвестный сценарий %s, доступные сценарии external_send, usb_copy, cloud_uplod", scenario)
	}

	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	return events, nil
}

func GenerateExternalSend(rng *rand.Rand, count int, baseTime time.Time) ([]Event, error) {
	events := make([]Event, count)

	if (count < 5) {
		return nil, fmt.Errorf("Для данного сценария (external_send) значение count должно быть не менее 5")
	}

	user := "user_017"
	machine := "pc_003"
	department := "sales"
	fileID := "file_778"
	fileName := "client_base.xlsx"
	fileExt := "xlsx"
	contentClasses := []string{"client_data", "personal_data"}
	destinationID := "dst_009"
	destinationType := "external"
	destination := "external_email_001"

	// Определяем главное событие
	mainEventTime := baseTime.Add(10 * time.Minute)
	events[0] = Event{
		EventID:         "evt_12345",
		TimeStamp:       mainEventTime.Format(time.RFC3339),
		UserID:          user,
		MachineID:       machine,
		Department:      &department,
		Action:          "email_send",
		Channel:         "email",
		FileID:          &fileID,
		FileName:        &fileName,
		FileExt:         &fileExt,
		ContentClasses:  contentClasses,
		DestinationID:   &destinationID,
		DestinationType: &destinationType,
		Destination:     &destination,
		SizeBytes:       int64SizeBytes(204800),
		Severity:        strPtr("high"),
	}

	// Событие до (открытие файла)
	beforeMainEvent := mainEventTime.Add(-10 * time.Minute)
	events[1] = Event{
		EventID:        "evt_12338",
		TimeStamp:      beforeMainEvent.Format(time.RFC3339),
		UserID:         user,
		MachineID:      machine,
		Department:     &department,
		Action:         "open_file",
		Channel:        "local",
		FileID:         &fileID,
		FileName:       &fileName,
		FileExt:        &fileExt,
		ContentClasses: contentClasses,
		SizeBytes:      int64SizeBytes(204800),
		Severity:       strPtr("low"),
	}

	// Событие до (создание архива)
	beforeMainEvent2 := mainEventTime.Add(-6 * time.Minute)
	events[2] = Event{
		EventID:        "evt_12339",
		TimeStamp:      beforeMainEvent2.Format(time.RFC3339),
		UserID:         user,
		MachineID:      machine,
		Department:     &department,
		Action:         "create_archive",
		Channel:        "local",
		FileID:         &fileID,
		FileName:       strPtr("client_base.zip"),
		FileExt:        strPtr("zip"),
		ContentClasses: contentClasses,
		SizeBytes:      int64SizeBytes(102400),
		Severity:       strPtr("medium"),
	}

	// События после (удаление файла)
	afterMainEvent := mainEventTime.Add(40 * time.Minute)
	events[3] = Event{
		EventID:        "evt_12347",
		TimeStamp:      afterMainEvent.Format(time.RFC3339),
		UserID:         user,
		MachineID:      machine,
		Department:     &department,
		Action:         "delete_file",
		Channel:        "local",
		FileID:         &fileID,
		FileName:       &fileName,
		FileExt:        &fileExt,
		ContentClasses: contentClasses,
		SizeBytes:      int64SizeBytes(204800),
		Severity:       strPtr("high"),
	}

	// События того же пользователя, но с другим файлом (проверка связанных по пользователю файлов)
	eventTime := mainEventTime.Add(15 * time.Minute)
	events[4] = Event{
		EventID:        "evt_12346",
		TimeStamp:      eventTime.Format(time.RFC3339),
		UserID:         user,
		MachineID:      machine,
		Department:     &department,
		Action:         "open_file",
		Channel:        "local",
		FileID:         strPtr("file_012"),
		FileName:       strPtr("info.pdf"),
		FileExt:        strPtr("pdf"),
		ContentClasses: []string{"legal"},
		SizeBytes:      int64SizeBytes(102400),
		Severity:       strPtr("low"),
	}

	// Генерируем оставшиеся count-5 событий, начиная с индекса 5, так как предыдущие 5 уже составлены
	for i := 5; i < count; i++ {
		events[i] = GenerateRandomEvent(rng, baseTime, i - 5) // индекс с 0
	}
	return events, nil
}

func GenerateUSBCopy(rng *rand.Rand, count int, baseTime time.Time) ([]Event, error) {
	events := make([]Event, count)

	if (count < 4) {
		return nil, fmt.Errorf("Для данного сценария (usb_copy) значение count должно быть не менее 4")
	}

	user := "user_017"
	machine := "pc_003"
	department := "finance"
	fileID := "file_779"
	fileName := "report_finance.xlsx"
	fileExt := "xlsx"
	contentClasses := []string{"finance"}
	destinationID := "dst_010"
	destinationType := "usb"
	destination := "usb_device_001"

	// Определяем главное событие
	mainEventTime := baseTime.Add(15 * time.Minute)
	events[0] = Event{
		EventID:         "evt_12345",
		TimeStamp:       mainEventTime.Format(time.RFC3339),
		UserID:          user,
		MachineID:       machine,
		Department:      &department,
		Action:          "copy_to_usb",
		Channel:         "usb",
		FileID:          &fileID,
		FileName:        &fileName,
		FileExt:         &fileExt,
		ContentClasses:  contentClasses,
		DestinationID:   &destinationID,
		DestinationType: &destinationType,
		Destination:     &destination,
		SizeBytes:       int64SizeBytes(102400),
		Severity:        strPtr("high"),
	}

	// События до (открытие файла)
	beforeMainEvent := mainEventTime.Add(-17 * time.Minute)
	events[1] = Event{
		EventID:        "evt_12338",
		TimeStamp:      beforeMainEvent.Format(time.RFC3339),
		UserID:         user,
		MachineID:      machine,
		Department:     &department,
		Action:         "open_file",
		Channel:        "local",
		FileID:         &fileID,
		FileName:       &fileName,
		FileExt:        &fileExt,
		ContentClasses: contentClasses,
		SizeBytes:      int64SizeBytes(102400),
		Severity:       strPtr("low"),
	}

	// Событие после (удаление файла)
	afterMainEvent := mainEventTime.Add(40 * time.Minute)
	events[2] = Event{
		EventID:        "evt_12347",
		TimeStamp:      afterMainEvent.Format(time.RFC3339),
		UserID:         user,
		MachineID:      machine,
		Department:     &department,
		Action:         "delete_file",
		Channel:        "local",
		FileID:         &fileID,
		FileName:       &fileName,
		FileExt:        &fileExt,
		ContentClasses: contentClasses,
		SizeBytes:      int64SizeBytes(204800),
		Severity:       strPtr("high"),
	}

	// События того же пользователя, но с другим файлом (проверка связанных по пользователю файлов)
	eventTime := mainEventTime.Add(16 * time.Minute)
	events[3] = Event{
		EventID:        "evt_12346",
		TimeStamp:      eventTime.Format(time.RFC3339),
		UserID:         user,
		MachineID:      machine,
		Department:     &department,
		Action:         "open_file",
		Channel:        "local",
		FileID:         strPtr("file_012"),
		FileName:       strPtr("info.pdf"),
		FileExt:        strPtr("pdf"),
		ContentClasses: []string{"legal"},
		SizeBytes:      int64SizeBytes(102400),
		Severity:       strPtr("low"),
	}

	// Генерируем оставшиеся count-4 событий, начиная с индекса 4, так как предыдущие 4 уже составлены
	for i := 4; i < count; i++ {
		events[i] = GenerateRandomEvent(rng, baseTime, i - 4) // индекс с 0
	}
	return events, nil
}

func GenerateCloudUpload(rng *rand.Rand, count int, baseTime time.Time) ([]Event, error) {
	events := make([]Event, count)

	if (count < 4) {
		return nil, fmt.Errorf("Для данного сценария (cloud_upload) значение count должно быть не менее 4")
	}

	user := "user_017"
	machine := "pc_003"
	department := "dev"
	fileID := "file_780"
	fileName := "project.zip"
	fileExt := "zip"
	contentClasses := []string{"source_code"}
	destinationID := "dst_012"
	destinationType := "cloud"
	destination := "cloud_storage_001"

	// Определяем главное событие
	mainEventTime := baseTime.Add(17 * time.Minute)
	events[0] = Event{
		EventID:         "evt_12345",
		TimeStamp:       mainEventTime.Format(time.RFC3339),
		UserID:          user,
		MachineID:       machine,
		Department:      &department,
		Action:          "cloud_upload",
		Channel:         "cloud",
		FileID:          &fileID,
		FileName:        &fileName,
		FileExt:         &fileExt,
		ContentClasses:  contentClasses,
		DestinationID:   &destinationID,
		DestinationType: &destinationType,
		Destination:     &destination,
		SizeBytes:       int64SizeBytes(4476205),
		Severity:        strPtr("high"),
	}

	// События до (создание архива)
	beforeMainEvent := mainEventTime.Add(-20 * time.Minute)
	events[1] = Event{
		EventID:        "evt_12338",
		TimeStamp:      beforeMainEvent.Format(time.RFC3339),
		UserID:         user,
		MachineID:      machine,
		Department:     &department,
		Action:         "create_archive",
		Channel:        "local",
		FileID:         &fileID,
		FileName:       &fileName,
		FileExt:        &fileExt,
		ContentClasses: contentClasses,
		SizeBytes:      int64SizeBytes(4476205),
		Severity:       strPtr("medium"),
	}

	// Событие после (удаление файла)
	afterMainEvent := mainEventTime.Add(40 * time.Minute)
	events[2] = Event{
		EventID:        "evt_12347",
		TimeStamp:      afterMainEvent.Format(time.RFC3339),
		UserID:         user,
		MachineID:      machine,
		Department:     &department,
		Action:         "delete_file",
		Channel:        "local",
		FileID:         &fileID,
		FileName:       &fileName,
		FileExt:        &fileExt,
		ContentClasses: contentClasses,
		SizeBytes:      int64SizeBytes(4476205),
		Severity:       strPtr("high"),
	}

	// События того же пользователя, но с другим файлом (проверка связанных по пользователю файлов)
	eventTime := mainEventTime.Add(16 * time.Minute)
	events[3] = Event{
		EventID:        "evt_12346",
		TimeStamp:      eventTime.Format(time.RFC3339),
		UserID:         user,
		MachineID:      machine,
		Department:     &department,
		Action:         "open_file",
		Channel:        "local",
		FileID:         strPtr("file_015"),
		FileName:       strPtr("info.pdf"),
		FileExt:        strPtr("pdf"),
		ContentClasses: []string{"legal"},
		SizeBytes:      int64SizeBytes(4476205),
		Severity:       strPtr("low"),
	}

	// Генерируем оставшиеся count-4 событий, начиная с индекса, так как предыдущие 4 уже составлены
	for i := 4; i < count; i++ {
		events[i] = GenerateRandomEvent(rng, baseTime, i - 4) // индекс с 0
	}
	return events, nil
}

func GenerateRandomEvent(rng *rand.Rand, baseTime time.Time, index int) Event {
	// Списки возможных значений
	users := []string{"user_015", "user_016", "user_017", "user_018", "user_019", "user_020"}
	machines := []string{"pc_001", "pc_002", "pc_003", "pc_004", "pc_005"}
	departments := []string{"sales", "hr", "finance", "dev", "legal", "support"}
	actions := []string{"open_file", "copy_file", "create_archive", "email_send", "cloud_upload", "messenger_send", "copy_to_usb", "delete_file", "print_file"}
	channels := []string{"local", "email", "usb", "cloud", "messenger", "printer"}
	fileNames := []string{"doc.docx", "report.xlsx", "data.pdf", "archive.zip", "main.go", "backup.sql", "test.txt"}
	fileExts := []string{"docx", "xlsx", "pdf", "zip", "go", "sql", "txt"}
	contentClasses := [][]string{{"client_data"}, {"personal_data"}, {"finance"}, {"source_code"}, {"legal"}, {"none"}}
	severitis := []string{"low", "medium", "high", "critical"}

	user := users[rng.Intn(len(users))]
	machine := machines[rng.Intn(len(machines))]
	department := departments[rng.Intn(len(departments))]
	action := actions[rng.Intn(len(actions))]
	channel := channels[rng.Intn(len(channels))]
	fileName := fileNames[rng.Intn(len(fileNames))]
	fileExt := fileExts[rng.Intn(len(fileExts))]
	contentClass := contentClasses[rng.Intn(len(contentClasses))]
	severity := severitis[rng.Intn(len(severitis))]
	sizeBytes := int64(rng.Intn(1000000))                      // до 1 MБ
	// Используем переданных индекс с смещением 100000 для уникальности значений и детерминизма
	eventID := fmt.Sprintf("evt_%07d", index+100000)
	fileID := fmt.Sprintf("file_%03d", rng.Intn(100)+1)

	timestamp := baseTime.Add(time.Duration(rng.Intn(2*24*60*60)) * time.Second).Format(time.RFC3339) // 2*24*60*60 - количесвто секунд в 2 днях

	return Event{
		EventID:        eventID,
		TimeStamp:      timestamp,
		UserID:         user,
		MachineID:      machine,
		Department:     &department,
		Action:         action,
		Channel:        channel,
		FileID:         &fileID,
		FileName:       &fileName,
		FileExt:        &fileExt,
		ContentClasses: contentClass,
		SizeBytes:      &sizeBytes,
		Severity:       &severity,
	}
}

// Создаёт события по событиям с повторяющимися связями и различными сценариями событий
func GenerateStructuredBenchmarkEvents(count int) []Event {
	rng := rand.New(rand.NewSource(42))
	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	events := make([]Event, count)

	var event Event
	// Определяем сценарий (external_send 60% от всех событий, usb-copy - 20%, cloud_upload - 20%)
	for i := 0; i < count; i++ {
		scenario := i % 5

		switch scenario {
		case 0, 1, 2:
			event = GenerateExternalSendEvent(rng, baseTime, i)
		case 3:
			event = GenerateUSBCopyEvent(rng, baseTime, i)
		case 4:
			event = GenerateCloudUploadEvent(rng, baseTime, i)
		}
		events[i] = event
	}
	return events
}

// Создаёт событие по сценарию external_send
func GenerateExternalSendEvent(rng *rand.Rand, baseTime time.Time, count int) Event {
	users := []string{"user_015", "user_016", "user_017", "user_018", "user_019", "user_020"}
	machines := []string{"pc_001", "pc_002", "pc_003", "pc_004", "pc_005"}
	departments := []string{"sales", "hr", "finance", "dev", "legal", "support"}
	actions := []string{"open_file", "create_archive", "email_send", "delete_file"}
	channels := []string{"local", "email"}

	// повторяющиеся id для связей 
	userIdx := count % len(users)
	fileIdx := (count / 2) % 10  // 10 уникальных значений, каждое из которых используется несколько раз
	destIdx := count % 3 // 3 уникальных значений, каждое из которых используется несколько раз

	user := users[userIdx]
	machine := machines[count%len(machines)]
	department := departments[count%len(departments)]
	action := actions[rng.Intn(len(actions))]
	channel := channels[rng.Intn(len(channels))]
	fileID := fmt.Sprintf("file_%03d", 100+fileIdx)
	fileName := fmt.Sprintf("client_data_%d.xlsx", fileIdx)
	fileExt := "xlsx"
	destID := fmt.Sprintf("dst_%03d", 400+destIdx)
	destType := "external"
	dest := fmt.Sprintf("external_email_%d@mail.com", destIdx)
	contentClass := []string{"client_data", "personal_data"}
	severity := "high"
	sizeBytes := int64(102400 + rng.Intn(204800))    
	
	timestamp := baseTime.Add(time.Duration(rng.Intn(2*24*60*60)) *time.Second).Format(time.RFC3339)
	eventID := fmt.Sprintf("evt_%07d", count+1)

	return Event{
		EventID:        eventID,
		TimeStamp:      timestamp,
		UserID:         user,
		MachineID:      machine,
		Department:     &department,
		Action:         action,
		Channel:        channel,
		FileID:         &fileID,
		FileName:       &fileName,
		FileExt:        &fileExt,
		ContentClasses: contentClass,
		DestinationID: 	&destID,
		DestinationType: &destType,
		Destination: 	&dest,
		SizeBytes:      &sizeBytes,
		Severity:       &severity,
	}
}

// Создаёт событие по сценарию usb_copy
func GenerateUSBCopyEvent(rng *rand.Rand, baseTime time.Time, count int) Event {
	users := []string{"user_017", "user_018", "user_019", "user_020", "user_021"}
	machines := []string{"pc_003", "pc_004", "pc_005"}
	actions := []string{"copy_to_usb", "open_file", "delete_file"}
	channels := []string{"local", "usb"}

	// повторяющиеся id для связей 
	userIdx := count % len(users)
	fileIdx := (count / 3) % 8  // 8 уникальных значений, каждое из которых используется несколько раз
	destIdx := count % 2 // 2 уникальных значений, каждое из которых используется несколько раз

	user := users[userIdx]
	machine := machines[count%len(machines)]
	action := actions[rng.Intn(len(actions))]
	channel := channels[rng.Intn(len(channels))]
	fileID := fmt.Sprintf("file_%03d", 300+fileIdx)
	fileName := fmt.Sprintf("report_%d.xlsx", fileIdx)
	fileExt := "xlsx"
	destID := fmt.Sprintf("dst_%03d", 300+destIdx)
	destType := "usb"
	dest := fmt.Sprintf("usb_device_%02d", destIdx+1)
	contentClass := []string{"finance"}
	severity := "high"
	sizeBytes := int64(102400 + rng.Intn(102400))    
	
	timestamp := baseTime.Add(time.Duration(rng.Intn(2*24*60*60)) *time.Second).Format(time.RFC3339)
	eventID := fmt.Sprintf("evt_%07d", count+1)

	return Event{
		EventID:        eventID,
		TimeStamp:      timestamp,
		UserID:         user,
		MachineID:      machine,
		Department:     strPtr("finance"),
		Action:         action,
		Channel:        channel,
		FileID:         &fileID,
		FileName:       &fileName,
		FileExt:        &fileExt,
		ContentClasses: contentClass,
		DestinationID: 	&destID,
		DestinationType: &destType,
		Destination: 	&dest,
		SizeBytes:      &sizeBytes,
		Severity:       &severity,
	}
}

// Создаёт событие по сценарию cloud_upload
func GenerateCloudUploadEvent(rng *rand.Rand, baseTime time.Time, count int) Event {
	users := []string{"user_016", "user_017", "user_018", "user_019", "user_020"}
	machines := []string{"pc_001", "pc_002", "pc_003"}
	actions := []string{"cloud_upload", "create_archive", "delete_file"}
	channels := []string{"local", "cloud"}

	// повторяющиеся id для связей 
	userIdx := count % len(users)
	fileIdx := (count / 4) % 6  // 6 уникальных значений, каждое из которых используется несколько раз
	destIdx := count % 2 // 2 уникальных значений, каждое из которых используется несколько раз

	user := users[userIdx]
	machine := machines[count%len(machines)]
	action := actions[rng.Intn(len(actions))]
	channel := channels[rng.Intn(len(channels))]
	fileID := fmt.Sprintf("file_%03d", 500+fileIdx)
	fileName := fmt.Sprintf("report_%d.zip", fileIdx)
	fileExt := "zip"
	destID := fmt.Sprintf("dst_%03d", 400+destIdx)
	destType := "cloud"
	dest := fmt.Sprintf("cloud_storage_%02d", destIdx+1)
	contentClass := []string{"source_code"}
	severity := "medium"
	sizeBytes := int64(204800 + rng.Intn(4096000))    
	
	timestamp := baseTime.Add(time.Duration(rng.Intn(2*24*60*60)) *time.Second).Format(time.RFC3339)
	eventID := fmt.Sprintf("evt_%07d", count+1)

	return Event{
		EventID:        eventID,
		TimeStamp:      timestamp,
		UserID:         user,
		MachineID:      machine,
		Department:     strPtr("dev"),
		Action:         action,
		Channel:        channel,
		FileID:         &fileID,
		FileName:       &fileName,
		FileExt:        &fileExt,
		ContentClasses: contentClass,
		DestinationID: 	&destID,
		DestinationType: &destType,
		Destination: 	&dest,
		SizeBytes:      &sizeBytes,
		Severity:       &severity,
	}
}