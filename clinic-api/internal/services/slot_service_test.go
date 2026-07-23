package services

import (
	"testing"
	"time"

	"github.com/savanainformatics/clinic-api/internal/models"
)

func TestGenerateSlots(t *testing.T) {
	// Fixed reference time: Monday, 2026-08-03 08:00 UTC
	// Go Weekday: Monday=1, our schema day_of_week: Monday=0
	now := time.Date(2026, 8, 3, 8, 0, 0, 0, time.UTC)
	date := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC) // Monday
	slotDuration := 30 * time.Minute
	bufferDuration := 1 * time.Hour

	tests := []struct {
		name         string
		workingHours []models.WorkingHours
		bookedSlots  []time.Time
		date         time.Time
		now          time.Time
		expected     int
	}{
		{
			name: "full working day, no bookings",
			workingHours: []models.WorkingHours{
				{DayOfWeek: 0, StartTime: "09:00:00", EndTime: "17:00:00"},
			},
			bookedSlots: []time.Time{},
			date:        date,
			now:         now,
			expected:    16, // 8 hours = 16 slots
		},
		{
			name: "some slots already booked",
			workingHours: []models.WorkingHours{
				{DayOfWeek: 0, StartTime: "09:00:00", EndTime: "12:00:00"},
			},
			bookedSlots: []time.Time{
				time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC),
				time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC),
			},
			date:     date,
			now:      now,
			expected: 4, // 6 slots total (9:00-12:00), 2 booked = 4
		},
		{
			name: "fully booked day",
			workingHours: []models.WorkingHours{
				{DayOfWeek: 0, StartTime: "09:00:00", EndTime: "10:00:00"},
			},
			bookedSlots: []time.Time{
				time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC),
				time.Date(2026, 8, 3, 9, 30, 0, 0, time.UTC),
			},
			date:     date,
			now:      now,
			expected: 0, // 2 slots, both booked
		},
		{
			name: "day with no working hours",
			workingHours: []models.WorkingHours{
				{DayOfWeek: 1, StartTime: "09:00:00", EndTime: "17:00:00"}, // Tuesday, not Monday
			},
			bookedSlots: []time.Time{},
			date:        date,
			now:         now,
			expected:    0,
		},
		{
			name: "slots in the past excluded",
			workingHours: []models.WorkingHours{
				{DayOfWeek: 0, StartTime: "07:00:00", EndTime: "10:00:00"},
			},
			bookedSlots: []time.Time{},
			date:        date,
			now:         time.Date(2026, 8, 3, 9, 30, 0, 0, time.UTC),
			expected:    0, // 7:00, 7:30, 8:00, 8:30, 9:00 are all past or within buffer
		},
		{
			name: "slots within 1 hour of now excluded",
			workingHours: []models.WorkingHours{
				{DayOfWeek: 0, StartTime: "08:30:00", EndTime: "10:00:00"},
			},
			bookedSlots: []time.Time{},
			date:        date,
			now:         time.Date(2026, 8, 3, 8, 0, 0, 0, time.UTC),
			expected:    2, // 8:30 within 1h (excluded), 9:00 at boundary (included), 9:30 after (included)
		},
		{
			name: "slots exactly on the 1-hour boundary included",
			workingHours: []models.WorkingHours{
				{DayOfWeek: 0, StartTime: "09:00:00", EndTime: "10:00:00"},
			},
			bookedSlots: []time.Time{},
			date:        date,
			now:         time.Date(2026, 8, 3, 8, 0, 0, 0, time.UTC),
			expected:    2, // 9:00 is exactly 1 hour from 8:00 (boundary, included), 9:30 also available
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slots := GenerateSlots(tt.workingHours, tt.bookedSlots, tt.date, tt.now, slotDuration, bufferDuration)
			if len(slots) != tt.expected {
				t.Errorf("expected %d slots, got %d", tt.expected, len(slots))
				for _, s := range slots {
					t.Logf("  slot: %s", s.Start.Format(time.RFC3339))
				}
			}
		})
	}
}
