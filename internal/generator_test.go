package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Count равно 0
func TestGenerateEventsInvalidCount(t *testing.T) {
	_, err := GenerateEvents(0, "external_send", 42)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Некорректное значение count:")
}

// Запрошен неизвестный сценарий
func TestGenerateEventsUnknownScenario(t *testing.T) {
	_, err := GenerateEvents(100, "unknown_scenario", 42)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Неизвестный сценарий")
}

// Минимально возможное корректное число событий в external_send
func TestGenerateEventsExternalSendMinCount(t *testing.T) {
	events, err := GenerateEvents(5, "external_send", 42)
	require.NoError(t, err)
	assert.Len(t, events, 5) // проверяем количество созданных событий

	// Проверяем поля главного события
	assert.Equal(t, "evt_12345", events[0].EventID)
	assert.Equal(t, "email_send", events[0].Action)
	assert.Equal(t, "external", *events[0].DestinationType)
	assert.Equal(t, "user_017", events[0].UserID)
}

// Минимально возможное корректное число событий в usb_copy
func TestGenerateEventsUSBCopyMinCount(t *testing.T) {
	events, err := GenerateEvents(4, "usb_copy", 42)
	require.NoError(t, err)
	assert.Len(t, events, 4) // проверяем количество созданных событий

	// Проверяем поля главного события
	assert.Equal(t, "evt_12345", events[0].EventID)
	assert.Equal(t, "copy_to_usb", events[0].Action)
	assert.Equal(t, "usb", *events[0].DestinationType)
	assert.Equal(t, "user_017", events[0].UserID)
}

// Минимально возможное корректное число событий в cloud_upload
func TestGenerateEventsCloudUploadMinCount(t *testing.T) {
	events, err := GenerateEvents(4, "cloud_upload", 42)
	require.NoError(t, err)
	assert.Len(t, events, 4) // проверяем количество созданных событий

	// Проверяем поля главного события
	assert.Equal(t, "evt_12345", events[0].EventID)
	assert.Equal(t, "cloud_upload", events[0].Action)
	assert.Equal(t, "cloud", *events[0].DestinationType)
	assert.Equal(t, "user_017", events[0].UserID)
}

// Count меньше минимального количества событий сценария external_send
func TestGenerateEventsExternalSendTooSmallCount(t *testing.T) {
	_, err := GenerateEvents(3, "external_send", 42)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Для данного сценария (external_send) значение count должно быть не менее 5")
}

// Count меньше минимального количества событий сценария usb_copy
func TestGenerateEventsUSBCopyTooSmallCount(t *testing.T) {
	_, err := GenerateEvents(3, "usb_copy", 42)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Для данного сценария (usb_copy) значение count должно быть не менее 4")
}

// Count меньше минимального количества событий сценария cloud_upload
func TestGenerateCloudUploadTooSmallCount(t *testing.T) {
	_, err := GenerateEvents(3, "cloud_upload", 42)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Для данного сценария (cloud_upload) значение count должно быть не менее 4")
}


// Одинаковое seed должно выдавать одинаковые результаты генерации
func TestGenerateEventsDeterministicSeed(t *testing.T) {
	events1, err1 := GenerateEvents(5, "external_send", 95)
	events2, err2 := GenerateEvents(5, "external_send", 95)

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Equal(t, events1, events2)
}

func TestGenerateEventsDiffSeed(t *testing.T) {
	events1, err1 := GenerateEvents(5, "external_send", 95)
	events2, err2 := GenerateEvents(5, "external_send", 98)

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, events1, events2)
}


// Создание ранодомных событий external_send
func TestGenerateEventsExternalRandomEvents(t *testing.T) {
	events, err := GenerateEvents(13, "external_send", 42)

	require.NoError(t, err)
	assert.Len(t, events, 13)

	// Проверяем, что первые 4 точно соответствует тем, что обязательно должны присутствовать в сценарии (предсказуемые ID)
	assert.Equal(t, "evt_12345", events[0].EventID) // главное событие
	assert.Equal(t, "evt_12338", events[1].EventID) // событие до
	assert.Equal(t, "evt_12339", events[2].EventID) // событие до
	assert.Equal(t, "evt_12347", events[3].EventID) // событие после
	assert.Equal(t, "evt_12346", events[4].EventID) // связное событие того же пользователя
}


// Создание ранодомных событий cloud_upload
func TestGenerateCloudUploadRandomEvents(t *testing.T) {
	events, err := GenerateEvents(13, "cloud_upload", 42)

	require.NoError(t, err)
	assert.Len(t, events, 13)

	// Проверяем, что первые 4 точно соответствует тем, что обязательно должны присутствовать в сценарии (предсказуемые ID)
	assert.Equal(t, "evt_12345", events[0].EventID) // главное событие
	assert.Equal(t, "evt_12338", events[1].EventID) // событие до
	assert.Equal(t, "evt_12347", events[2].EventID) // событие после
	assert.Equal(t, "evt_12346", events[3].EventID) // связное событие того же пользователя
}

// Создание ранодомных событий usb_copy
func TestGenerateUSBCopyRandomEvents(t *testing.T) {
	events, err := GenerateEvents(13, "usb_copy", 42)

	require.NoError(t, err)
	assert.Len(t, events, 13)

	// Проверяем, что первые 5 точно соответствует тем, что обязательно должны присутствовать в сценарии (предсказуемые ID)
	assert.Equal(t, "evt_12345", events[0].EventID) // главное событие
	assert.Equal(t, "evt_12338", events[1].EventID) // событие до
	assert.Equal(t, "evt_12347", events[2].EventID) // событие после
	assert.Equal(t, "evt_12346", events[3].EventID) // связное событие того же пользователя
}
