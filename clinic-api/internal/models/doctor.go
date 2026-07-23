package models

import "time"

type Doctor struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Specialisation     string    `json:"specialisation"`
	SlotDurationMinutes int      `json:"slot_duration_minutes"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}