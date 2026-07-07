package model

// RecordError is the structured error shape stored in CAL records.
type RecordError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
