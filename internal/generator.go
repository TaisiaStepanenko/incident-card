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

	if (err != nil) {
		return nil, fmt.Errorf("%w", err)
	}

	return events, nil
}

func GenerateExternalSend(rng *rand.Rand, count int, baseTime time.Time) ([]Event, error) {
	events := make([]Event, count)

	if (count > 0 && count < 5) {
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
		EventID: "evt_12345",
		TimeStamp: mainEventTime.Format(time.RFC3339), 
		UserID:	user, 
		MachineID: machine,
		Department:	&department,
		Action:	"email_send", 
		Channel: "email", 
		FileID:	&fileID,
		FileName: &fileName,
		FileExt: &fileExt,
		ContentClasses: contentClasses,
		DestinationID: &destinationID,
		DestinationType: &destinationType,
		Destination: &destination,
		SizeBytes: int64SizeBytes(204800),
		Severity: strPtr("high"),
	}

	// Событие до (открытие файла)
	beforeMainEvent := mainEventTime.Add(-10 * time.Minute)
	events[1] = Event{
		EventID: "evt_12338",
		TimeStamp: beforeMainEvent.Format(time.RFC3339), 
		UserID:	user, 
		MachineID: machine,
		Department:	&department,
		Action:	"open_file", 
		Channel: "local", 
		FileID:	&fileID,
		FileName: &fileName,
		FileExt: &fileExt,
		ContentClasses: contentClasses,
		SizeBytes: int64SizeBytes(204800),
		Severity: strPtr("low"),
	}

	// Событие до (создание архива)
	beforeMainEvent2 := mainEventTime.Add(-6 * time.Minute)
	events[2] = Event{
		EventID: "evt_12339",
		TimeStamp: beforeMainEvent2.Format(time.RFC3339), 
		UserID:	user, 
		MachineID: machine,
		Department:	&department,
		Action:	"create_archive", 
		Channel: "local", 
		FileID:	&fileID,
		FileName: strPtr("client_base.zip"),
		FileExt: strPtr("zip"),
		ContentClasses: contentClasses,
		SizeBytes: int64SizeBytes(102400),
		Severity: strPtr("medium"),
	}

	// События после (удаление файла)
	afterMainEvent := mainEventTime.Add(40 * time.Minute)
	events[3] = Event{
		EventID: "evt_12347",
		TimeStamp: afterMainEvent.Format(time.RFC3339), 
		UserID:	user, 
		MachineID: machine,
		Department:	&department,
		Action:	"delete_file", 
		Channel: "local", 
		FileID:	&fileID,
		FileName: &fileName,
		FileExt: &fileExt,
		ContentClasses: contentClasses,
		SizeBytes: int64SizeBytes(204800),
		Severity: strPtr("high"),
	}

	// События того же пользователя, но с другим файлом (проверка связанных по пользователю файлов)
	eventTime := mainEventTime.Add(15 * time.Minute)
	events[4] = Event{
		EventID: "evt_12346",
		TimeStamp: eventTime.Format(time.RFC3339), 
		UserID:	user, 
		MachineID: machine,
		Department:	&department,
		Action:	"open_file", 
		Channel: "local", 
		FileID:	strPtr("file_012"),
		FileName: strPtr("info.pdf"),
		FileExt: strPtr("pdf"),
		ContentClasses: []string{"legal"},
		SizeBytes: int64SizeBytes(102400),
		Severity: strPtr("low"),
	}

	// Генерируем оставшиеся count-5 событий, начиная с 6, так как предыдущие 5 уже составлены
	for i := 5; i<count; i++ {
		events[i] = GenerateRandomEvent(rng, baseTime)
	}
	return events, nil
}

func GenerateUSBCopy(rng *rand.Rand, count int, baseTime time.Time) ([]Event, error) {
	events := make([]Event, count)

	if (count > 0 && count < 4) {
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
		EventID: "evt_12345",
		TimeStamp: mainEventTime.Format(time.RFC3339), 
		UserID:	user, 
		MachineID: machine,
		Department:	&department,
		Action:	"copy_to_usb", 
		Channel: "usb", 
		FileID:	&fileID,
		FileName: &fileName,
		FileExt: &fileExt,
		ContentClasses: contentClasses,
		DestinationID: &destinationID,
		DestinationType: &destinationType,
		Destination: &destination,
		SizeBytes: int64SizeBytes(102400),
		Severity: strPtr("high"),
	}

	// События до (открытие файла)
	beforeMainEvent := mainEventTime.Add(-17 * time.Minute)
	events[1] = Event{
		EventID: "evt_12338",
		TimeStamp: beforeMainEvent.Format(time.RFC3339), 
		UserID:	user, 
		MachineID: machine,
		Department:	&department,
		Action:	"open_file", 
		Channel: "local", 
		FileID:	&fileID,
		FileName: &fileName,
		FileExt: &fileExt,
		ContentClasses: contentClasses,
		SizeBytes: int64SizeBytes(102400),
		Severity: strPtr("low"),
	}

	// Событие после (удаление файла)
	afterMainEvent := mainEventTime.Add(40 * time.Minute)
	events[2] = Event{
		EventID: "evt_12347",
		TimeStamp: afterMainEvent.Format(time.RFC3339), 
		UserID:	user, 
		MachineID: machine,
		Department:	&department,
		Action:	"delete_file", 
		Channel: "local", 
		FileID:	&fileID,
		FileName: &fileName,
		FileExt: &fileExt,
		ContentClasses: contentClasses,
		SizeBytes: int64SizeBytes(204800),
		Severity: strPtr("high"),
	}

	// События того же пользователя, но с другим файлом (проверка связанных по пользователю файлов)
	eventTime := mainEventTime.Add(16 * time.Minute)
	events[3] = Event{
		EventID: "evt_12346",
		TimeStamp: eventTime.Format(time.RFC3339), 
		UserID:	user, 
		MachineID: machine,
		Department:	&department,
		Action:	"open_file", 
		Channel: "local", 
		FileID:	strPtr("file_012"),
		FileName: strPtr("info.pdf"),
		FileExt: strPtr("pdf"),
		ContentClasses: []string{"legal"},
		SizeBytes: int64SizeBytes(102400),
		Severity: strPtr("low"),
	}

	// Генерируем оставшиеся count-4 событий, начиная с 5, так как предыдущие 4 уже составлены
	for i := 4; i<count; i++ {
		events[i] = GenerateRandomEvent(rng, baseTime)
	}
	return events, nil
}

func GenerateCloudUpload(rng *rand.Rand, count int, baseTime time.Time) ([]Event, error) {
	events := make([]Event, count)

	if (count > 0 && count < 4) {
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
		EventID: "evt_12345",
		TimeStamp: mainEventTime.Format(time.RFC3339), 
		UserID:	user, 
		MachineID: machine,
		Department:	&department,
		Action:	"cloud_upload", 
		Channel: "cloud", 
		FileID:	&fileID,
		FileName: &fileName,
		FileExt: &fileExt,
		ContentClasses: contentClasses,
		DestinationID: &destinationID,
		DestinationType: &destinationType,
		Destination: &destination,
		SizeBytes: int64SizeBytes(4476205),
		Severity: strPtr("high"),
	}

	// События до (создание архива)
	beforeMainEvent := mainEventTime.Add(-20 * time.Minute)
	events[1] = Event{
		EventID: "evt_12338",
		TimeStamp: beforeMainEvent.Format(time.RFC3339), 
		UserID:	user, 
		MachineID: machine,
		Department:	&department,
		Action:	"create_archive", 
		Channel: "local", 
		FileID:	&fileID,
		FileName: &fileName,
		FileExt: &fileExt,
		ContentClasses: contentClasses,
		SizeBytes: int64SizeBytes(4476205),
		Severity: strPtr("medium"),
	}

	// Событие после (удаление файла)
	afterMainEvent := mainEventTime.Add(40 * time.Minute)
	events[2] = Event{
		EventID: "evt_12347",
		TimeStamp: afterMainEvent.Format(time.RFC3339), 
		UserID:	user, 
		MachineID: machine,
		Department:	&department,
		Action:	"delete_file", 
		Channel: "local", 
		FileID:	&fileID,
		FileName: &fileName,
		FileExt: &fileExt,
		ContentClasses: contentClasses,
		SizeBytes: int64SizeBytes(4476205),
		Severity: strPtr("high"),
	}

	// События того же пользователя, но с другим файлом (проверка связанных по пользователю файлов)
	eventTime := mainEventTime.Add(16 * time.Minute)
	events[3] = Event{
		EventID: "evt_12346",
		TimeStamp: eventTime.Format(time.RFC3339), 
		UserID:	user, 
		MachineID: machine,
		Department:	&department,
		Action:	"open_file", 
		Channel: "local", 
		FileID:	strPtr("file_015"),
		FileName: strPtr("info.pdf"),
		FileExt: strPtr("pdf"),
		ContentClasses: []string{"legal"},
		SizeBytes: int64SizeBytes(4476205),
		Severity: strPtr("low"),
	}

	// Генерируем оставшиеся count-4 событий, начиная с 5, так как предыдущие 4 уже составлены
	for i := 4; i<count; i++ {
		events[i] = GenerateRandomEvent(rng, baseTime)
	}
	return events, nil
}

func GenerateRandomEvent(rng *rand.Rand, baseTime time.Time) Event {
	// Списки возможных значений
	users := []string{"user_015", "user_016", "user_017", "user_018", "user_019", "user_020"}
	machines := []string{"pc_001", "pc_002", "pc_003", "pc_004", "pc_005"}
	departments := []string{ "sales", "hr", "finance", "dev", "legal", "support"}
	actions := []string{ "open_file", "copy_file", "create_archive", "email_send", "cloud_upload", "messenger_send", "copy_to_usb", "delete_file", "print_file"}
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
	sizeBytes := int64(rng.Intn(1000000)) // до 1 MБ
	eventID := fmt.Sprintf("evt_%06d", 13000+rng.Intn(100000))  // начинаем с 13000, чтобы точно избежать повторного использования ID
	fileID := fmt.Sprintf("file_%03d", rng.Intn(100) + 1)

	timestamp := baseTime.Add(time.Duration(rng.Intn(2*24*60*60)) * time.Second).Format(time.RFC3339) // 2*24*60*60 - количесвто секунд в 2 днях

	return Event{
		EventID: eventID,
		TimeStamp: timestamp, 
		UserID:	user, 
		MachineID: machine,
		Department:	&department,
		Action:	action, 
		Channel: channel, 
		FileID:	&fileID,
		FileName: &fileName,
		FileExt: &fileExt,
		ContentClasses: contentClass,
		SizeBytes: &sizeBytes,
		Severity: &severity,
	}
}
