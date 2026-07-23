package repositories

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/savanainformatics/clinic-api/internal/models"
)

type DoctorRepo struct {
	pool *pgxpool.Pool
}

func NewDoctorRepo(pool *pgxpool.Pool) *DoctorRepo {
	return &DoctorRepo{pool: pool}
}

func (r *DoctorRepo) Create(ctx context.Context, doctor *models.Doctor) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO doctors (name, specialisation, slot_duration_minutes)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at, updated_at`,
		doctor.Name, doctor.Specialisation, doctor.SlotDurationMinutes,
	).Scan(&doctor.ID, &doctor.CreatedAt, &doctor.UpdatedAt)
}

func (r *DoctorRepo) GetAll(ctx context.Context) ([]models.Doctor, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, specialisation, slot_duration_minutes, created_at, updated_at
		 FROM doctors ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var doctors []models.Doctor
	for rows.Next() {
		var d models.Doctor
		if err := rows.Scan(&d.ID, &d.Name, &d.Specialisation, &d.SlotDurationMinutes, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		doctors = append(doctors, d)
	}
	if doctors == nil {
		doctors = []models.Doctor{}
	}
	return doctors, rows.Err()
}

func (r *DoctorRepo) GetByID(ctx context.Context, id string) (*models.Doctor, error) {
	var d models.Doctor
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, specialisation, slot_duration_minutes, created_at, updated_at
		 FROM doctors WHERE id = $1`, id,
	).Scan(&d.ID, &d.Name, &d.Specialisation, &d.SlotDurationMinutes, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

func (r *DoctorRepo) Update(ctx context.Context, id string, doctor *models.Doctor) error {
	return r.pool.QueryRow(ctx,
		`UPDATE doctors SET name = $1, specialisation = $2, updated_at = now()
		 WHERE id = $3
		 RETURNING id, name, specialisation, slot_duration_minutes, created_at, updated_at`,
		doctor.Name, doctor.Specialisation, id,
	).Scan(&doctor.ID, &doctor.Name, &doctor.Specialisation, &doctor.SlotDurationMinutes, &doctor.CreatedAt, &doctor.UpdatedAt)
}

func (r *DoctorRepo) GetWorkingHours(ctx context.Context, doctorID string) ([]models.WorkingHours, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, doctor_id, day_of_week, start_time, end_time
		 FROM working_hours WHERE doctor_id = $1
		 ORDER BY day_of_week`, doctorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hours []models.WorkingHours
	for rows.Next() {
		var wh models.WorkingHours
		if err := rows.Scan(&wh.ID, &wh.DoctorID, &wh.DayOfWeek, &wh.StartTime, &wh.EndTime); err != nil {
			return nil, err
		}
		hours = append(hours, wh)
	}
	if hours == nil {
		hours = []models.WorkingHours{}
	}
	return hours, rows.Err()
}

func (r *DoctorRepo) SetWorkingHours(ctx context.Context, doctorID string, hours []models.WorkingHoursInput) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Delete existing working hours for the doctor
	_, err = tx.Exec(ctx, `DELETE FROM working_hours WHERE doctor_id = $1`, doctorID)
	if err != nil {
		return err
	}

	// Insert new working hours
	for _, h := range hours {
		_, err = tx.Exec(ctx,
			`INSERT INTO working_hours (doctor_id, day_of_week, start_time, end_time)
			 VALUES ($1, $2, $3::time, $4::time)`,
			doctorID, h.DayOfWeek, h.StartTime, h.EndTime)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
