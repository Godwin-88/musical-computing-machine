# Clinic Appointment Booking System

A full-stack clinic appointment booking system built with Go, Supabase, and Next.js.

## Architecture

```mermaid
graph TB
    subgraph "Frontend"
        NEXT[Next.js Admin UI<br/>Vercel]
    end

    subgraph "API Layer"
        GO[Go API<br/>Render / Docker]
        CHI[Chi Router]
        JWT[JWT Auth Middleware]
    end

    subgraph "Database"
        SUPABASE[Supabase<br/>PostgreSQL]
        AUTH[Supabase Auth]
        RLS[Row-Level Security]
    end

    NEXT -->|HTTPS + Bearer JWT| GO
    NEXT -->|Supabase JS SDK| AUTH
    GO -->|pgx v5| SUPABASE
    GO -->|JWT Validation| JWT
    SUPABASE --> RLS

    classDef frontend fill:#e1f5fe,stroke:#0288d1
    classDef api fill:#fff3e0,stroke:#f57c00
    classDef db fill:#e8f5e9,stroke:#388e3c
    class NEXT frontend
    class GO,CHI,JWT api
    class SUPABASE,AUTH,RLS db
```

## Data Flow

```mermaid
sequenceDiagram
    participant Admin as Clinic Admin (Browser)
    participant Next as Next.js UI (Vercel)
    participant Auth as Supabase Auth
    participant API as Go API (Render)
    participant DB as PostgreSQL (Supabase)

    Note over Admin,DB: Authentication Flow
    Admin->>Next: GET login page
    Next->>Admin: Login form (email + password)
    Admin->>Next: Submit credentials
    Next->>Auth: signInWithPassword()
    Auth-->>Next: JWT tokens
    Next->>Next: Store session in cookie
    Next-->>Admin: Redirect to dashboard

    Note over Admin,DB: Doctor Management Flow
    Admin->>Next: Click Add Doctor
    Next->>API: GET list doctors
    API->>API: Validate JWT
    API->>DB: Query all doctors
    DB-->>API: doctor records
    API-->>Next: JSON array
    Next-->>Admin: Rendered table

    Admin->>Next: Fill name + specialisation
    Next->>API: POST create doctor
    API->>API: Validate role clinic_admin
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
│   │   ├── middleware/            # Auth + logging
│   │   │   ├── auth.go            # JWT validation + role check
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
│   │   ├── (auth)/login/         # Login page
│   │   ├── (dashboard)/          # Protected pages
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
│   │   ├── api.ts                # Typed fetch wrapper
│   │   └── supabase.ts           # Supabase clients
│   ├── middleware.ts              # Route protection
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
| **Database** | Supabase (managed PostgreSQL via pgx) |
| **Auth** | Supabase Auth (JWT validation with HS256) |
| **Admin UI** | Next.js 14 (App Router), TypeScript, Tailwind CSS, shadcn/ui |
| **Font** | Raleway (via Google Fonts / next/font) |
| **API Hosting** | Render (Docker) |
| **Admin UI Hosting** | Vercel |
| **CI/CD** | GitHub Actions |

## API Endpoints

### Public (no auth required)

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Health check (DB ping → 200 or 503) |
| `GET` | `/doctors` | List all doctors |
| `GET` | `/doctors/{id}` | Get doctor + working hours |
| `GET` | `/doctors/{id}/availability?date=YYYY-MM-DD` | Available slots for a date |

### Protected (requires `clinic_admin` JWT)

| Method | Path | Description |
|---|---|---|
| `POST` | `/doctors` | Create doctor |
| `PUT` | `/doctors/{id}` | Update doctor profile |
| `PUT` | `/doctors/{id}/working-hours` | Set working hours (full replace) |
| `POST` | `/appointments` | Book a slot |
| `PATCH` | `/appointments/{id}/cancel` | Cancel with reason |
| `PATCH` | `/appointments/{id}/reschedule` | Move to new slot |
| `GET` | `/patients` | List all patients (?search=) |
| `POST` | `/patients` | Create patient |
| `GET` | `/patients/{id}` | Get patient details |
| `GET` | `/patients/{id}/appointments` | Patient appointments (?include_past=true) |

### HTTP Status Codes

| Code | Meaning |
|---|---|
| `200` | Success (read / update) |
| `201` | Resource created |
| `401` | Missing or invalid JWT |
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
- Supabase CLI (optional, for local Supabase)

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

# Edit .env with your Supabase credentials (or use defaults for local dev)
# DATABASE_URL=postgres://postgres:postgres@localhost:5432/clinic_dev?sslmode=disable
# SUPABASE_JWT_SECRET=your-supabase-jwt-secret

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

# Edit .env.local:
# NEXT_PUBLIC_SUPABASE_URL=https://<project>.supabase.co
# NEXT_PUBLIC_SUPABASE_ANON_KEY=your-anon-key
# NEXT_PUBLIC_API_URL=http://localhost:8080

# Install dependencies
npm install

# Start development server
npm run dev
```

### 4. Apply database migrations

```bash
cd clinic-api
make migrate   # Requires Supabase CLI
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

## Public URLs

| Service | URL |
|---|---|
| **API** | TBD |
| **Admin UI** | TBD |

## Deployment

- **Branch:** `main` triggers deployment
- **CI:** Runs on every PR — lint (`golangci-lint`) + test (`-race`, coverage ≥ 80%)
- **CD:** On merge to `main` — deploy Go API to Render + Admin UI to Vercel