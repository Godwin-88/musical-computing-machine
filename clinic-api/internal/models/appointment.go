package models

import "time"

type Appointment struct {
	ID                string     `json:"id"`
	DoctorID          string     `json:"doctor_id"`
	PatientID         string     `json:"patient_id"`
	StartTime         time.Time  `json:"start_time"`
	EndTime           time.Time  `json:"end_time"`
	Status            string     `json:"status"`
	CancellationReason *string   `json:"cancellation_reason,omitempty"`
	CancelledAt       *time.Time `json:"cancelled_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	// Joined fields for responses
	DoctorName  string `json:"doctor_name,omitempty"`
	PatientName string `json:"patient_name,omitempty"`
}