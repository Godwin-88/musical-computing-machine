import { createBrowserSupabaseClient } from "./supabase";

export interface ApiError {
  error: {
    code: string;
    message: string;
  };
}

export interface Doctor {
  id: string;
  name: string;
  specialisation: string;
  slot_duration_minutes: number;
  working_hours?: WorkingHours[];
  created_at?: string;
  updated_at?: string;
}

export interface WorkingHours {
  id?: string;
  doctor_id?: string;
  day_of_week: number;
  start_time: string;
  end_time: string;
}

export interface Patient {
  id: string;
  name: string;
  email: string;
  phone?: string;
  auth_user_id?: string;
  created_at: string;
}

export interface Appointment {
  id: string;
  doctor_id: string;
  doctor_name: string;
  patient_id: string;
  patient_name: string;
  start_time: string;
  end_time: string;
  status: "BOOKED" | "CANCELLED" | "RESCHEDULED";
  cancellation_reason?: string;
  cancelled_at?: string;
  created_at: string;
  updated_at: string;
}

export interface Slot {
  start: string;
  end: string;
}

export interface AvailabilityResponse {
  doctor_id: string;
  date: string;
  available_slots: Slot[];
}

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

async function getAccessToken(): Promise<string | null> {
  const supabase = createBrowserSupabaseClient();
  const {
    data: { session },
  } = await supabase.auth.getSession();
  return session?.access_token || null;
}

async function apiFetch<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const token = await getAccessToken();
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as Record<string, string>),
  };

  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const response = await fetch(`${API_URL}${path}`, {
    ...options,
    headers,
  });

  if (!response.ok) {
    const errorBody = await response.json().catch(() => null);
    if (errorBody?.error) {
      throw errorBody as ApiError;
    }
    throw {
      error: {
        code: "UNKNOWN_ERROR",
        message: `Request failed with status ${response.status}`,
      },
    } as ApiError;
  }

  return response.json();
}

// Doctor API
export async function listDoctors(): Promise<Doctor[]> {
  return apiFetch<Doctor[]>("/doctors");
}

export async function getDoctor(id: string): Promise<Doctor> {
  return apiFetch<Doctor>(`/doctors/${id}`);
}

export async function createDoctor(
  data: Pick<Doctor, "name" | "specialisation">
): Promise<Doctor> {
  return apiFetch<Doctor>("/doctors", {
    method: "POST",
    body: JSON.stringify({ ...data, slot_duration_minutes: 30 }),
  });
}

export async function updateDoctor(
  id: string,
  data: Pick<Doctor, "name" | "specialisation">
): Promise<Doctor> {
  return apiFetch<Doctor>(`/doctors/${id}`, {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

export async function setWorkingHours(
  doctorId: string,
  workingHours: WorkingHours[]
): Promise<WorkingHours[]> {
  return apiFetch<WorkingHours[]>(`/doctors/${doctorId}/working-hours`, {
    method: "PUT",
    body: JSON.stringify({ working_hours: workingHours }),
  });
}

export async function getAvailability(
  doctorId: string,
  date: string
): Promise<AvailabilityResponse> {
  return apiFetch<AvailabilityResponse>(
    `/doctors/${doctorId}/availability?date=${date}`
  );
}

// Appointment API
export async function bookAppointment(data: {
  doctor_id: string;
  patient_id: string;
  start_time: string;
}): Promise<Appointment> {
  return apiFetch<Appointment>("/appointments", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function cancelAppointment(
  id: string,
  reason: string
): Promise<Appointment> {
  return apiFetch<Appointment>(`/appointments/${id}/cancel`, {
    method: "PATCH",
    body: JSON.stringify({ reason }),
  });
}

export async function rescheduleAppointment(
  id: string,
  newStartTime: string
): Promise<Appointment> {
  return apiFetch<Appointment>(`/appointments/${id}/reschedule`, {
    method: "PATCH",
    body: JSON.stringify({ new_start_time: newStartTime }),
  });
}

export async function listAppointments(params?: {
  doctor_id?: string;
  status?: string;
}): Promise<Appointment[]> {
  const query = new URLSearchParams();
  if (params?.doctor_id) query.set("doctor_id", params.doctor_id);
  if (params?.status) query.set("status", params.status);
  const qs = query.toString();
  return apiFetch<Appointment[]>(`/appointments${qs ? `?${qs}` : ""}`);
}

// Patient API
export async function listPatients(search?: string): Promise<Patient[]> {
  const query = search ? `?search=${encodeURIComponent(search)}` : "";
  return apiFetch<Patient[]>(`/patients${query}`);
}

export async function getPatient(id: string): Promise<Patient> {
  return apiFetch<Patient>(`/patients/${id}`);
}

export async function createPatient(data: {
  name: string;
  email: string;
  phone?: string;
}): Promise<Patient> {
  return apiFetch<Patient>("/patients", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function getPatientAppointments(
  patientId: string,
  includePast?: boolean
): Promise<Appointment[]> {
  const query = includePast ? "?include_past=true" : "";
  return apiFetch<Appointment[]>(
    `/patients/${patientId}/appointments${query}`
  );
}