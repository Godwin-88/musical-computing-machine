package services

import (
	"context"
	"time"

	"github.com/savanainformatics/clinic-api/internal/models"
	"github.com/savanainformatics/clinic-api/internal/repositories"
)

type AppointmentService struct {
	appointmentRepo *repositories.AppointmentRepo
}

func NewAppointmentService(appointmentRepo *repositories.AppointmentRepo) *AppointmentService {
	return &AppointmentService{appointmentRepo: appointmentRepo}
}

func (s *AppointmentService) BookAppointment(ctx context.Context, doctorID, patientID string, startTime time.Time, slotDuration time.Duration) (*models.Appointment, error) {
	appointment := &models.Appointment{
		DoctorID:  doctorID,
		PatientID: patientID,
		StartTime: startTime,
		EndTime:   startTime.Add(slotDuration),
		Status:    "BOOKED",
	}

	err := s.appointmentRepo.Create(ctx, appointment)
	if err != nil {
		return nil, err
	}

	return appointment, nil
}

func (s *AppointmentService) CancelAppointment(ctx context.Context, id, reason string) (*models.Appointment, error) {
	return s.appointmentRepo.Cancel(ctx, id, reason)
}

func (s *AppointmentService) RescheduleAppointment(ctx context.Context, id string, newStartTime time.Time) (*models.Appointment, error) {
	return s.appointmentRepo.Reschedule(ctx, id, newStartTime)
}
