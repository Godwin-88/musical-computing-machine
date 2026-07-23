package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/savanainformatics/clinic-api/internal/config"
	"github.com/savanainformatics/clinic-api/internal/database"
	"github.com/savanainformatics/clinic-api/internal/errors"
	"github.com/savanainformatics/clinic-api/internal/handlers"
	"github.com/savanainformatics/clinic-api/internal/middleware"
	"github.com/savanainformatics/clinic-api/internal/repositories"
	"github.com/savanainformatics/clinic-api/internal/services"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	// Connect to database
	pool := database.Connect(ctx, cfg.DatabaseURL)
	defer pool.Close()

	// Initialize repositories
	doctorRepo := repositories.NewDoctorRepo(pool)
	appointmentRepo := repositories.NewAppointmentRepo(pool)
	patientRepo := repositories.NewPatientRepo(pool)

	// Initialize services
	appointmentService := services.NewAppointmentService(appointmentRepo)

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler(pool)
	doctorHandler := handlers.NewDoctorHandler(doctorRepo, appointmentRepo, pool)
	appointmentHandler := handlers.NewAppointmentHandler(appointmentRepo, doctorRepo, patientRepo, appointmentService, pool)
	patientHandler := handlers.NewPatientHandler(patientRepo, pool)

	// Build router
	r := chi.NewRouter()

	// Global middleware
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(chiMiddleware.Recoverer)

	// Custom panic recovery that sanitizes responses
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Printf("PANIC: %v", rec)
					status, errResp := errors.Internal()
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(status)
					json.NewEncoder(w).Encode(errResp)
				}
			}()
			next.ServeHTTP(w, r)
		})
	})

	// Public routes
	r.Get("/health", healthHandler.Check)
	r.Get("/doctors", doctorHandler.List)
	r.Get("/doctors/{id}", doctorHandler.GetByID)
	r.Get("/doctors/{id}/availability", doctorHandler.GetAvailability)

	// Protected routes (require clinic_admin role)
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(cfg.SupabaseJWTSecret))
		r.Use(middleware.RequireRole("clinic_admin"))

		// Doctor management
		r.Post("/doctors", doctorHandler.Create)
		r.Put("/doctors/{id}", doctorHandler.Update)
		r.Put("/doctors/{id}/working-hours", doctorHandler.SetWorkingHours)

		// Appointment management
		r.Post("/appointments", appointmentHandler.Book)
		r.Patch("/appointments/{id}/cancel", appointmentHandler.Cancel)
		r.Patch("/appointments/{id}/reschedule", appointmentHandler.Reschedule)

		// Patient management
		r.Get("/patients", patientHandler.List)
		r.Post("/patients", patientHandler.Create)
		r.Get("/patients/{id}", patientHandler.GetByID)
		r.Get("/patients/{id}/appointments", patientHandler.GetAppointments)
	})

	// Start server
	addr := fmt.Sprintf(":%s", cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		log.Printf("Server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced shutdown: %v", err)
	}

	log.Println("Server stopped")
}
