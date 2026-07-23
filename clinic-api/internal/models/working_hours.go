package models

import "time"

type WorkingHours struct {
	ID        string    `json:"id"`
	DoctorID  string    `json:"doctor_id"`
	DayOfWeek int       `json:"day_of_week"`
	StartTime string    `json:"start_time"` // HH:MM:SS format
	EndTime   string    `json:"end_time"`   // HH:MM:SS format
}

type WorkingHoursInput struct {
	DayOfWeek int    `json:"day_of_week"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

// Slot represents an available time slot
type Slot struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}