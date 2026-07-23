# Clinic Appointment Booking System — Product Specification v2

> **Stack:** Go (Chi) · Supabase (PostgreSQL + Auth + RLS) · Render (API) · Next.js (Admin UI)
>
> **Capability Canvas Alignment:** `Manage Digital Core` › `Manage Healthcare Provider Core Operations`
> enabled by `Manage Digital Backoffice` · `Manage Digital IT` · `Manage Digital Security` · `Manage Digital Channels`
>
> **Document Version:** 2.0 | **Prepared by:** Solutions Architect | **Date:** 2026-07-23

---

## Table of Contents

1. [Section 1 — System Design](#section-1--system-design)
   - 1.1 [Domain Context from Capability Canvas](#11-domain-context-from-capability-canvas)
   - 1.2 [Full Stack Architecture](#12-full-stack-architecture)
   - 1.3 [Data Models](#13-data-models)
   - 1.4 [Supabase Integration Points](#14-supabase-integration-points)
   - 1.5 [Security: Auth & Row-Level Security](#15-security-auth--row-level-security)
   - 1.6 [Go Project Layout](#16-go-project-layout)
   - 1.7 [Key Design Decisions & Trade-offs](#17-key-design-decisions--trade-offs)
2. [Section 2 — API Specification (Go + Chi)](#section-2--api-specification)
   - [EPIC-01 — Doctor Schedule Management](#epic-01--doctor-schedule-management)
   - [EPIC-02 — Appointment Booking](#epic-02--appointment-booking)
   - [EPIC-03 — Appointment Lifecycle Management](#epic-03--appointment-lifecycle-management)
   - [EPIC-04 — Patient Appointment Portal](#epic-04--patient-appointment-portal)
   - [EPIC-05 — System Integrity & Validation Engine](#epic-05--system-integrity--validation-engine)
3. [Section 3 — Next.js Admin UI](#section-3--nextjs-admin-ui)
   - [EPIC-06 — Admin Authentication & Access Control](#epic-06--admin-authentication--access-control)
   - [EPIC-07 — Doctor & Schedule Management UI](#epic-07--doctor--schedule-management-ui)
   - [EPIC-08 — Booking Management UI](#epic-08--booking-management-ui)
   - [EPIC-09 — Patient Management UI](#epic-09--patient-management-ui)
4. [Section 4 — Deployment & CI/CD](#section-4--deployment--cicd)
   - [EPIC-10 — Supabase Infrastructure Setup](#epic-10--supabase-infrastructure-setup)
   - [EPIC-11 — Render Deployment (Go API)](#epic-11--render-deployment-go-api)
   - [EPIC-12 — CI/CD Pipeline](#epic-12--cicd-pipeline)
5. [Appendix A — Entity Relationship Summary](#appendix-a--entity-relationship-summary)
6. [Appendix B — API Contract Summary](#appendix-b--api-contract-summary)
7. [Appendix C — RLS Policy Summary](#appendix-c--rls-policy-summary)
8. [Appendix D — Capability Canvas Traceability Matrix](#appendix-d--capability-canvas-traceability-matrix)

---

## Section 1 — System Design

### 1.1 Domain Context from Capability Canvas

The canvas positions this system within **`Manage Healthcare Provider Core Operations`**, enabled by the following cross-cutting digital domains:

| Enabling Domain (Canvas) | Role in This System |
|---|---|
| `Manage Digital Backoffice` › `Manage Human Resources` | Doctor roster, working-hours config |
| `Manage Digital Backoffice` › `Manage Finance` | Future: billing per appointment |
| `Manage Digital IT` | Go API on Render, Supabase managed DB |
| `Manage Digital Security` | Supabase Auth JWT + RLS policies |
| `Manage Digital Channels` | REST API consumed by Next.js Admin UI and future patient-facing web/mobile |
| `Manage Digital Intelligence` | Future: booking analytics, no-show dashboards |

---

### 1.2 Full Stack Architecture

```
┌───────────────────────────────────────────────────────────────────┐
│                      Next.js Admin UI                              │
│                (Vercel or Render Static Site)                      │
│   Pages: /login  /doctors  /appointments  /patients                │
│   Auth: Supabase Auth JS SDK (session management)                  │
└────────────┬──────────────────────────────────┬───────────────────┘
             │ HTTPS (Supabase Auth SDK)         │ HTTPS + Bearer JWT
             │                                   │
┌────────────▼──────────┐          ┌─────────────▼───────────────────┐
│   Supabase Auth       │          │   Go API on Render               │
│   (email/password     │          │   (Chi router)                   │
│    + JWT issuance)    │          │                                  │
│                       │          │  /internal/                      │
│  RLS Policies on      │          │    handlers/                     │
│  all tables           │          │    services/                     │
└────────────┬──────────┘          │    repositories/                 │
             │ pgx/supabase        │    middleware/                   │
             │ connection pooler   │    models/                       │
┌────────────▼──────────────────────▼──────────────────────────────┐
│                     Supabase (PostgreSQL)                          │
│   Tables: doctors  working_hours  patients  appointments           │
│   Auth schema: auth.users (managed by Supabase)                   │
│   RLS: enabled on all tables                                      │
│   Unique index: (doctor_id, start_time) WHERE status='BOOKED'     │
└───────────────────────────────────────────────────────────────────┘
```

**Request authentication flow:**

```
Next.js Admin UI
  → Supabase Auth login → receives JWT (access_token)
  → attaches JWT as Authorization: Bearer <token> on all API calls
  → Go API middleware validates JWT against Supabase JWKS endpoint
  → handler proceeds if valid; 401 if not
```

---

### 1.3 Data Models

#### `doctors`

| Column | Type | Notes |
|---|---|---|
| `id` | `uuid` PK | `gen_random_uuid()` |
| `name` | `text NOT NULL` | |
| `specialisation` | `text NOT NULL` | |
| `slot_duration_minutes` | `integer NOT NULL DEFAULT 30` | |
| `created_at` | `timestamptz DEFAULT now()` | |
| `updated_at` | `timestamptz DEFAULT now()` | |

#### `working_hours`

| Column | Type | Notes |
|---|---|---|
| `id` | `uuid` PK | |
| `doctor_id` | `uuid FK → doctors.id` | `ON DELETE CASCADE` |
| `day_of_week` | `integer NOT NULL` | 0=Monday … 6=Sunday |
| `start_time` | `time NOT NULL` | `HH:MM:SS` |
| `end_time` | `time NOT NULL` | `HH:MM:SS` |
| `UNIQUE` | `(doctor_id, day_of_week)` | One schedule per doctor per day |

#### `patients`

| Column | Type | Notes |
|---|---|---|
| `id` | `uuid` PK | |
| `auth_user_id` | `uuid NULLABLE` | Links to `auth.users.id` for future patient self-service |
| `name` | `text NOT NULL` | |
| `email` | `text UNIQUE NOT NULL` | |
| `phone` | `text` | |
| `created_at` | `timestamptz DEFAULT now()` | |

#### `appointments`

| Column | Type | Notes |
|---|---|---|
| `id` | `uuid` PK | |
| `doctor_id` | `uuid FK → doctors.id` | |
| `patient_id` | `uuid FK → patients.id` | |
| `start_time` | `timestamptz NOT NULL` | UTC |
| `end_time` | `timestamptz NOT NULL` | Computed: `start_time + interval '30 minutes'` |
| `status` | `text NOT NULL DEFAULT 'BOOKED'` | `CHECK (status IN ('BOOKED','CANCELLED','RESCHEDULED'))` |
| `cancellation_reason` | `text` | NULL unless cancelled |
| `cancelled_at` | `timestamptz` | NULL unless cancelled |
| `created_at` | `timestamptz DEFAULT now()` | |
| `updated_at` | `timestamptz DEFAULT now()` | |

**Critical index:**
```sql
CREATE UNIQUE INDEX appointments_no_double_booking
  ON appointments (doctor_id, start_time)
  WHERE status = 'BOOKED';
```

---

### 1.4 Supabase Integration Points

| Feature | How Used |
|---|---|
| **Managed PostgreSQL** | Primary database; connection via `DATABASE_URL` (pooler mode for API, direct for migrations) |
| **Supabase Auth** | Issues JWTs for admin users; Go API validates them via JWKS |
| **Row-Level Security (RLS)** | Applied to all tables — see Section 1.5 |
| **Supabase JS SDK** | Used in Next.js for login, session management, token refresh |
| **Supabase Migrations** | `supabase db push` / `supabase migration new` for schema versioning |
| **Supabase Dashboard** | Admin can inspect tables directly; useful for ops |
| **Connection Pooler (PgBouncer)** | Go API connects via port `6543` (transaction mode) for scalability |

---

### 1.5 Security: Auth & Row-Level Security

#### JWT Validation (Go Middleware)

The Go API validates every protected request:

1. Extracts `Authorization: Bearer <jwt>` header.
2. Fetches Supabase JWKS from `https://<project>.supabase.co/auth/v1/keys` (cached with TTL).
3. Verifies signature, `exp`, and `iss` claims.
4. Injects `user_id` and `role` into the request context.
5. Returns `401 Unauthorized` if token is absent, expired, or invalid.

#### Roles

| Role | Description | Assigned via |
|---|---|---|
| `clinic_admin` | Full read/write on all resources | Supabase Dashboard → user metadata |
| `service_role` | API-to-Supabase internal calls | Supabase service key (never exposed to browser) |

#### RLS Policies (Summary — see Appendix C for full SQL)

| Table | Policy | Effect |
|---|---|---|
| `doctors` | Admin read/write | Only `clinic_admin` JWT can INSERT/UPDATE/DELETE |
| `working_hours` | Admin read/write | Same as doctors |
| `patients` | Admin read/write | Admin can manage all; future: patient reads own row |
| `appointments` | Admin read/write | Admin can manage all; future: patient reads/cancels own |

> **Note:** The Go API connects using the Supabase **service role key** (bypasses RLS) for server-side operations. RLS primarily protects direct Supabase client calls from the Next.js UI if it ever queries the DB directly.

---

### 1.6 Go Project Layout

```
clinic-api/
  cmd/
    server/
      main.go               # Entry point: config load, DB connect, router start
  internal/
    config/
      config.go             # Env var loading via os.Getenv / godotenv
    database/
      db.go                 # pgx pool setup, Supabase connection string
    middleware/
      auth.go               # JWT validation middleware (Chi middleware pattern)
      logger.go             # Request logging
    models/
      doctor.go
      patient.go
      appointment.go
      working_hours.go
    handlers/
      doctors.go            # HTTP handlers for /doctors
      appointments.go       # HTTP handlers for /appointments
      patients.go           # HTTP handlers for /patients
      health.go
    services/
      slot_service.go       # Slot computation logic (pure functions, testable)
      appointment_service.go
    repositories/
      doctor_repo.go        # pgx queries
      appointment_repo.go
      patient_repo.go
    errors/
      errors.go             # APIError type, helper constructors
  migrations/               # Supabase SQL migrations
    001_initial_schema.sql
    002_rls_policies.sql
  Dockerfile
  docker-compose.yml        # Local dev: Go API + local Supabase (via supabase CLI)
  go.mod
  go.sum
  .env.example
  README.md

clinic-admin/               # Next.js app (separate repo or monorepo subfolder)
  app/
    (auth)/login/page.tsx
    (dashboard)/
      doctors/page.tsx
      doctors/[id]/page.tsx
      appointments/page.tsx
      patients/page.tsx
  components/
    DoctorForm.tsx
    AppointmentTable.tsx
    SlotPicker.tsx
    PatientForm.tsx
  lib/
    supabase.ts             # Supabase JS client
    api.ts                  # Fetch wrapper → Go API with JWT
  middleware.ts             # Next.js route protection (check session)
```

---

### 1.7 Key Design Decisions & Trade-offs

| Decision | Choice | Rationale | Trade-off |
|---|---|---|---|
| Go + Chi | Chi over Gin/Echo | Lightweight, idiomatic, stdlib-compatible; no magic; easy to test handlers | Less ecosystem sugar than Gin; slightly more boilerplate |
| Supabase Auth (not custom JWT) | Supabase Auth | Auth-as-a-service; handles token issuance, refresh, user mgmt; no bcrypt/JWT infra to build | Tied to Supabase; migration to another provider requires re-issuing sessions |
| RLS on all tables | Enabled even though API uses service role | Defense-in-depth; Next.js could query Supabase directly in future without opening security holes | Requires discipline keeping policies in sync with migrations |
| Slot computation | On-the-fly (not pre-generated rows) | No dead slot rows; instant schedule changes; trivial to add doctors | Slightly heavier read path — negligible for 5 doctors |
| pgx (not GORM) | pgx v5 direct | Full SQL control; no ORM magic hiding query behaviour; better for complex slot queries | More verbose; no auto-migration |
| Supabase connection pooler | PgBouncer (port 6543) | Handles Go's goroutine-per-request concurrency without exhausting DB connections | Transaction mode: no session-level features (SET LOCAL, advisory locks — use app-level locks instead) |
| Render for API hosting | Render | Simple Docker deploy; free tier available; auto-deploy from GitHub | Cold starts on free tier (mitigated by health check pings) |
| Atomic reschedule | Single DB transaction with `FOR UPDATE` lock | Prevents race where two clients try to claim the same new slot | Short transaction window; must avoid network I/O inside transaction |
| Next.js Admin UI auth | Supabase JS SDK session | Token refresh handled automatically; SSR-compatible | Admin must have Supabase account; not suitable for non-technical clinic staff without onboarding |

---

---

## Section 2 — API Specification

> All protected endpoints require `Authorization: Bearer <supabase_jwt>` with role `clinic_admin`.
> The JWT is validated by the Go Chi middleware before any handler executes.

---

## EPIC-01 — Doctor Schedule Management

**Canvas alignment:** `Manage Healthcare Provider Core Operations` › `Manage Clinical Workforce` › **Manage Doctor Schedule**

**Epic Goal:** Enable clinic admins to create and configure doctor profiles and their per-weekday working hours, which are the foundation for all slot availability calculations.

---

### FEATURE-01.1 — Doctor Profile Management

---

#### US-01.1.1 — Create a Doctor Profile

**As a** clinic admin,
**I want to** create a new doctor record with name and specialisation,
**So that** the doctor appears in the system and can be assigned working hours and receive bookings.

**Endpoint:** `POST /doctors` *(protected)*

**Request Body:**
```json
{
  "name": "Dr. Amina Osei",
  "specialisation": "General Practice",
  "slot_duration_minutes": 30
}
```

**Acceptance Criteria:**

- [ ] Returns `201 Created` with the new doctor object including `id`, `name`, `specialisation`, `slot_duration_minutes`, `created_at`.
- [ ] `name` is required; if blank, return `422`: `"Doctor name is required."`.
- [ ] `specialisation` is required; if blank, return `422`: `"Doctor specialisation is required."`.
- [ ] `slot_duration_minutes` defaults to `30` if not provided; only `30` is accepted in v1 — any other value returns `422`: `"Only 30-minute slots are supported in this version."`.
- [ ] Requires valid `clinic_admin` JWT; missing or invalid token returns `401 Unauthorized`.

---

#### US-01.1.2 — List All Doctors

**As a** patient (unauthenticated) or admin,
**I want to** retrieve the list of all doctors,
**So that** I can browse available practitioners.

**Endpoint:** `GET /doctors` *(public)*

**Acceptance Criteria:**

- [ ] Returns `200 OK` with array of `{ id, name, specialisation }`.
- [ ] Returns `200 OK` with `[]` if no doctors exist.
- [ ] No authentication required.

---

#### US-01.1.3 — Get a Single Doctor

**Endpoint:** `GET /doctors/{id}` *(public)*

**Acceptance Criteria:**

- [ ] Returns `200 OK` with full doctor object including working hours array.
- [ ] Working hours format: `[{ "day_of_week": 0, "start_time": "09:00", "end_time": "17:00" }, ...]`.
- [ ] Returns `404 Not Found`: `"Doctor with id '{id}' not found."` if not found.

---

#### US-01.1.4 — Update a Doctor Profile

**Endpoint:** `PUT /doctors/{id}` *(protected)*

**Acceptance Criteria:**

- [ ] Returns `200 OK` with the updated doctor object.
- [ ] Only `name` and `specialisation` are updatable via this endpoint.
- [ ] Returns `404` if doctor does not exist.
- [ ] Requires `clinic_admin` JWT.

---

### FEATURE-01.2 — Working Hours Configuration

---

#### US-01.2.1 — Set Working Hours for a Doctor

**As a** clinic admin,
**I want to** define which days and hours a doctor is available each week,
**So that** the system can generate accurate slot availability for patients.

**Endpoint:** `PUT /doctors/{id}/working-hours` *(protected)*

**Request Body:**
```json
{
  "working_hours": [
    { "day_of_week": 0, "start_time": "09:00", "end_time": "17:00" },
    { "day_of_week": 2, "start_time": "09:00", "end_time": "13:00" },
    { "day_of_week": 4, "start_time": "14:00", "end_time": "18:00" }
  ]
}
```

**Acceptance Criteria:**

- [ ] Returns `200 OK` with the complete updated working hours for the doctor.
- [ ] This is a **full replacement** — any weekday not included in the payload is cleared.
- [ ] `day_of_week` must be 0–6; invalid values return `422`: `"day_of_week must be between 0 (Monday) and 6 (Sunday)."`.
- [ ] `start_time` must be before `end_time`; violation returns `422`: `"start_time must be before end_time."`.
- [ ] Times must be on 30-minute boundaries (`:00` or `:30`); violation returns `422`: `"Working hours must start and end on 30-minute boundaries."`.
- [ ] Duplicate `day_of_week` values in the payload return `422`: `"Duplicate day_of_week entries are not allowed."`.
- [ ] Returns `404` if doctor does not exist.
- [ ] Requires `clinic_admin` JWT.

---

#### US-01.2.2 — Slot Alignment Enforcement

**As a** system,
**I want to** reject any booking whose start time does not align to 30-minute boundaries within the doctor's working hours,
**So that** the doctor's schedule is never fragmented.

**Acceptance Criteria:**

- [ ] A `start_time` of e.g. `09:15` for a 30-minute-slot doctor returns `422`: `"Slot start time must align to 30-minute boundaries."`.
- [ ] `end_time` is always derived as `start_time + 30 minutes` — never accepted from the client.

---

---

## EPIC-02 — Appointment Booking

**Canvas alignment:** `Manage Healthcare Provider Core Operations` › `Manage Patient Access` › **Manage Appointment Scheduling**

---

### FEATURE-02.1 — Doctor Availability Query

---

#### US-02.1.1 — View Available Slots for a Doctor on a Date

**As a** patient or admin,
**I want to** query all free 30-minute slots for a specific doctor on a given date,
**So that** I can pick a slot to book.

**Endpoint:** `GET /doctors/{id}/availability?date=YYYY-MM-DD` *(public)*

**Acceptance Criteria:**

- [ ] Returns `200 OK`: `{ "doctor_id": "...", "date": "YYYY-MM-DD", "available_slots": [ { "start": "2026-08-01T09:00:00Z", "end": "2026-08-01T09:30:00Z" }, ... ] }`.
- [ ] Slots are computed from working hours for that weekday, minus already-`BOOKED` appointments.
- [ ] Slots whose `start_time` is in the past (relative to UTC now) are excluded.
- [ ] **Bonus:** Slots starting within 1 hour of UTC now are excluded.
- [ ] If the doctor has no working hours for that weekday, return `200 OK` with `"available_slots": []`.
- [ ] If `date` is missing/malformed, return `422`: `"Query parameter 'date' is required in YYYY-MM-DD format."`.
- [ ] If `date` is in the past, return `422`: `"Cannot query availability for a past date."`.
- [ ] If doctor not found, return `404`: `"Doctor with id '{id}' not found."`.
- [ ] Slots returned in ascending chronological order.
- [ ] No authentication required.

**Go implementation note:** Slot computation lives entirely in `internal/services/slot_service.go` as pure functions (`GenerateSlots(workingHours, bookedSlots, referenceTime time.Time) []Slot`) — no DB calls inside the function, making it fully unit-testable.

---

### FEATURE-02.2 — Slot Booking

---

#### US-02.2.1 — Book an Available Appointment

**As a** patient (via admin or future self-service),
**I want to** book a specific 30-minute slot with a doctor,
**So that** the time is reserved exclusively for me.

**Endpoint:** `POST /appointments` *(protected)*

**Request Body:**
```json
{
  "doctor_id": "uuid",
  "patient_id": "uuid",
  "start_time": "2026-08-01T09:00:00+03:00"
}
```

**Acceptance Criteria:**

- [ ] Returns `201 Created` with full appointment object: `{ id, doctor_id, doctor_name, patient_id, patient_name, start_time, end_time, status }`.
- [ ] `Location` response header: `/appointments/{id}`.
- [ ] `start_time` validated against doctor's working hours → `422`: `"The requested slot is outside the doctor's working hours."`.
- [ ] `start_time` not in the past → `422`: `"Cannot book an appointment in the past."`.
- [ ] `start_time` at least 1 hour from now (bonus) → `422`: `"Appointments must be booked at least 1 hour in advance."`.
- [ ] `start_time` on 30-min boundary → `422`: `"Slot start time must align to 30-minute boundaries."`.
- [ ] Slot not already booked → `409`: `"The requested slot is already booked. Please choose another time."`.
- [ ] `doctor_id` exists → `404`: `"Doctor with id '{id}' not found."`.
- [ ] `patient_id` exists → `404`: `"Patient with id '{id}' not found."`.
- [ ] Concurrent requests for the same slot: exactly one `201`, one `409` (enforced by DB unique partial index — Go catches `pgconn.PgError` with code `23505` and maps to `409`).
- [ ] Requires `clinic_admin` JWT.

---

#### US-02.2.2 — Concurrent Booking Race Condition Handling (Go-specific)

**As a** system,
**I want to** handle simultaneous booking attempts for the same slot gracefully,
**So that** double-booking is impossible even under concurrent load.

**Acceptance Criteria:**

- [ ] The repository layer detects PostgreSQL error code `23505` (unique violation) on the `appointments_no_double_booking` index and returns a typed `ErrSlotConflict` error.
- [ ] The handler maps `ErrSlotConflict` to `409 Conflict` with the message: `"The requested slot is already booked. Please choose another time."`.
- [ ] A concurrent booking test (using goroutines) verifies that sending 10 simultaneous booking requests for the same slot results in exactly 1 success and 9 conflicts.

---

---

## EPIC-03 — Appointment Lifecycle Management

**Canvas alignment:** `Manage Healthcare Provider Core Operations` › `Manage Patient Access` › **Manage Appointment Lifecycle**

---

### FEATURE-03.1 — Appointment Cancellation

---

#### US-03.1.1 — Cancel an Active Appointment

**As a** clinic admin,
**I want to** cancel a booked appointment with a reason,
**So that** the slot is freed for another patient and a record of the cancellation is kept.

**Endpoint:** `PATCH /appointments/{id}/cancel` *(protected)*

**Request Body:**
```json
{ "reason": "Patient unable to attend due to travel" }
```

**Acceptance Criteria:**

- [ ] Returns `200 OK` with updated appointment: `status: "CANCELLED"`, `cancellation_reason`, `cancelled_at`.
- [ ] Freed slot immediately reappears in `GET /doctors/{id}/availability` for that date.
- [ ] `reason` required, non-blank → `422`: `"Cancellation reason is required."`.
- [ ] `reason` max 500 chars → `422`: `"Cancellation reason must not exceed 500 characters."`.
- [ ] Already cancelled → `409`: `"Appointment '{id}' is already cancelled."`.
- [ ] Not found → `404`: `"Appointment with id '{id}' not found."`.
- [ ] Requires `clinic_admin` JWT.

---

#### US-03.1.2 — Prevent Double Cancellation

**Acceptance Criteria:**

- [ ] `PATCH /appointments/{id}/cancel` on `status = "CANCELLED"` returns `409` without modifying the record.
- [ ] Existing `cancellation_reason` and `cancelled_at` are unchanged.

---

### FEATURE-03.2 — Appointment Rescheduling

---

#### US-03.2.1 — Reschedule an Active Appointment

**As a** clinic admin,
**I want to** move a booked appointment to a different valid slot,
**So that** the patient's commitment is preserved with minimum disruption.

**Endpoint:** `PATCH /appointments/{id}/reschedule` *(protected)*

**Request Body:**
```json
{ "new_start_time": "2026-08-05T14:00:00+03:00" }
```

**Acceptance Criteria:**

- [ ] Returns `200 OK` with updated appointment showing new `start_time`, `end_time`, `status: "BOOKED"`.
- [ ] Original slot released and new slot claimed atomically within a single pgx transaction.
- [ ] New slot undergoes full validation: working hours, not past, not within 1 hour, 30-min aligned, not taken.
- [ ] New slot already taken → `409`: `"The requested slot is already booked. Please choose another time."`.
- [ ] `status = "CANCELLED"` → `409`: `"Cannot reschedule a cancelled appointment."`.
- [ ] `new_start_time` equals current `start_time` → `422`: `"New start time must differ from the current appointment time."`.
- [ ] Not found → `404`: `"Appointment with id '{id}' not found."`.
- [ ] Transaction rollback on any failure: original appointment remains intact.
- [ ] Requires `clinic_admin` JWT.

---

#### US-03.2.2 — Atomic Slot Swap (Go pgx Transaction)

**As a** system,
**I want to** use a pgx transaction with `SELECT ... FOR UPDATE` on the original appointment,
**So that** no race condition can cause two patients to hold the same slot after a reschedule.

**Acceptance Criteria:**

- [ ] Repository function `RescheduleAppointment` opens a pgx transaction.
- [ ] Locks the original appointment row with `SELECT ... FOR UPDATE NOWAIT`.
- [ ] Validates and inserts/updates the new slot inside the same transaction.
- [ ] On any error, transaction is rolled back before returning.
- [ ] `NOWAIT` means concurrent reschedule of the same appointment returns `409` immediately without blocking.

---

---

## EPIC-04 — Patient Appointment Portal

**Canvas alignment:** `Manage Healthcare Provider Core Operations` › `Manage Patient Engagement` › **Manage Patient Appointment View**

---

### FEATURE-04.1 — Patient Appointment History

---

#### US-04.1.1 — View Patient's Upcoming Appointments (Bonus)

**As a** clinic admin,
**I want to** see all upcoming appointments for a specific patient sorted by date,
**So that** I can provide accurate information and send reminders.

**Endpoint:** `GET /patients/{id}/appointments` *(protected)*

**Query Parameters:** `?include_past=false` (default) | `?include_past=true`

**Acceptance Criteria:**

- [ ] Returns `200 OK` with array of appointment objects sorted by `start_time` ascending.
- [ ] Default: only `status = "BOOKED"` AND `start_time > now`.
- [ ] `include_past=true`: returns all appointments for the patient regardless of status or date.
- [ ] Each object includes: `id`, `doctor_id`, `doctor_name`, `start_time`, `end_time`, `status`.
- [ ] No upcoming appointments → `200 OK` with `[]`.
- [ ] Patient not found → `404`: `"Patient with id '{id}' not found."`.
- [ ] Requires `clinic_admin` JWT.

---

---

## EPIC-05 — System Integrity & Validation Engine

**Canvas alignment:** `Manage Digital Security` + `Manage Digital IT` › **Manage API Integrity**

---

### FEATURE-05.1 — Consistent Error Response Schema (Go)

---

#### US-05.1.1 — Standardised JSON Error Responses

**As an** API consumer or Next.js frontend,
**I want to** receive consistently structured error responses,
**So that** I can write generic error-handling logic without parsing endpoint-specific formats.

**Acceptance Criteria:**

- [ ] All error responses: `{ "error": { "code": "MACHINE_CODE", "message": "Human-readable message" } }`.
- [ ] `internal/errors/errors.go` defines `APIError` struct and helpers: `NotFound()`, `Conflict()`, `Validation()`, `Unauthorized()`, `Internal()`.
- [ ] Chi middleware catches unhandled panics and returns `500` with `{ "error": { "code": "INTERNAL_ERROR", "message": "An unexpected error occurred." } }` — no stack traces in response body.
- [ ] HTTP codes: `200`, `201`, `401`, `404`, `409`, `422`, `503`.

---

### FEATURE-05.2 — Test Coverage

---

#### US-05.2.1 — Unit Tests for Slot Service (Go)

**Acceptance Criteria:**

- [ ] `internal/services/slot_service_test.go` covers: full working day with no bookings, partial bookings excluded, fully booked day returns `[]`, no working hours for weekday returns `[]`, past slots excluded, slots within 1 hour excluded (bonus).
- [ ] Tests use `time.Time` injection — no real clock calls inside `GenerateSlots`.
- [ ] Run with `go test ./internal/services/...` — zero external dependencies.

---

#### US-05.2.2 — Integration Tests for API Endpoints (Go)

**Acceptance Criteria:**

- [ ] `httptest.NewRecorder()` used for handler-level tests — no live server needed.
- [ ] Test DB uses a separate Supabase project or local PostgreSQL via `docker-compose.test.yml`.
- [ ] Tests cover full booking lifecycle: create → availability reduced → cancel → availability restored → reschedule → new slot taken, old slot released.
- [ ] Concurrent booking test: 10 goroutines book the same slot simultaneously; exactly 1 succeeds.
- [ ] Run with `go test ./...` — all pass in CI.
- [ ] Coverage target: ≥ 80% on `internal/services` and `internal/handlers`.

---

---

## Section 3 — Next.js Admin UI

> **Tech stack:** Next.js 14 (App Router) · TypeScript · Tailwind CSS · shadcn/ui · Supabase JS SDK v2
>
> **Auth:** Supabase Auth (email/password) — session stored in cookies via `@supabase/ssr`
>
> **API calls:** All data mutations and reads go through the Go API with the Supabase JWT attached as `Authorization: Bearer`
>
> **Deployment:** Vercel (recommended) or Render Static Site

---

## EPIC-06 — Admin Authentication & Access Control

**Canvas alignment:** `Manage Digital Security` › `Manage Identity & Access` › **Manage Admin Authentication**

---

### FEATURE-06.1 — Admin Login & Session Management

---

#### US-06.1.1 — Admin Login Page

**As a** clinic administrator,
**I want to** log in with my email and password,
**So that** I can securely access the clinic management dashboard.

**Route:** `/login`

**Acceptance Criteria:**

- [ ] Login page renders at `/login` with email and password fields and a submit button.
- [ ] On successful login, Supabase Auth returns a session (JWT + refresh token) stored in an HTTP-only cookie via `@supabase/ssr`.
- [ ] User is redirected to `/doctors` (dashboard home) on success.
- [ ] On failed login (wrong credentials), display inline error: `"Invalid email or password."`.
- [ ] On network error, display: `"Unable to connect. Please try again."`.
- [ ] Login form is accessible (keyboard-navigable, ARIA labels on inputs).
- [ ] Page is unauthenticated — redirects to `/doctors` if already logged in.

---

#### US-06.1.2 — Protected Route Middleware

**As a** system,
**I want to** redirect unauthenticated users to `/login` when they attempt to access any dashboard route,
**So that** the admin UI is never accessible without a valid session.

**Acceptance Criteria:**

- [ ] Next.js `middleware.ts` checks for a valid Supabase session on all routes under `/(dashboard)`.
- [ ] Unauthenticated request to any dashboard page is redirected to `/login?redirect=<original_path>`.
- [ ] After login, the user is sent to the originally requested path.
- [ ] Expired sessions trigger re-authentication (Supabase SDK handles token refresh; on failure, redirect to `/login`).

---

#### US-06.1.3 — Admin Logout

**Acceptance Criteria:**

- [ ] A "Sign out" button is visible in the sidebar/nav on all dashboard pages.
- [ ] Clicking it calls `supabase.auth.signOut()`, clears the session cookie, and redirects to `/login`.
- [ ] After sign-out, pressing the browser back button does not return to the dashboard (session is gone).

---

---

## EPIC-07 — Doctor & Schedule Management UI

**Canvas alignment:** `Manage Healthcare Provider Core Operations` › `Manage Clinical Workforce` › **Manage Doctor Schedule**

---

### FEATURE-07.1 — Doctor List & Profile Management

---

#### US-07.1.1 — View All Doctors

**As a** clinic admin,
**I want to** see a table of all doctors,
**So that** I have a quick overview of the clinic's practitioners.

**Route:** `/doctors`

**Acceptance Criteria:**

- [ ] Page fetches `GET /doctors` from the Go API and renders a table with columns: Name, Specialisation, Configured Days, Actions.
- [ ] "Configured Days" shows which weekdays have working hours set (e.g. "Mon, Wed, Fri").
- [ ] Each row has "Edit" and "View Schedule" action buttons.
- [ ] An "Add Doctor" button links to `/doctors/new`.
- [ ] Loading state: skeleton rows shown while fetching.
- [ ] Empty state: "No doctors configured yet. Add your first doctor." with a CTA button.
- [ ] Error state: "Failed to load doctors. Please try again." with a retry button.

---

#### US-07.1.2 — Add a New Doctor

**Route:** `/doctors/new`

**Acceptance Criteria:**

- [ ] Form fields: Name (required), Specialisation (required).
- [ ] On submit, calls `POST /doctors` with the JWT in the header.
- [ ] On success, redirects to `/doctors/{id}` (the new doctor's schedule page) with a toast notification: "Doctor added successfully."
- [ ] Inline validation: required fields highlighted before submission.
- [ ] Server-side validation errors displayed inline below the relevant field.
- [ ] Cancel button returns to `/doctors` without saving.

---

#### US-07.1.3 — Edit a Doctor Profile

**Route:** `/doctors/{id}/edit`

**Acceptance Criteria:**

- [ ] Form pre-populated with existing `name` and `specialisation`.
- [ ] On submit, calls `PUT /doctors/{id}`.
- [ ] On success, redirects to `/doctors/{id}` with toast: "Doctor updated."
- [ ] If the doctor no longer exists (404 from API), display an error page: "Doctor not found."

---

### FEATURE-07.2 — Working Hours Management UI

---

#### US-07.2.1 — View and Edit a Doctor's Working Hours

**As a** clinic admin,
**I want to** set a doctor's working hours per weekday using a visual schedule editor,
**So that** availability is configured accurately without manually editing data.

**Route:** `/doctors/{id}`

**Acceptance Criteria:**

- [ ] Page displays the doctor's name, specialisation, and a 7-day weekly schedule grid.
- [ ] Each day shows a toggle (enabled/disabled) and, when enabled, a start-time and end-time picker (30-minute increments only).
- [ ] Changes are saved via a single "Save Schedule" button that calls `PUT /doctors/{id}/working-hours`.
- [ ] On success, toast: "Working hours saved."
- [ ] On validation error (e.g. start after end), inline error displayed next to the offending day.
- [ ] Unsaved changes trigger a "You have unsaved changes. Leave?" browser confirmation on navigation away.
- [ ] Time pickers only show options on 30-minute boundaries (`:00` and `:30`).

---

---

## EPIC-08 — Booking Management UI

**Canvas alignment:** `Manage Healthcare Provider Core Operations` › `Manage Patient Access` › **Manage Appointment Scheduling + Lifecycle**

---

### FEATURE-08.1 — Appointments List & Filtering

---

#### US-08.1.1 — View All Appointments

**As a** clinic admin,
**I want to** see all appointments in a filterable table,
**So that** I can monitor the clinic's booking load and status at a glance.

**Route:** `/appointments`

**Acceptance Criteria:**

- [ ] Table columns: Date & Time, Doctor, Patient, Status, Actions.
- [ ] Filter controls: Doctor (dropdown), Date range (date picker), Status (`BOOKED` / `CANCELLED` / `ALL`).
- [ ] Default view: upcoming `BOOKED` appointments sorted by date ascending.
- [ ] Status badge: `BOOKED` (green), `CANCELLED` (red), `RESCHEDULED` (amber).
- [ ] Pagination: 25 rows per page with Next/Previous controls.
- [ ] Each row: "Cancel" and "Reschedule" action buttons (disabled if status is `CANCELLED`).
- [ ] Loading, empty, and error states handled.

---

### FEATURE-08.2 — Book an Appointment

---

#### US-08.2.1 — Admin Books an Appointment for a Patient

**As a** clinic admin,
**I want to** book an appointment by selecting a doctor, date, available slot, and patient,
**So that** a patient who calls or walks in can be scheduled immediately.

**Route:** `/appointments/new`

**Acceptance Criteria:**

- [ ] Step 1: Select doctor (dropdown from `GET /doctors`).
- [ ] Step 2: Select date (date picker; past dates disabled).
- [ ] Step 3: Slot grid rendered from `GET /doctors/{id}/availability?date=...` — available slots shown as clickable buttons; taken/past slots greyed out.
- [ ] Step 4: Select or search patient (typeahead from `GET /patients`).
- [ ] Confirm step: summary of doctor, date, time, patient — "Confirm Booking" button.
- [ ] On success: redirects to `/appointments` with toast: "Appointment booked for {patient name} with {doctor name} on {date} at {time}."
- [ ] Slot grid auto-refreshes if admin idles for > 60 seconds before confirming (to show newly taken slots).
- [ ] All validation errors from the API displayed as inline or toast messages.

---

### FEATURE-08.3 — Cancel an Appointment

---

#### US-08.3.1 — Cancel via Modal

**As a** clinic admin,
**I want to** cancel an appointment from the appointments table with a reason,
**So that** the patient's slot is freed without leaving the main appointments view.

**Acceptance Criteria:**

- [ ] Clicking "Cancel" on an appointment row opens a modal: "Cancel Appointment" with a reason textarea and "Confirm Cancellation" / "Dismiss" buttons.
- [ ] "Confirm Cancellation" disabled until reason field has at least 1 non-whitespace character.
- [ ] On confirm, calls `PATCH /appointments/{id}/cancel`; modal closes on success; row status badge updates to `CANCELLED`; toast: "Appointment cancelled."
- [ ] On API error, modal stays open and displays the error message.
- [ ] Already-cancelled appointments: "Cancel" button is disabled with tooltip "Already cancelled."

---

### FEATURE-08.4 — Reschedule an Appointment

---

#### US-08.4.1 — Reschedule via Slide-over Panel

**As a** clinic admin,
**I want to** move an appointment to a new slot without leaving the appointments view,
**So that** rescheduling is quick and low-friction.

**Acceptance Criteria:**

- [ ] Clicking "Reschedule" opens a slide-over panel (not a full page navigation).
- [ ] Panel shows current appointment details (read-only) and a date picker + slot grid for the same doctor.
- [ ] Slot grid fetches `GET /doctors/{id}/availability?date=...` for the selected new date.
- [ ] The current slot is shown as "Current" and not selectable as a new slot.
- [ ] On confirm, calls `PATCH /appointments/{id}/reschedule` with `new_start_time`.
- [ ] On success, panel closes; row updates to new date/time; toast: "Appointment rescheduled."
- [ ] On API error (slot taken, cancelled, etc.) the error message is shown inside the panel.

---

---

## EPIC-09 — Patient Management UI

**Canvas alignment:** `Manage Healthcare Provider Core Operations` › `Manage Patient Engagement` › **Manage Patient Records**

---

### FEATURE-09.1 — Patient List & Profile Management

---

#### US-09.1.1 — View All Patients

**Route:** `/patients`

**Acceptance Criteria:**

- [ ] Table: Name, Email, Phone, Upcoming Appointments (count), Actions.
- [ ] Search bar filters by name or email (client-side for ≤ 500 rows; server-side for larger datasets).
- [ ] "Add Patient" button links to `/patients/new`.
- [ ] Each row has a "View" button linking to `/patients/{id}`.

---

#### US-09.1.2 — Add a New Patient

**Route:** `/patients/new`

**Acceptance Criteria:**

- [ ] Form fields: Name (required), Email (required, valid email format), Phone (optional).
- [ ] On submit, calls `POST /patients`.
- [ ] On success: redirect to `/patients/{id}` with toast: "Patient added."
- [ ] Duplicate email returns `409` from API → display inline: `"A patient with this email already exists."`.

---

#### US-09.1.3 — View Patient Profile & Appointment History

**Route:** `/patients/{id}`

**Acceptance Criteria:**

- [ ] Patient details section: name, email, phone with an "Edit" button.
- [ ] Appointments section: fetches `GET /patients/{id}/appointments?include_past=true`.
- [ ] Table: Date, Time, Doctor, Status — sorted by date descending.
- [ ] Toggle switch "Show upcoming only" / "Show all" changes the `include_past` param and re-fetches.
- [ ] "Book Appointment" button links to `/appointments/new?patient_id={id}` (pre-selects the patient in the booking flow).

---

---

## Section 4 — Deployment & CI/CD

---

## EPIC-10 — Supabase Infrastructure Setup

**Canvas alignment:** `Manage Digital IT` › `Manage Infrastructure` › **Manage Database Infrastructure**

---

### FEATURE-10.1 — Supabase Project Configuration

---

#### US-10.1.1 — Provision Supabase Project

**As a** developer,
**I want to** provision a Supabase project with the correct schema and RLS policies,
**So that** the production database is ready for the Go API and Next.js UI to connect.

**Acceptance Criteria:**

- [ ] A Supabase project exists (free tier or Pro) with a PostgreSQL database.
- [ ] Schema migrations in `migrations/` are applied via `supabase db push` (or `supabase migration up`).
- [ ] Tables `doctors`, `working_hours`, `patients`, `appointments` exist with correct columns, types, and constraints.
- [ ] The unique partial index `appointments_no_double_booking` exists and is confirmed in the Supabase dashboard.
- [ ] RLS is enabled on all four tables.
- [ ] RLS policies (Appendix C) are applied and verified.
- [ ] A `clinic_admin` test user is created in Supabase Auth for local testing.

---

#### US-10.1.2 — Supabase Environment Variables

**Acceptance Criteria:**

- [ ] `.env.example` documents: `SUPABASE_URL`, `SUPABASE_ANON_KEY`, `SUPABASE_SERVICE_ROLE_KEY`, `DATABASE_URL` (pooler), `DATABASE_URL_DIRECT` (for migrations), `SUPABASE_JWT_SECRET`.
- [ ] Go API uses `SUPABASE_SERVICE_ROLE_KEY` for DB access (bypasses RLS server-side).
- [ ] Go API uses `SUPABASE_JWT_SECRET` to validate incoming JWTs from admin users.
- [ ] Next.js uses `NEXT_PUBLIC_SUPABASE_URL` and `NEXT_PUBLIC_SUPABASE_ANON_KEY` (public, browser-safe).
- [ ] `SUPABASE_SERVICE_ROLE_KEY` is **never** exposed to the browser or committed to the repository.

---

---

## EPIC-11 — Render Deployment (Go API)

**Canvas alignment:** `Manage Digital IT` › `Manage Infrastructure` › **Manage Application Hosting**

---

### FEATURE-11.1 — Containerised Go API on Render

---

#### US-11.1.1 — Deploy Go API to Render via Docker

**As a** clinic operator and reviewer,
**I want to** access the Go API at a stable public HTTPS URL,
**So that** the Next.js admin UI and any patient-facing clients can connect to it.

**Acceptance Criteria:**

- [ ] A `Dockerfile` exists using a multi-stage build: `golang:1.23-alpine` build stage → `alpine:3.20` runtime stage (minimal image, no Go toolchain in production).
- [ ] Render Web Service is configured with: `Docker` environment, branch `main`, auto-deploy on push.
- [ ] Environment variables (`DATABASE_URL`, `SUPABASE_JWT_SECRET`, etc.) are set in Render's environment variable UI — never in the repository.
- [ ] The API is reachable at `https://<service>.onrender.com`.
- [ ] `GET /health` returns `200 OK`: `{ "status": "ok", "database": "connected" }`.
- [ ] The public URL is documented in `README.md`.

**Dockerfile (structure):**
```dockerfile
# Stage 1: Build
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o clinic-api ./cmd/server

# Stage 2: Runtime
FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/clinic-api .
EXPOSE 8080
CMD ["./clinic-api"]
```

---

#### US-11.1.2 — Local Development Environment

**Acceptance Criteria:**

- [ ] `docker-compose.yml` runs the Go API + a local PostgreSQL instance.
- [ ] `supabase start` (Supabase CLI) can be used as an alternative for full local Supabase (Auth + DB).
- [ ] A `Makefile` provides: `make run` (local), `make test` (`go test ./...`), `make build` (Docker build), `make migrate` (`supabase db push`).
- [ ] `README.md` covers the full local setup in under 10 steps.

---

#### US-11.1.3 — Health Check Endpoint

**Endpoint:** `GET /health` *(public)*

**Acceptance Criteria:**

- [ ] Pings the database with `SELECT 1`.
- [ ] DB connected → `200 OK`: `{ "status": "ok", "database": "connected" }`.
- [ ] DB unreachable → `503 Service Unavailable`: `{ "status": "degraded", "database": "disconnected" }`.
- [ ] Render health check is configured to call `/health` every 30 seconds.
- [ ] No authentication required.

---

---

## EPIC-12 — CI/CD Pipeline

**Canvas alignment:** `Manage Digital IT` › `Manage DevOps` › **Manage Continuous Delivery**

---

### FEATURE-12.1 — CI: Automated Testing on Pull Request

---

#### US-12.1.1 — Run Go Tests on Every PR

**As a** developer,
**I want to** have linting and tests run automatically on every pull request,
**So that** regressions are caught before merge.

**Trigger:** Every pull request targeting `main`

**Pipeline (`.github/workflows/ci.yml`):**

```yaml
name: CI
on:
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_DB: clinic_test
          POSTGRES_USER: postgres
          POSTGRES_PASSWORD: postgres
        ports: ["5432:5432"]
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
          cache: true

      - name: Install dependencies
        run: go mod download

      - name: Run linter (golangci-lint)
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest

      - name: Apply test migrations
        run: |
          go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
          migrate -path ./migrations -database "$TEST_DATABASE_URL" up
        env:
          TEST_DATABASE_URL: postgres://postgres:postgres@localhost:5432/clinic_test?sslmode=disable

      - name: Run tests
        run: go test -race -coverprofile=coverage.out ./...
        env:
          DATABASE_URL: postgres://postgres:postgres@localhost:5432/clinic_test?sslmode=disable
          SUPABASE_JWT_SECRET: test-secret-for-ci

      - name: Check coverage ≥ 80%
        run: |
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | tr -d '%')
          echo "Coverage: $COVERAGE%"
          awk "BEGIN { exit ($COVERAGE < 80) }"
```

**Acceptance Criteria:**

- [ ] Pipeline triggers on every PR to `main`.
- [ ] Linting via `golangci-lint` — fails PR on lint errors.
- [ ] Tests run with `-race` flag to catch data races.
- [ ] Coverage check enforced at ≥ 80% on `internal/services` and `internal/handlers`.
- [ ] PostgreSQL service container used — no external DB dependency in CI.
- [ ] Branch protection on `main`: merge blocked if CI fails.
- [ ] Pipeline completes in under 5 minutes.

---

### FEATURE-12.2 — CD: Automated Deployment on Merge to Main

---

#### US-12.2.1 — Auto-Deploy Go API to Render on Merge

**As a** clinic operator,
**I want to** have the Go API automatically deployed to Render when a PR is merged to `main`,
**So that** production always reflects the latest approved code with no manual steps.

**Trigger:** Push to `main` (after CI passes)

**Pipeline (`.github/workflows/cd.yml`):**

```yaml
name: CD
on:
  push:
    branches: [main]

jobs:
  test:
    uses: ./.github/workflows/ci.yml   # Reuse CI job

  deploy-api:
    needs: [test]
    runs-on: ubuntu-latest
    steps:
      - name: Trigger Render Deploy
        run: |
          curl -X POST "${{ secrets.RENDER_API_DEPLOY_HOOK }}" \
            -H "Content-Type: application/json"

  deploy-admin:
    needs: [test]
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Deploy Next.js to Vercel
        uses: amondnet/vercel-action@v25
        with:
          vercel-token: ${{ secrets.VERCEL_TOKEN }}
          vercel-org-id: ${{ secrets.VERCEL_ORG_ID }}
          vercel-project-id: ${{ secrets.VERCEL_PROJECT_ID }}
          vercel-args: '--prod'
```

**Acceptance Criteria:**

- [ ] CD pipeline triggers only after the CI `test` job passes (`needs: [test]`).
- [ ] Go API deployed to Render via deploy hook (webhook URL stored in GitHub Actions secret `RENDER_API_DEPLOY_HOOK`).
- [ ] Next.js Admin UI deployed to Vercel via `vercel-action`.
- [ ] After deployment, `GET <RENDER_URL>/health` returns `200 OK` within 3 minutes.
- [ ] Deployment logs visible in GitHub Actions run history.
- [ ] All secrets (`RENDER_API_DEPLOY_HOOK`, `VERCEL_TOKEN`, etc.) stored in GitHub repository secrets — never hardcoded.
- [ ] `README.md` documents: the public API URL, the public admin UI URL, the deployment-triggering branch (`main`), and a description of what each pipeline stage does.

---

#### US-12.2.2 — README Deployment Documentation

**Acceptance Criteria (README must include):**

- [ ] **Public API URL** — the live Render URL (e.g. `https://clinic-api.onrender.com`).
- [ ] **Public Admin UI URL** — the live Vercel URL (e.g. `https://clinic-admin.vercel.app`).
- [ ] **Deployment branch** — `main` triggers deployment.
- [ ] **Pipeline description** — CI runs on PR (lint + test + coverage); CD runs on merge to main (deploy API to Render + admin to Vercel).
- [ ] **Local setup** — step-by-step: `git clone`, copy `.env.example` to `.env`, `make run` or `docker compose up`, `make migrate`, `go test ./...`.
- [ ] **Supabase setup** — how to create a project, apply migrations, create the admin user.
- [ ] **Tech stack table** — Go 1.23 · Chi · pgx v5 · Supabase · Next.js 14 · Render · Vercel · GitHub Actions.

---

---

## Appendix A — Entity Relationship Summary

```
doctors (1) ──────────< working_hours (7 max, one per weekday)
doctors (1) ──────────< appointments (many)
patients (1) ─────────< appointments (many)
auth.users (1) ───────? patients (optional: auth_user_id, for future self-service)

appointments.status ∈ { 'BOOKED', 'CANCELLED', 'RESCHEDULED' }

UNIQUE INDEX: (doctor_id, start_time) WHERE status = 'BOOKED'
  → enforces slot exclusivity at DB level, catching races the app layer misses
```

---

## Appendix B — API Contract Summary

| Method | Endpoint | Auth | Description | Success |
|---|---|---|---|---|
| `GET` | `/doctors` | Public | List all doctors | `200` |
| `POST` | `/doctors` | Admin JWT | Create doctor | `201` |
| `GET` | `/doctors/{id}` | Public | Get doctor + working hours | `200` |
| `PUT` | `/doctors/{id}` | Admin JWT | Update doctor profile | `200` |
| `PUT` | `/doctors/{id}/working-hours` | Admin JWT | Set working hours (full replace) | `200` |
| `GET` | `/doctors/{id}/availability?date=` | Public | Available slots on date | `200` |
| `POST` | `/appointments` | Admin JWT | Book a slot | `201` |
| `PATCH` | `/appointments/{id}/cancel` | Admin JWT | Cancel with reason | `200` |
| `PATCH` | `/appointments/{id}/reschedule` | Admin JWT | Move to new slot | `200` |
| `GET` | `/patients` | Admin JWT | List all patients | `200` |
| `POST` | `/patients` | Admin JWT | Create patient | `201` |
| `GET` | `/patients/{id}` | Admin JWT | Get patient details | `200` |
| `GET` | `/patients/{id}/appointments` | Admin JWT | Patient's appointments | `200` |
| `GET` | `/health` | Public | Health check | `200` |

**HTTP Status Code Semantics:**

| Code | Meaning |
|---|---|
| `200` | Success (read / update) |
| `201` | Resource created |
| `401` | Missing or invalid JWT |
| `404` | Resource not found |
| `409` | State conflict (slot taken / already cancelled) |
| `422` | Validation failure |
| `503` | Database unreachable (health check) |

---

## Appendix C — RLS Policy Summary

```sql
-- Enable RLS on all tables
ALTER TABLE doctors ENABLE ROW LEVEL SECURITY;
ALTER TABLE working_hours ENABLE ROW LEVEL SECURITY;
ALTER TABLE patients ENABLE ROW LEVEL SECURITY;
ALTER TABLE appointments ENABLE ROW LEVEL SECURITY;

-- doctors: public read, admin write
CREATE POLICY "doctors_public_read" ON doctors
  FOR SELECT USING (true);

CREATE POLICY "doctors_admin_write" ON doctors
  FOR ALL USING (
    auth.jwt() ->> 'role' = 'clinic_admin'
  );

-- working_hours: same as doctors
CREATE POLICY "working_hours_public_read" ON working_hours
  FOR SELECT USING (true);

CREATE POLICY "working_hours_admin_write" ON working_hours
  FOR ALL USING (
    auth.jwt() ->> 'role' = 'clinic_admin'
  );

-- patients: admin only (future: patient reads own row)
CREATE POLICY "patients_admin_all" ON patients
  FOR ALL USING (
    auth.jwt() ->> 'role' = 'clinic_admin'
  );

-- appointments: admin only (future: patient reads/cancels own)
CREATE POLICY "appointments_admin_all" ON appointments
  FOR ALL USING (
    auth.jwt() ->> 'role' = 'clinic_admin'
  );
```

> **Note:** The Go API connects via the Supabase **service role key** which bypasses RLS. These policies protect against direct Supabase client calls (e.g. from a compromised browser session) and provide the security foundation for future patient self-service without requiring API changes.

---

## Appendix D — Capability Canvas Traceability Matrix

| Epic | Canvas Domain | Canvas SubDomain | Canvas Capability |
|---|---|---|---|
| EPIC-01 Doctor Schedule Mgmt | Manage Healthcare Provider Core Operations | Manage Clinical Workforce | Manage Doctor Schedule |
| EPIC-02 Appointment Booking | Manage Healthcare Provider Core Operations | Manage Patient Access | Manage Appointment Scheduling |
| EPIC-03 Appointment Lifecycle | Manage Healthcare Provider Core Operations | Manage Patient Access | Manage Appointment Lifecycle |
| EPIC-04 Patient Portal | Manage Healthcare Provider Core Operations | Manage Patient Engagement | Manage Patient Appointment View |
| EPIC-05 System Integrity | Manage Digital Security + Manage Digital IT | Manage API Integrity | Manage Validation Engine |
| EPIC-06 Admin Auth (Next.js) | Manage Digital Security | Manage Identity & Access | Manage Admin Authentication |
| EPIC-07 Doctor Mgmt UI | Manage Healthcare Provider Core Operations | Manage Clinical Workforce | Manage Doctor Schedule |
| EPIC-08 Booking Mgmt UI | Manage Healthcare Provider Core Operations | Manage Patient Access | Manage Appointment Scheduling + Lifecycle |
| EPIC-09 Patient Mgmt UI | Manage Healthcare Provider Core Operations | Manage Patient Engagement | Manage Patient Records |
| EPIC-10 Supabase Infra | Manage Digital IT | Manage Infrastructure | Manage Database Infrastructure |
| EPIC-11 Render Deployment | Manage Digital IT | Manage Infrastructure | Manage Application Hosting |
| EPIC-12 CI/CD Pipeline | Manage Digital IT | Manage DevOps | Manage Continuous Delivery |

> All epics are cross-cuttingly enabled by `Manage Digital Security` (JWT + RLS) and `Manage Digital Intelligence` (future analytics) per the canvas `:ENABLES` relationships.

---

*End of Specification Document v2.0*
