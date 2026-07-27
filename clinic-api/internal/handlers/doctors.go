package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/savanainformatics/clinic-api/internal/errors"
	"github.com/savanainformatics/clinic-api/internal/models"
	"github.com/savanainformatics/clinic-api/internal/repositories"
	"github.com/savanainformatics/clinic-api/internal/services"
)

type DoctorHandler struct {
	doctorRepo  *repositories.DoctorRepo
	appointRepo *repositories.AppointmentRepo
	pool        *pgxpool.Pool
}

func NewDoctorHandler(doctorRepo *repositories.DoctorRepo, appointRepo *repositories.AppointmentRepo, pool *pgxpool.Pool) *DoctorHandler {
	return &DoctorHandler{
		doctorRepo:  doctorRepo,
		appointRepo: appointRepo,
		pool:        pool,
	}
}

type createDoctorRequest struct {
	Name                string `json:"name"`
	Specialisation      string `json:"specialisation"`
	SlotDurationMinutes *int   `json:"slot_duration_minutes,omitempty"`
}

func (h *DoctorHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createDoctorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		status, errResp := errors.Validation("Invalid request body")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	if req.Name == "" {
		status, errResp := errors.Validation("Doctor name is required.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	if req.Specialisation == "" {
		status, errResp := errors.Validation("Doctor specialisation is required.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	slotDuration := 30
	if req.SlotDurationMinutes != nil {
		if *req.SlotDurationMinutes != 30 {
			status, errResp := errors.Validation("Only 30-minute slots are supported in this version.")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(errResp)
			return
		}
		slotDuration = *req.SlotDurationMinutes
	}

	doctor := &models.Doctor{
		Name:                req.Name,
		Specialisation:      req.Specialisation,
		SlotDurationMinutes: slotDuration,
	}

	if err := h.doctorRepo.Create(r.Context(), doctor); err != nil {
		log.Printf("ERROR creating doctor: %v", err)
		status, errResp := errors.Internal()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(doctor)
}

func (h *DoctorHandler) List(w http.ResponseWriter, r *http.Request) {
	doctors, err := h.doctorRepo.GetAll(r.Context())
	if err != nil {
		log.Printf("ERROR listing doctors: %v", err)
		status, errResp := errors.Internal()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	// Add working hours to each doctor
	type doctorResponse struct {
		ID                  string                `json:"id"`
		Name                string                `json:"name"`
		Specialisation      string                `json:"specialisation"`
		SlotDurationMinutes int                   `json:"slot_duration_minutes"`
		WorkingHours        []models.WorkingHours `json:"working_hours,omitempty"`
	}

	var response []doctorResponse
	for _, d := range doctors {
		wh, err := h.doctorRepo.GetWorkingHours(r.Context(), d.ID)
		whList := []models.WorkingHours{}
		if err == nil {
			whList = wh
		}
		response = append(response, doctorResponse{
			ID:                  d.ID,
			Name:                d.Name,
			Specialisation:      d.Specialisation,
			SlotDurationMinutes: d.SlotDurationMinutes,
			WorkingHours:        whList,
		})
	}

	if response == nil {
		response = []doctorResponse{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

func (h *DoctorHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	doctor, err := h.doctorRepo.GetByID(r.Context(), id)
	if err != nil {
		log.Printf("ERROR getting doctor %s: %v", id, err)
		status, errResp := errors.Internal()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}
	if doctor == nil {
		status, errResp := errors.NotFound("Doctor with id '" + id + "' not found.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	wh, err := h.doctorRepo.GetWorkingHours(r.Context(), doctor.ID)
	if err != nil {
		log.Printf("ERROR getting working hours for doctor %s: %v", doctor.ID, err)
		status, errResp := errors.Internal()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}
	if wh == nil {
		wh = []models.WorkingHours{}
	}

	type doctorResponse struct {
		ID                  string                `json:"id"`
		Name                string                `json:"name"`
		Specialisation      string                `json:"specialisation"`
		SlotDurationMinutes int                   `json:"slot_duration_minutes"`
		WorkingHours        []models.WorkingHours `json:"working_hours"`
		CreatedAt           time.Time             `json:"created_at"`
		UpdatedAt           time.Time             `json:"updated_at"`
	}

	resp := doctorResponse{
		ID:                  doctor.ID,
		Name:                doctor.Name,
		Specialisation:      doctor.Specialisation,
		SlotDurationMinutes: doctor.SlotDurationMinutes,
		WorkingHours:        wh,
		CreatedAt:           doctor.CreatedAt,
		UpdatedAt:           doctor.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

type updateDoctorRequest struct {
	Name           string `json:"name"`
	Specialisation string `json:"specialisation"`
}

func (h *DoctorHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateDoctorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		status, errResp := errors.Validation("Invalid request body")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	doctor := &models.Doctor{
		Name:           req.Name,
		Specialisation: req.Specialisation,
	}

	if err := h.doctorRepo.Update(r.Context(), id, doctor); err != nil {
		if err.Error() == "no rows in result set" {
			status, errResp := errors.NotFound("Doctor with id '" + id + "' not found.")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(errResp)
			return
		}
		log.Printf("ERROR updating doctor %s: %v", id, err)
		status, errResp := errors.Internal()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(doctor)
}

type setWorkingHoursRequest struct {
	WorkingHours []models.WorkingHoursInput `json:"working_hours"`
}

func (h *DoctorHandler) SetWorkingHours(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Verify doctor exists
	doctor, err := h.doctorRepo.GetByID(r.Context(), id)
	if err != nil {
		log.Printf("ERROR getting doctor %s for setWorkingHours: %v", id, err)
		status, errResp := errors.Internal()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}
	if doctor == nil {
		status, errResp := errors.NotFound("Doctor with id '" + id + "' not found.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	var req setWorkingHoursRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		status, errResp := errors.Validation("Invalid request body")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	seenDays := make(map[int]bool)
	for _, wh := range req.WorkingHours {
		if wh.DayOfWeek < 0 || wh.DayOfWeek > 6 {
			status, errResp := errors.Validation("day_of_week must be between 0 (Monday) and 6 (Sunday).")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(errResp)
			return
		}
		if seenDays[wh.DayOfWeek] {
			status, errResp := errors.Validation("Duplicate day_of_week entries are not allowed.")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(errResp)
			return
		}
		seenDays[wh.DayOfWeek] = true

		// Validate time format and 30-minute boundaries
		startParsed, err := time.Parse("15:04", wh.StartTime)
		if err != nil {
			startParsed, err = time.Parse("15:04:05", wh.StartTime)
			if err != nil {
				status, errResp := errors.Validation("Invalid time format. Use HH:MM.")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				_ = json.NewEncoder(w).Encode(errResp)
				return
			}
		}

		endParsed, err := time.Parse("15:04", wh.EndTime)
		if err != nil {
			endParsed, err = time.Parse("15:04:05", wh.EndTime)
			if err != nil {
				status, errResp := errors.Validation("Invalid time format. Use HH:MM.")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				_ = json.NewEncoder(w).Encode(errResp)
				return
			}
		}

		// Check 30-minute boundaries
		if startParsed.Minute()%30 != 0 || endParsed.Minute()%30 != 0 {
			status, errResp := errors.Validation("Working hours must start and end on 30-minute boundaries.")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(errResp)
			return
		}

		if !endParsed.After(startParsed) {
			status, errResp := errors.Validation("start_time must be before end_time.")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(errResp)
			return
		}
	}

	if err := h.doctorRepo.SetWorkingHours(r.Context(), id, req.WorkingHours); err != nil {
		log.Printf("ERROR setting working hours for doctor %s: %v", id, err)
		status, errResp := errors.Internal()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	// Return updated working hours
	wh, err := h.doctorRepo.GetWorkingHours(r.Context(), id)
	if err != nil {
		log.Printf("ERROR getting working hours after set for doctor %s: %v", id, err)
		status, errResp := errors.Internal()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}
	if wh == nil {
		wh = []models.WorkingHours{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(wh)
}

func (h *DoctorHandler) GetAvailability(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	dateStr := r.URL.Query().Get("date")

	if dateStr == "" {
		status, errResp := errors.Validation("Query parameter 'date' is required in YYYY-MM-DD format.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		status, errResp := errors.Validation("Query parameter 'date' is required in YYYY-MM-DD format.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	if date.Before(today) {
		status, errResp := errors.Validation("Cannot query availability for a past date.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	// Verify doctor exists
	doctor, err := h.doctorRepo.GetByID(r.Context(), id)
	if err != nil {
		log.Printf("ERROR getting doctor %s for availability: %v", id, err)
		status, errResp := errors.Internal()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}
	if doctor == nil {
		status, errResp := errors.NotFound("Doctor with id '" + id + "' not found.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	// Get working hours
	wh, err := h.doctorRepo.GetWorkingHours(r.Context(), id)
	if err != nil {
		log.Printf("ERROR getting working hours for doctor %s availability: %v", id, err)
		status, errResp := errors.Internal()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	// Get booked slots
	bookedSlots, err := h.appointRepo.GetBookedSlotsForDate(r.Context(), id, date)
	if err != nil {
		log.Printf("ERROR getting booked slots for doctor %s on %s: %v", id, dateStr, err)
		status, errResp := errors.Internal()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	slotDuration := time.Duration(doctor.SlotDurationMinutes) * time.Minute
	bufferDuration := 1 * time.Hour

	slots := services.GenerateSlots(wh, bookedSlots, date, now, slotDuration, bufferDuration)

	response := struct {
		DoctorID       string        `json:"doctor_id"`
		Date           string        `json:"date"`
		AvailableSlots []models.Slot `json:"available_slots"`
	}{
		DoctorID:       id,
		Date:           dateStr,
		AvailableSlots: slots,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}
