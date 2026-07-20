package internal

func strPtr(s string) *string {
	return &s
}

func int64SizeBytes(i int64) *int64 {
	return &i
}
