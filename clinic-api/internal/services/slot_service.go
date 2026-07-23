package services

import (
	"time"

	"github.com/savanainformatics/clinic-api/internal/models"
)

func GenerateSlots(
	workingHours []models.WorkingHours,
	bookedSlots []time.Time,
	date time.Time,
	now time.Time,
	slotDuration time.Duration,
	bufferDuration time.Duration,
) []models.Slot {
	// Build a set of booked slots for O(1) lookup
	bookedSet := make(map[time.Time]bool)
	for _, bs := range bookedSlots {
		// Normalize to the date being queried
		normalized := time.Date(date.Year(), date.Month(), date.Day(),
			bs.Hour(), bs.Minute(), 0, 0, time.UTC)
		bookedSet[normalized] = true
	}

	loc := time.UTC

	// Find matching working hours for this weekday
	weekday := int(date.Weekday())
	// Go's Weekday: Sunday=0, Monday=1, ... Saturday=6
	// Our schema: Monday=0, Tuesday=1, ... Sunday=6
	dayOfWeek := (weekday + 6) % 7 // Convert Go weekday to schema day_of_week

	var wh *models.WorkingHours
	for _, h := range workingHours {
		if h.DayOfWeek == dayOfWeek {
			wh = &h
			break
		}
	}

	if wh == nil {
		return []models.Slot{}
	}

	startParts, _ := parseTime(wh.StartTime)
	endParts, _ := parseTime(wh.EndTime)

	startHour, startMin := startParts[0], startParts[1]
	endHour, endMin := endParts[0], endParts[1]

	slotStart := time.Date(date.Year(), date.Month(), date.Day(),
		startHour, startMin, 0, 0, loc)
	slotEnd := time.Date(date.Year(), date.Month(), date.Day(),
		endHour, endMin, 0, 0, loc)

	var slots []models.Slot
	for slotStart.Before(slotEnd) {
		slotEndTime := slotStart.Add(slotDuration)

		if slotEndTime.After(slotEnd) {
			break
		}

		// Exclude slots in the past
		if slotStart.Before(now) || slotStart.Equal(now) {
			slotStart = slotStart.Add(slotDuration)
			continue
		}

		// Exclude slots within buffer duration of now
		if slotStart.Before(now.Add(bufferDuration)) {
			slotStart = slotStart.Add(slotDuration)
			continue
		}

		// Exclude booked slots
		if bookedSet[slotStart] {
			slotStart = slotStart.Add(slotDuration)
			continue
		}

		slots = append(slots, models.Slot{
			Start: slotStart,
			End:   slotEndTime,
		})

		slotStart = slotStart.Add(slotDuration)
	}

	if slots == nil {
		slots = []models.Slot{}
	}

	return slots
}

// parseTime parses a time string in HH:MM:SS or HH:MM format and returns hour, minute
func parseTime(t string) ([2]int, error) {
	parsed, err := time.Parse("15:04:05", t)
	if err != nil {
		parsed, err = time.Parse("15:04", t)
		if err != nil {
			return [2]int{0, 0}, err
		}
		return [2]int{parsed.Hour(), parsed.Minute()}, nil
	}
	return [2]int{parsed.Hour(), parsed.Minute()}, nil
}
