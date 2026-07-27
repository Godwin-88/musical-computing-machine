package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/savanainformatics/clinic-api/internal/errors"
	"github.com/savanainformatics/clinic-api/internal/repositories"
	"github.com/savanainformatics/clinic-api/internal/services"
)

type AppointmentHandler struct {
	appointmentRepo    *repositories.AppointmentRepo
	doctorRepo         *repositories.DoctorRepo
	patientRepo        *repositories.PatientRepo
	appointmentService *services.AppointmentService
	pool               *pgxpool.Pool
}

func NewAppointmentHandler(
	appointmentRepo *repositories.AppointmentRepo,
	doctorRepo *repositories.DoctorRepo,
	patientRepo *repositories.PatientRepo,
	appointmentService *services.AppointmentService,
	pool *pgxpool.Pool,
) *AppointmentHandler {
	return &AppointmentHandler{
		appointmentRepo:    appointmentRepo,
		doctorRepo:         doctorRepo,
		patientRepo:        patientRepo,
		appointmentService: appointmentService,
		pool:               pool,
	}
}

type bookAppointmentRequest struct {
	DoctorID  string `json:"doctor_id"`
	PatientID string `json:"patient_id"`
	StartTime string `json:"start_time"`
}

func (h *AppointmentHandler) Book(w http.ResponseWriter, r *http.Request) {
	var req bookAppointmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		status, errResp := errors.Validation("Invalid request body")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	if req.DoctorID == "" {
		status, errResp := errors.Validation("doctor_id is required.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	if req.PatientID == "" {
		status, errResp := errors.Validation("patient_id is required.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	if req.StartTime == "" {
		status, errResp := errors.Validation("start_time is required.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		// Try alternative format
		startTime, err = time.Parse("2006-01-02T15:04:05Z07:00", req.StartTime)
		if err != nil {
			status, errResp := errors.Validation("Invalid start_time format. Use ISO 8601 (e.g. 2026-08-01T09:00:00Z).")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(errResp)
			return
		}
	}

	startTime = startTime.UTC()

	// Verify doctor exists
	doctor, err := h.doctorRepo.GetByID(r.Context(), req.DoctorID)
	if err != nil {
		status, errResp := errors.Internal()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}
	if doctor == nil {
		status, errResp := errors.NotFound("Doctor with id '" + req.DoctorID + "' not found.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	// Verify patient exists
	patient, err := h.patientRepo.GetByID(r.Context(), req.PatientID)
	if err != nil {
		status, errResp := errors.Internal()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}
	if patient == nil {
		status, errResp := errors.NotFound("Patient with id '" + req.PatientID + "' not found.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	// Validate slot is on 30-minute boundary
	if startTime.Minute()%30 != 0 || startTime.Second() != 0 {
		status, errResp := errors.Validation("Slot start time must align to 30-minute boundaries.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	now := time.Now().UTC()

	// Validate slot is not in the past
	if startTime.Before(now) || startTime.Equal(now) {
		status, errResp := errors.Validation("Cannot book an appointment in the past.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	// Validate slot is at least 1 hour from now
	if startTime.Before(now.Add(1 * time.Hour)) {
		status, errResp := errors.Validation("Appointments must be booked at least 1 hour in advance.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	// Validate slot falls within working hours
	wh, err := h.doctorRepo.GetWorkingHours(r.Context(), req.DoctorID)
	if err != nil {
		status, errResp := errors.Internal()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	// Get booked slots for the date
	bookedSlots, err := h.appointmentRepo.GetBookedSlotsForDate(r.Context(), req.DoctorID, startTime)
	if err != nil {
		status, errResp := errors.Internal()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	slotDuration := time.Duration(doctor.SlotDurationMinutes) * time.Minute
	bufferDuration := 1 * time.Hour

	slots := services.GenerateSlots(wh, bookedSlots, startTime, now, slotDuration, bufferDuration)

	// Check if requested slot is available
	slotFound := false
	for _, slot := range slots {
		if slot.Start.Equal(startTime) {
			slotFound = true
			break
		}
	}

	if !slotFound {
		status, errResp := errors.Validation("The requested slot is outside the doctor's working hours.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	// Book the appointment
	appointment, err := h.appointmentService.BookAppointment(r.Context(), req.DoctorID, req.PatientID, startTime, slotDuration)
	if err != nil {
		if err == repositories.ErrSlotConflict {
			status, errResp := errors.Conflict("The requested slot is already booked. Please choose another time.")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(errResp)
			return
		}
		status, errResp := errors.Internal()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	w.Header().Set("Location", "/appointments/"+appointment.ID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(appointment)
}

type cancelAppointmentRequest struct {
	Reason string `json:"reason"`
}

func (h *AppointmentHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req cancelAppointmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		status, errResp := errors.Validation("Invalid request body")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	if req.Reason == "" {
		status, errResp := errors.Validation("Cancellation reason is required.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	if len(req.Reason) > 500 {
		status, errResp := errors.Validation("Cancellation reason must not exceed 500 characters.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	appointment, err := h.appointmentService.CancelAppointment(r.Context(), id, req.Reason)
	if err != nil {
		if err == repositories.ErrNotFound {
			status, errResp := errors.NotFound("Appointment with id '" + id + "' not found.")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(errResp)
			return
		}
		if err == repositories.ErrAlreadyCancelled {
			status, errResp := errors.Conflict("Appointment '" + id + "' is already cancelled.")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(errResp)
			return
		}
		status, errResp := errors.Internal()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(appointment)
}

type rescheduleAppointmentRequest struct {
	NewStartTime string `json:"new_start_time"`
}

func (h *AppointmentHandler) Reschedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req rescheduleAppointmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		status, errResp := errors.Validation("Invalid request body")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	if req.NewStartTime == "" {
		status, errResp := errors.Validation("new_start_time is required.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	newStartTime, err := time.Parse(time.RFC3339, req.NewStartTime)
	if err != nil {
		newStartTime, err = time.Parse("2006-01-02T15:04:05Z07:00", req.NewStartTime)
		if err != nil {
			status, errResp := errors.Validation("Invalid new_start_time format. Use ISO 8601.")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(errResp)
			return
		}
	}

	newStartTime = newStartTime.UTC()

	// Get the original appointment to validate
	existingAppointment, err := h.appointmentRepo.GetByID(r.Context(), id)
	if err != nil {
		status, errResp := errors.Internal()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}
	if existingAppointment == nil {
		status, errResp := errors.NotFound("Appointment with id '" + id + "' not found.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	if existingAppointment.Status == "CANCELLED" {
		status, errResp := errors.Conflict("Cannot reschedule a cancelled appointment.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	// Check that new start time differs from current
	if newStartTime.Equal(existingAppointment.StartTime) {
		status, errResp := errors.Validation("New start time must differ from the current appointment time.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	// Validate the new slot
	// 30-minute boundary
	if newStartTime.Minute()%30 != 0 || newStartTime.Second() != 0 {
		status, errResp := errors.Validation("Slot start time must align to 30-minute boundaries.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	now := time.Now().UTC()
	if newStartTime.Before(now) || newStartTime.Equal(now) {
		status, errResp := errors.Validation("Cannot book an appointment in the past.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	if newStartTime.Before(now.Add(1 * time.Hour)) {
		status, errResp := errors.Validation("Appointments must be booked at least 1 hour in advance.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	appointment, err := h.appointmentService.RescheduleAppointment(r.Context(), id, newStartTime)
	if err != nil {
		if err == repositories.ErrNotFound {
			status, errResp := errors.NotFound("Appointment with id '" + id + "' not found.")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(errResp)
			return
		}
		if err == repositories.ErrSlotConflict {
			status, errResp := errors.Conflict("The requested slot is already booked. Please choose another time.")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(errResp)
			return
		}
		if err == repositories.ErrAlreadyCancelled {
			status, errResp := errors.Conflict("Cannot reschedule a cancelled appointment.")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(errResp)
			return
		}
		status, errResp := errors.Internal()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(appointment)
}

func (h *AppointmentHandler) List(w http.ResponseWriter, r *http.Request) {
	doctorID := r.URL.Query().Get("doctor_id")
	status := r.URL.Query().Get("status")

	limit := 25
	offset := 0

	appointments, err := h.appointmentRepo.GetAll(r.Context(), doctorID, status, limit, offset)
	if err != nil {
		status, errResp := errors.Internal()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(appointments)
}
