package repositories

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/savanainformatics/clinic-api/internal/models"
)

type PatientRepo struct {
	pool *pgxpool.Pool
}

func NewPatientRepo(pool *pgxpool.Pool) *PatientRepo {
	return &PatientRepo{pool: pool}
}

func (r *PatientRepo) Create(ctx context.Context, patient *models.Patient) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO patients (name, email, phone)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at`,
		patient.Name, patient.Email, patient.Phone,
	).Scan(&patient.ID, &patient.CreatedAt)
}

func (r *PatientRepo) GetByEmail(ctx context.Context, email string) (*models.Patient, error) {
	var p models.Patient
	err := r.pool.QueryRow(ctx,
		`SELECT id, auth_user_id::text, name, email, phone, created_at
		 FROM patients WHERE email = $1`, email,
	).Scan(&p.ID, &p.AuthUserID, &p.Name, &p.Email, &p.Phone, &p.CreatedAt)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *PatientRepo) GetByID(ctx context.Context, id string) (*models.Patient, error) {
	var p models.Patient
	err := r.pool.QueryRow(ctx,
		`SELECT id, auth_user_id::text, name, email, phone, created_at
		 FROM patients WHERE id = $1`, id,
	).Scan(&p.ID, &p.AuthUserID, &p.Name, &p.Email, &p.Phone, &p.CreatedAt)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *PatientRepo) GetAll(ctx context.Context, search string) ([]models.Patient, error) {
	query := `SELECT id, auth_user_id::text, name, email, phone, created_at FROM patients`
	var args []interface{}

	if search != "" {
		query += ` WHERE name ILIKE $1 OR email ILIKE $1`
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern)
	}

	query += ` ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patients []models.Patient
	for rows.Next() {
		var p models.Patient
		if err := rows.Scan(&p.ID, &p.AuthUserID, &p.Name, &p.Email, &p.Phone, &p.CreatedAt); err != nil {
			return nil, err
		}
		patients = append(patients, p)
	}
	if patients == nil {
		patients = []models.Patient{}
	}
	return patients, rows.Err()
}

// IsDuplicateEmailError checks if an error is a PostgreSQL unique constraint violation on email
func IsDuplicateEmailError(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
