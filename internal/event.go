package internal

type Event struct {
	EventID         string   `json:"event_ID"`
	TimeStamp       string   `json:"timestamp"`
	UserID          string   `json:"user_ID"`
	MachineID       string   `json:"machine_ID"`
	Department      *string  `json:"department,omitempty"`
	Action          string   `json:"action"`
	Channel         string   `json:"channel"`
	FileID          *string  `json:"file_ID,omitempty"`
	FileName        *string  `json:"file_name,omitempty"`
	FileExt         *string  `json:"file_ext,omitempty"`
	ContentClasses  []string `json:"content_classes,omitempty"`
	DestinationID   *string  `json:"destination_ID,omitempty"`
	DestinationType *string  `json:"destination_type,omitempty"`
	Destination     *string  `json:"destination,omitempty"`
	SizeBytes       *int64   `json:"size_bytes,omitempty"`
	Severity        *string  `json:"severity,omitempty"`
}

type Request struct {
	IncidentID             string `json:"incIDent_ID,omitempty"`
	MainEventID            string `json:"main_event_ID"`
	WindowBefore           string `json:"window_before"`
	WindowAfter            string `json:"window_after"`
	IncludeSameUser        *bool  `json:"include_same_user,omitempty"`
	IncludeSameFile        *bool  `json:"include_same_file,omitempty"`
	IncludeSameDestination *bool  `json:"include_same_destination,omitempty"`
	MaxEventsPerSection    int    `json:"max_events_per_section,omitempty"`
}

type MainEvent struct {
	EventID string `json:"event_ID"`
	Action  string `json:"action"`
}

type Answer struct {
	IncIDentID            string    `json:"incIDent_ID,omitempty"`
	MainEvent             MainEvent `json:"main_event"`
	Summary               string    `json:"summary"`
	ContextBefore         []string  `json:"context_before,omitempty"`
	ContextAfter          []string  `json:"context_after,omitempty"`
	SameUserEvents        *[]string `json:"same_user_events,omitempty"`
	SameFileEvents        *[]string `json:"same_file_events,omitempty"`
	SameDestinationEvents *[]string `json:"same_destination_events,omitempty"`
	SuspiciousFactors     *[]string `json:"suspicious_factors,omitempty"`
}

type Condition struct {
	Field    string   `json:"field"`
	Equals   *string  `json:"equals,omitempty"`
	In       []string `json:"in,omitempty"`
	Contains *string  `json:"contains,omitempty"`
	Exists   *bool    `json:"exists,omitempty"`
}

type Rules struct {
	FactorID  string    `json:"factor_ID"`
	Title     string    `json:"title"`
	Condition Condition `json:"condition"`
}