# Clinic Appointment Booking System

A full-stack clinic appointment booking system built with Go, PostgreSQL, and Next.js.

- **API:** https://musical-computing-machine-04cx.onrender.com
- **Admin UI:** https://clinic-admin-five.vercel.app

## Architecture

```mermaid
graph TB
    subgraph "Frontend"
        NEXT[Next.js Admin UI<br/>Vercel]
    end

    subgraph "API Layer"
        GO[Go API<br/>Render / Docker]
        CHI[Chi Router]
    end

    subgraph "Database"
        PG[PostgreSQL<br/>Supabase / Docker]
    end

    NEXT -->|HTTPS| GO
    GO -->|pgx v5| PG

    classDef frontend fill:#e1f5fe,stroke:#0288d1
    classDef api fill:#fff3e0,stroke:#f57c00
    classDef db fill:#e8f5e9,stroke:#388e3c
    class NEXT frontend
    class GO,CHI api
    class PG db
```

## Data Flow

```mermaid
sequenceDiagram
    participant Admin as Clinic Admin (Browser)
    participant Next as Next.js UI (Vercel)
    participant API as Go API (Render)
    participant DB as PostgreSQL (Supabase)

    Note over Admin,DB: Doctor Management Flow
    Admin->>Next: Click Add Doctor
    Next->>API: GET list doctors
    API->>DB: Query all doctors
    DB-->>API: doctor records
    API-->>Next: JSON array
    Next-->>Admin: Rendered table

    Admin->>Next: Fill name + specialisation
    Next->>API: POST create doctor
    API->>DB: Insert doctor
    DB-->>API: Created record
    API-->>Next: 201 + doctor
    Next-->>Admin: Redirect to schedule

    Admin->>Next: Toggle days + pick times
    Next->>API: PUT working-hours
    API->>DB: Delete old + insert new (transaction)
    DB-->>API: Success
    API-->>Next: 200 OK
    Next-->>Admin: Saved confirmation

    Note over Admin,DB: Slot Availability Query
    Admin->>Next: View booking page
    Next->>API: GET availability by date
    API->>DB: Get working hours
    DB-->>API: hours for weekday
    API->>DB: Get booked slots
    DB-->>API: booked start times
    API->>API: GenerateSlots() pure function
    API-->>Next: available_slots array
    Next-->>Admin: Slot grid

    Note over Admin,DB: Appointment Booking Flow
    Admin->>Next: Select slot + patient
    Next->>API: POST book appointment
    API->>API: Validate rules
    API->>DB: Insert appointment
    DB-->>API: 23505 or success
    alt Slot Taken
        API-->>Next: 409 Conflict
        Next-->>Admin: Slot already booked
    else Success
        API-->>Next: 201 Created
        Next-->>Admin: Redirect to list
    end

    Note over Admin,DB: Cancellation Flow
    Admin->>Next: Click Cancel + enter reason
    Next->>API: PATCH cancel appointment
    API->>DB: Transaction: lock row + update status
    DB-->>API: Updated row
    API-->>Next: 200 OK
    Next-->>Admin: Badge updated to CANCELLED

    Note over Admin,DB: Reschedule Flow
    Admin->>Next: Pick new slot
    Next->>API: PATCH reschedule appointment
    API->>DB: Transaction: FOR UPDATE NOWAIT
    API->>DB: Check new slot availability
    alt New slot taken
        DB-->>API: conflict
        API-->>Next: 409 Conflict
        Next-->>Admin: Error in panel
    else Success
        API->>DB: Update start/end time + commit
        DB-->>API: Updated row
        API-->>Next: 200 OK
        Next-->>Admin: Row updated
    end

    Note over Admin,DB: Patient Management Flow
    Admin->>Next: Search by name or email
    Next->>API: GET patients with search
    API->>DB: ILIKE query
    DB-->>API: Matching patients
    API-->>Next: Patient list
    Next-->>Admin: Rendered table

    Admin->>Next: Fill form + submit
    Next->>API: POST create patient
    API->>DB: Insert patient
    DB-->>API: 23505 or success
    alt Duplicate email
        API-->>Next: 409 Conflict
        Next-->>Admin: Email already exists
    else Success
        API-->>Next: 201 Created
        Next-->>Admin: Redirect to detail
    end
```

## System Design

### Models & Components

**Domain Models:**
- **Doctor** — name, specialisation, slot duration (default 30 min)
- **WorkingHours** — doctor_id, day_of_week (0=Mon..6=Sun), start_time, end_time (on 30-min boundaries)
- **Patient** — name, email (unique), phone
- **Appointment** — doctor_id, patient_id, start_time, end_time, status (BOOKED/CANCELLED), cancellation_reason, cancelled_at

**Key Components:**
- **Handlers** — HTTP layer (Chi router), request parsing, response formatting
- **Services** — Business logic (slot generation, validation rules)
- **Repositories** — Data access via pgx v5 (raw SQL queries, transactions with `FOR UPDATE`/`FOR UPDATE NOWAIT`)
- **Middleware** — Request logging, CORS, panic recovery, request ID

### Key Decisions & Trade-offs

| Decision | Rationale | Trade-off |
|---|---|---|
| **Go + Chi** over FastAPI/Django | Go's simplicity, fast compile times, excellent concurrency for API workloads. Chi is lightweight with no heavy ORM overhead. | More boilerplate for validation compared to Django REST Framework's serializers. |
| **PostgreSQL** | Mature, excellent JSON support, `FOR UPDATE NOWAIT` for pessimistic locking, partial unique indexes for anti-double-booking. | Heavier than SQLite for local dev, but Supabase offers a managed free tier. |
| **Partial unique index** for anti-double-booking | `CREATE UNIQUE INDEX ... WHERE status = 'BOOKED'` prevents concurrent bookings at the database level. | Requires understanding of partial indexes; not as portable to other databases. |
| **Pessimistic locking** (FOR UPDATE / FOR UPDATE NOWAIT) | Prevents race conditions on cancel and reschedule operations without application-level mutexes. | Slightly lower throughput than optimistic locking, but correctness is paramount for bookings. |
| **UUID primary keys** | Avoids sequential ID enumeration, works well with distributed systems (Supabase). | Slightly larger index size vs auto-increment integers. |
| **No authentication** | MVP scope — the spec says "Patients need to book appointments online" without specifying auth. | Would need to add auth for production use (Supabase Auth or JWT). |
| **1-hour booking buffer** | Prevents last-minute bookings where a patient might not make it in time (bonus requirement). | Hard-coded; could be configurable per-clinic in future. |
| **Go's `time.Time` for slot generation** | Pure function, fully testable, no database dependency for generating available slots. | Requires careful timezone handling (all times in UTC). |

## Project Structure

```
/
├── clinic-api/                    # Go API (Chi + pgx)
│   ├── cmd/server/main.go         # Entry point
│   ├── internal/
│   │   ├── config/config.go       # Env loading
│   │   ├── database/db.go         # pgx pool connection
│   │   ├── errors/errors.go       # APIError type + helpers
│   │   ├── handlers/              # HTTP handlers
│   │   │   ├── doctors.go
│   │   │   ├── appointments.go
│   │   │   ├── patients.go
│   │   │   └── health.go
│   │   ├── middleware/            # Logging only
│   │   │   └── logger.go          # Request logging
│   │   ├── models/                # Domain structs
│   │   │   ├── doctor.go
│   │   │   ├── patient.go
│   │   │   ├── appointment.go
│   │   │   └── working_hours.go
│   │   ├── repositories/          # Raw pgx queries
│   │   │   ├── doctor_repo.go
│   │   │   ├── appointment_repo.go
│   │   │   └── patient_repo.go
│   │   └── services/              # Business logic
│   │       ├── slot_service.go
│   │       └── appointment_service.go
│   ├── migrations/                # Database migrations
│   │   ├── 001_initial_schema.sql
│   │   └── 002_rls_policies.sql
│   ├── Dockerfile                 # Multi-stage build
│   ├── docker-compose.yml         # Local dev environment
│   ├── Makefile
│   ├── go.mod
│   └── .env.example
│
├── clinic-admin/                  # Next.js Admin UI
│   ├── app/
│   │   ├── (dashboard)/          # Dashboard pages (public)
│   │   │   ├── doctors/          # List, new, schedule editor
│   │   │   ├── appointments/     # List, new booking flow
│   │   │   └── patients/         # List, new, detail + history
│   │   └── layout.tsx            # Root layout (Raleway font)
│   ├── components/
│   │   ├── ui/button.tsx         # shadcn/ui button
│   │   ├── DoctorForm.tsx
│   │   ├── AppointmentTable.tsx
│   │   ├── SlotPicker.tsx
│   │   └── PatientForm.tsx
│   ├── lib/
│   │   └── api.ts                # Typed fetch wrapper
│   ├── tailwind.config.ts         # Tailwind + Raleway config
│   └── .env.local.example
│
├── .github/workflows/
│   ├── ci.yml                    # PR: lint + test + coverage
│   └── cd.yml                    # Merge to main: deploy
└── README.md
```

## Tech Stack

| Layer | Technology |
|---|---|
| **API** | Go 1.23, Chi router, pgx v5 |
| **Database** | PostgreSQL (Supabase managed) |
| **Admin UI** | Next.js 15 (App Router), TypeScript, Tailwind CSS, shadcn/ui |
| **Font** | Raleway (via Google Fonts / next/font) |
| **API Hosting** | Render (Docker / native Go build) |
| **Admin UI Hosting** | Vercel |
| **CI/CD** | GitHub Actions |

## Postman Collection

A complete Postman collection is included in the repository at `clinic-api.postman_collection.json`. It contains 15 pre-configured requests that walk through the full API flow:

1. **Health Check** — verify API is running
2. **Create Doctor** — creates a doctor and stores the ID
3. **Set Working Hours** — sets Mon-Fri working hours
4. **List Doctors** — verify doctor was created
5. **Get Availability** — fetch available slots for today
6. **Create Patient** — creates a patient and stores the ID
7. **Book Appointment** — books the first available slot
8. **List Appointments** — verify booking with doctor/patient names
9. **Cancel Appointment** — cancel with a reason
10. **Cancel Already Cancelled** — verify 409 error on double-cancel
11. **Book Another Appointment** — creates a new appointment for reschedule testing
12. **Reschedule Appointment** — move to a different slot
13. **Get Patient Appointments** — bonus endpoint with `include_past=true`
14. **Validation Error** — verify 422 on missing fields
15. **404 Not Found** — verify 404 on non-existent doctor

**To use:**
1. Open Postman
2. Click **Import** → **Upload Files** → select `clinic-api.postman_collection.json`
3. Run requests in order — variables are automatically chained

## API Endpoints

All endpoints are **public** (no authentication required).

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Health check (DB ping → 200 or 503) |
| `GET` | `/doctors` | List all doctors |
| `POST` | `/doctors` | Create doctor |
| `GET` | `/doctors/{id}` | Get doctor + working hours |
| `PUT` | `/doctors/{id}` | Update doctor profile |
| `PUT` | `/doctors/{id}/working-hours` | Set working hours (full replace) |
| `GET` | `/doctors/{id}/availability?date=YYYY-MM-DD` | Available slots for a date |
| `POST` | `/appointments` | Book a slot |
| `PATCH` | `/appointments/{id}/cancel` | Cancel with reason |
| `PATCH` | `/appointments/{id}/reschedule` | Move to new slot |
| `GET` | `/appointments` | List appointments (?doctor_id=&status=) |
| `GET` | `/patients` | List all patients (?search=) |
| `POST` | `/patients` | Create patient |
| `GET` | `/patients/{id}` | Get patient details |
| `GET` | `/patients/{id}/appointments` | Patient appointments (?include_past=true) |

### HTTP Status Codes

| Code | Meaning |
|---|---|
| `200` | Success (read / update) |
| `201` | Resource created |
| `404` | Resource not found |
| `409` | State conflict (slot taken / already cancelled) |
| `422` | Validation failure |
| `503` | Database unreachable |

### Error Response Format

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Doctor name is required."
  }
}
```

## Local Setup

### Prerequisites

- Go 1.23+
- Node.js 20+
- Docker & Docker Compose

### 1. Clone the repository

```bash
git clone <repo-url>
cd clinic-booking-system
```

### 2. Start the Go API (with Docker Compose)

```bash
cd clinic-api

# Copy environment file
cp .env.example .env

# Edit .env if needed (defaults work for local Docker PostgreSQL)
# DATABASE_URL=postgres://postgres:postgres@localhost:5432/clinic_dev?sslmode=disable

# Start API + PostgreSQL
docker compose up -d

# Or run locally:
# go run ./cmd/server
```

### 3. Set up the Admin UI

```bash
cd clinic-admin

# Copy environment file
cp .env.local.example .env.local

# Edit .env.local if needed:
# NEXT_PUBLIC_API_URL=http://localhost:8080

# Install dependencies
npm install

# Start development server
npm run dev
```

### 4. Apply database migrations

```bash
# Using psql against local PostgreSQL:
psql postgres://postgres:postgres@localhost:5432/clinic_dev -f migrations/001_initial_schema.sql
```

### 5. Run tests

```bash
cd clinic-api

# Unit tests (no DB required)
go test ./internal/services/...

# All tests (requires PostgreSQL running)
go test -race ./...
```

### Quick start (one-liner)

```bash
cd clinic-api && docker compose up -d
cd ../clinic-admin && npm install && npm run dev
```

Access the admin UI at [http://localhost:3000](http://localhost:3000) and the API at [http://localhost:8080](http://localhost:8080).

## Deployment

### CI/CD Pipeline

- **CI:** Runs on every PR to `main`:
  1. Lint with `golangci-lint` (v1.64.8)
  2. Apply migrations to a test PostgreSQL database
  3. Run `go test -race -coverprofile=coverage.out ./...`
  4. Check coverage ≥ 80%
- **CD:** On merge/push to `main`:
  1. Runs the same test suite
  2. Deploys Go API to Render via deploy hook
  3. Deploys Admin UI to Vercel via `amondnet/vercel-action`

### Public URLs

| Service | URL |
|---|---|
| **API** | https://musical-computing-machine-04cx.onrender.com |
| **Admin UI** | https://clinic-admin-five.vercel.app | for testing the endpoints on forms

### Deployment triggers

- **Branch:** `main`
- **API (Render):** Auto-deployed via GitHub Actions using a Render Deploy Hook
- **Admin UI (Vercel):** Auto-deployed via Vercel CLI from GitHub Actions (requires `VERCEL_TOKEN`, `VERCEL_ORG_ID`, `VERCEL_PROJECT_ID` GitHub secrets)

---

## Section 4: AI Reflection

### 1. What did you use AI for across the four sections?

- **Section 1 (System Design):** AI helped generate Mermaid diagrams for the architecture overview and data flow sequence diagram. I also used it to brainstorm trade-offs for technology choices (Go vs FastAPI, UUID vs serial IDs, optimistic vs pessimistic locking).
- **Section 2 (API Implementation):** AI helped generate Go handler boilerplate, error handling patterns, repository layer code, and the slot generation algorithm. It also helped write the test cases for `GenerateSlots`. The AI was used to create the initial structure and then iteratively refined.
- **Section 3 (Deployment & CI/CD):** AI helped write the GitHub Actions workflow YAML files, the Dockerfile, the multi-stage build configuration, and the Render + Vercel deployment setup.
- **Section 4 (AI Reflection):** This document itself was organized with AI assistance, though the substance of the answers reflects my own experience.

### 2. Give one example where an AI suggestion improved your work. What did you prompt it with?

**Prompt:** *"How should I prevent double-booking in a PostgreSQL-based clinic booking system? Consider concurrent requests."*

**AI Suggestion:** Use a PostgreSQL partial unique index on `(doctor_id, start_time) WHERE status = 'BOOKED'` combined with `FOR UPDATE` row-level locking in transactions for cancel/reschedule operations. The AI explained that the partial index would catch duplicate insert attempts at the database level (returning a 23505 unique violation), while `FOR UPDATE NOWAIT` would prevent race conditions between concurrent cancel and reschedule operations on the same appointment.

**Why it improved my work:** I was initially planning to handle booking conflicts entirely in application code with SELECT-then-INSERT logic, which has TOCTOU (time-of-check-time-of-use) race conditions. The AI's database-level approach is provably correct — the partial unique index guarantees no double-booking regardless of concurrent requests, and the pessimistic locking ensures consistency for state-changing operations. This is the kind of architectural decision where a second opinion added real value.

### 3. Give one example where AI output was wrong or incomplete and how you caught it.

**Example:** The AI generated the reschedule endpoint validation but **omitted the check for working hours** on the new slot. It validated the 30-minute boundary, checked the slot wasn't in the past, checked the 1-hour buffer, and checked for slot conflicts — but it never verified that the new time fell within the doctor's defined working hours for that day of the week.

**How I caught it:** During manual testing, I created a doctor with working hours 09:00–12:00, booked an appointment at 09:30, then tried to reschedule it to 14:00. The API accepted it even though the doctor wasn't working at 2 PM. I traced through the code and noticed the `Reschedule` handler in `appointments.go` never called `doctorRepo.GetWorkingHours()` or `services.GenerateSlots()` to validate the new slot against working hours — it only checked for conflicts with other bookings.

### 4. Name two decisions you made without AI. Why did you trust your own judgment there?

**Decision 1: Using Go over FastAPI/Django REST Framework.**

**Why I trusted my own judgment:** I've worked with FastAPI extensively and appreciate Python's rapid development speed. However, for this project I chose Go because:
- The spec emphasises correctness and thought process over speed of delivery
- Go's type system makes refactoring safe and catches entire classes of bugs at compile time
- Go's standard library `net/http` + Chi provides a lightweight, predictable runtime without magic
- The deployment surface is minimal — a single static binary vs a Python environment with dependencies

For a production booking system that must handle concurrent requests correctly, I value Go's simplicity and explicitness over Python's convenience.

**Decision 2: Using a partial unique index (`WHERE status = 'BOOKED'`) instead of an application-level lock.**

**Why I trusted my own judgment:** I've seen too many race condition bugs caused by "check then insert" patterns in booking systems. An application-level mutex or lock would work but adds complexity and becomes a bottleneck. A database-level partial unique index:
- Is declarative — the database guarantees the invariant
- Has minimal performance impact (PostgreSQL checks the index on insert)
- Handles concurrent requests automatically without any coordination code
- Is well-documented PostgreSQL behaviour that I've used successfully before

This is a pattern I've implemented in production systems, so I was confident in the trade-off even without AI validation.