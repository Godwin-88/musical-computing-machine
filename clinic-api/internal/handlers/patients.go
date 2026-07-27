package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/savanainformatics/clinic-api/internal/errors"
	"github.com/savanainformatics/clinic-api/internal/models"
	"github.com/savanainformatics/clinic-api/internal/repositories"
)

type PatientHandler struct {
	patientRepo *repositories.PatientRepo
	pool        *pgxpool.Pool
}

func NewPatientHandler(patientRepo *repositories.PatientRepo, pool *pgxpool.Pool) *PatientHandler {
	return &PatientHandler{
		patientRepo: patientRepo,
		pool:        pool,
	}
}

type createPatientRequest struct {
	Name  string  `json:"name"`
	Email string  `json:"email"`
	Phone *string `json:"phone,omitempty"`
}

func (h *PatientHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createPatientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		status, errResp := errors.Validation("Invalid request body")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	if req.Name == "" {
		status, errResp := errors.Validation("Patient name is required.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	if req.Email == "" {
		status, errResp := errors.Validation("Patient email is required.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	patient := &models.Patient{
		Name:  req.Name,
		Email: req.Email,
		Phone: req.Phone,
	}

	if err := h.patientRepo.Create(r.Context(), patient); err != nil {
		if repositories.IsDuplicateEmailError(err) {
			status, errResp := errors.Conflict("A patient with this email already exists.")
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
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(patient)
}

func (h *PatientHandler) List(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")

	patients, err := h.patientRepo.GetAll(r.Context(), search)
	if err != nil {
		status, errResp := errors.Internal()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(patients)
}

func (h *PatientHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	patient, err := h.patientRepo.GetByID(r.Context(), id)
	if err != nil {
		status, errResp := errors.Internal()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}
	if patient == nil {
		status, errResp := errors.NotFound("Patient with id '" + id + "' not found.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(patient)
}

// GetAppointments returns all appointments for a patient
func (h *PatientHandler) GetAppointments(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	includePast := r.URL.Query().Get("include_past") == "true"

	// Verify patient exists
	patient, err := h.patientRepo.GetByID(r.Context(), id)
	if err != nil {
		status, errResp := errors.Internal()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}
	if patient == nil {
		status, errResp := errors.NotFound("Patient with id '" + id + "' not found.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}

	// Query appointment with patient ID
	query := `SELECT a.id, a.doctor_id, d.name, a.patient_id, p.name,
	                 a.start_time, a.end_time, a.status,
	                 a.cancellation_reason, a.cancelled_at,
	                 a.created_at, a.updated_at
	          FROM appointments a
	          JOIN doctors d ON d.id = a.doctor_id
	          JOIN patients p ON p.id = a.patient_id
	          WHERE a.patient_id = $1`
	var args []interface{}
	args = append(args, id)

	if !includePast {
		query += ` AND a.status = 'BOOKED' AND a.start_time > NOW()`
	}

	query += ` ORDER BY a.start_time ASC`

	rows, err := h.pool.Query(r.Context(), query, args...)
	if err != nil {
		status, errResp := errors.Internal()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errResp)
		return
	}
	defer rows.Close()

	var appointments []models.Appointment
	for rows.Next() {
		var a models.Appointment
		if err := rows.Scan(&a.ID, &a.DoctorID, &a.DoctorName, &a.PatientID, &a.PatientName,
			&a.StartTime, &a.EndTime, &a.Status,
			&a.CancellationReason, &a.CancelledAt,
			&a.CreatedAt, &a.UpdatedAt); err != nil {
			status, errResp := errors.Internal()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(errResp)
			return
		}
		appointments = append(appointments, a)
	}
	if appointments == nil {
		appointments = []models.Appointment{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(appointments)
}
