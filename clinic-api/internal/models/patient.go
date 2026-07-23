package models

import "time"

type Patient struct {
	ID         string     `json:"id"`
	AuthUserID *string    `json:"auth_user_id,omitempty"`
	Name       string     `json:"name"`
	Email      string     `json:"email"`
	Phone      *string    `json:"phone,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}