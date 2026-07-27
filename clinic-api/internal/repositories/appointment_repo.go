package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/savanainformatics/clinic-api/internal/models"
)

var ErrSlotConflict = errors.New("slot already booked")
var ErrNotFound = errors.New("not found")
var ErrAlreadyCancelled = errors.New("already cancelled")

type AppointmentRepo struct {
	pool *pgxpool.Pool
}

func NewAppointmentRepo(pool *pgxpool.Pool) *AppointmentRepo {
	return &AppointmentRepo{pool: pool}
}

func (r *AppointmentRepo) Create(ctx context.Context, a *models.Appointment) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO appointments (doctor_id, patient_id, start_time, end_time, status)
		 VALUES ($1, $2, $3, $4, $5)`,
		a.DoctorID, a.PatientID, a.StartTime, a.EndTime, a.Status)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrSlotConflict
		}
		return err
	}

	// Fetch the created appointment with joined names
	return r.pool.QueryRow(ctx,
		`SELECT a.id, a.doctor_id, d.name, a.patient_id, p.name,
		        a.start_time, a.end_time, a.status,
		        a.cancellation_reason, a.cancelled_at,
		        a.created_at, a.updated_at
		 FROM appointments a
		 JOIN doctors d ON d.id = a.doctor_id
		 JOIN patients p ON p.id = a.patient_id
		 WHERE a.doctor_id = $1 AND a.patient_id = $2 AND a.start_time = $3`,
		a.DoctorID, a.PatientID, a.StartTime,
	).Scan(&a.ID, &a.DoctorID, &a.DoctorName, &a.PatientID, &a.PatientName,
		&a.StartTime, &a.EndTime, &a.Status,
		&a.CancellationReason, &a.CancelledAt,
		&a.CreatedAt, &a.UpdatedAt)
}

func (r *AppointmentRepo) GetByID(ctx context.Context, id string) (*models.Appointment, error) {
	var a models.Appointment
	err := r.pool.QueryRow(ctx,
		`SELECT a.id, a.doctor_id, d.name, a.patient_id, p.name,
		        a.start_time, a.end_time, a.status,
		        a.cancellation_reason, a.cancelled_at,
		        a.created_at, a.updated_at
		 FROM appointments a
		 JOIN doctors d ON d.id = a.doctor_id
		 JOIN patients p ON p.id = a.patient_id
		 WHERE a.id = $1`, id,
	).Scan(&a.ID, &a.DoctorID, &a.DoctorName, &a.PatientID, &a.PatientName,
		&a.StartTime, &a.EndTime, &a.Status,
		&a.CancellationReason, &a.CancelledAt,
		&a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *AppointmentRepo) GetBookedSlotsForDate(ctx context.Context, doctorID string, date time.Time) ([]time.Time, error) {
	startOfDay := date.UTC().Truncate(24 * time.Hour)
	endOfDay := startOfDay.Add(24 * time.Hour)

	rows, err := r.pool.Query(ctx,
		`SELECT start_time FROM appointments
		 WHERE doctor_id = $1
		   AND start_time >= $2 AND start_time < $3
		   AND status = 'BOOKED'`,
		doctorID, startOfDay, endOfDay)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slots []time.Time
	for rows.Next() {
		var t time.Time
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		slots = append(slots, t)
	}
	if slots == nil {
		slots = []time.Time{}
	}
	return slots, rows.Err()
}

func (r *AppointmentRepo) Cancel(ctx context.Context, id string, reason string) (*models.Appointment, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var currentStatus string
	var a models.Appointment
	err = tx.QueryRow(ctx,
		`SELECT a.id, a.doctor_id, d.name, a.patient_id, p.name,
		        a.start_time, a.end_time, a.status,
		        a.cancellation_reason, a.cancelled_at,
		        a.created_at, a.updated_at
		 FROM appointments a
		 JOIN doctors d ON d.id = a.doctor_id
		 JOIN patients p ON p.id = a.patient_id
		 WHERE a.id = $1 FOR UPDATE`, id,
	).Scan(&a.ID, &a.DoctorID, &a.DoctorName, &a.PatientID, &a.PatientName,
		&a.StartTime, &a.EndTime, &currentStatus,
		&a.CancellationReason, &a.CancelledAt,
		&a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if currentStatus == "CANCELLED" {
		return nil, ErrAlreadyCancelled
	}

	now := time.Now().UTC()
	_, err = tx.Exec(ctx,
		`UPDATE appointments SET status = 'CANCELLED', cancellation_reason = $1, cancelled_at = $2, updated_at = $2
		 WHERE id = $3`,
		reason, now, id)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	a.Status = "CANCELLED"
	a.CancellationReason = &reason
	a.CancelledAt = &now
	return &a, nil
}

func (r *AppointmentRepo) Reschedule(ctx context.Context, id string, newStartTime time.Time) (*models.Appointment, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Lock the original appointment row
	var currentStatus string
	var originalStartTime time.Time
	var doctorID, patientID string
	var existingAppointment models.Appointment

	err = tx.QueryRow(ctx,
		`SELECT a.id, a.doctor_id, a.patient_id, a.start_time, a.end_time, a.status,
		        a.cancellation_reason, a.cancelled_at, a.created_at, a.updated_at,
		        d.name, p.name
		 FROM appointments a
		 JOIN doctors d ON d.id = a.doctor_id
		 JOIN patients p ON p.id = a.patient_id
		 WHERE a.id = $1 FOR UPDATE NOWAIT`, id,
	).Scan(&existingAppointment.ID, &doctorID, &patientID, &originalStartTime,
		&existingAppointment.EndTime, &currentStatus,
		&existingAppointment.CancellationReason, &existingAppointment.CancelledAt,
		&existingAppointment.CreatedAt, &existingAppointment.UpdatedAt,
		&existingAppointment.DoctorName, &existingAppointment.PatientName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if currentStatus == "CANCELLED" {
		return nil, ErrAlreadyCancelled
	}

	// Check new slot doesn't conflict with another booking
	var conflictCount int
	err = tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM appointments
		 WHERE doctor_id = $1 AND start_time = $2 AND status = 'BOOKED' AND id != $3`,
		doctorID, newStartTime, id).Scan(&conflictCount)
	if err != nil {
		return nil, err
	}
	if conflictCount > 0 {
		return nil, ErrSlotConflict
	}

	newEndTime := newStartTime.Add(30 * time.Minute)
	now := time.Now().UTC()

	// Update the appointment to new time
	_, err = tx.Exec(ctx,
		`UPDATE appointments SET start_time = $1, end_time = $2, status = 'BOOKED', updated_at = $3
		 WHERE id = $4`,
		newStartTime, newEndTime, now, id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrSlotConflict
		}
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	existingAppointment.ID = id
	existingAppointment.DoctorID = doctorID
	existingAppointment.PatientID = patientID
	existingAppointment.StartTime = newStartTime
	existingAppointment.EndTime = newEndTime
	existingAppointment.Status = "BOOKED"
	existingAppointment.UpdatedAt = now

	return &existingAppointment, nil
}

func (r *AppointmentRepo) GetAll(ctx context.Context, doctorID, status string, limit, offset int) ([]models.Appointment, error) {
	query := `SELECT a.id, a.doctor_id, d.name, a.patient_id, p.name,
	                 a.start_time, a.end_time, a.status,
	                 a.cancellation_reason, a.cancelled_at,
	                 a.created_at, a.updated_at
	          FROM appointments a
	          JOIN doctors d ON d.id = a.doctor_id
	          JOIN patients p ON p.id = a.patient_id
	          WHERE 1=1`
	var args []interface{}
	argIdx := 1

	if doctorID != "" {
		query += ` AND a.doctor_id = $` + string(rune('0'+argIdx))
		args = append(args, doctorID)
		argIdx++
	}
	if status != "" {
		query += ` AND a.status = $` + string(rune('0'+argIdx))
		args = append(args, status)
		argIdx++
	}

	query += ` ORDER BY a.start_time DESC LIMIT $` + string(rune('0'+argIdx)) + ` OFFSET $` + string(rune('0'+argIdx+1))
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var appointments []models.Appointment
	for rows.Next() {
		var a models.Appointment
		if err := rows.Scan(&a.ID, &a.DoctorID, &a.DoctorName, &a.PatientID, &a.PatientName,
			&a.StartTime, &a.EndTime, &a.Status,
			&a.CancellationReason, &a.CancelledAt,
			&a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		appointments = append(appointments, a)
	}
	if appointments == nil {
		appointments = []models.Appointment{}
	}
	return appointments, rows.Err()
}
