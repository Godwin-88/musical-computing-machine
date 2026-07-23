"use client";

import { useState, useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { getPatient, getPatientAppointments, Patient, Appointment } from "@/lib/api";

export default function PatientDetailPage() {
  const params = useParams();
  const router = useRouter();
  const [patient, setPatient] = useState<Patient | null>(null);
  const [appointments, setAppointments] = useState<Appointment[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showUpcomingOnly, setShowUpcomingOnly] = useState(true);

  useEffect(() => {
    loadData();
  }, [params.id, showUpcomingOnly]);

  const loadData = async () => {
    setLoading(true);
    setError(null);
    try {
      const [patientData, appointmentData] = await Promise.all([
        getPatient(params.id as string),
        getPatientAppointments(params.id as string, !showUpcomingOnly),
      ]);
      setPatient(patientData);
      setAppointments(appointmentData);
    } catch (err: any) {
      setError(err?.error?.message || "Failed to load patient.");
    } finally {
      setLoading(false);
    }
  };

  const getStatusBadge = (status: string) => {
    const colors: Record<string, string> = {
      BOOKED: "bg-green-100 text-green-800",
      CANCELLED: "bg-red-100 text-red-800",
      RESCHEDULED: "bg-yellow-100 text-yellow-800",
    };
    return `inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${colors[status] || "bg-gray-100 text-gray-800"}`;
  };

  if (loading) {
    return (
      <div className="space-y-4">
        <div className="h-8 w-64 animate-pulse rounded bg-gray-100" />
        <div className="h-48 animate-pulse rounded-lg bg-gray-100" />
      </div>
    );
  }

  if (error && !patient) {
    return (
      <div className="rounded-md bg-red-50 p-4">
        <p className="text-sm text-red-700">{error}</p>
        <Link href="/patients" className="mt-2 inline-block text-sm font-medium text-blue-600">Back to Patients</Link>
      </div>
    );
  }

  if (!patient) return null;

  return (
    <div>
      <div className="mb-6">
        <Link href="/patients" className="text-sm text-blue-600 hover:text-blue-900">&larr; Back to Patients</Link>
        <div className="mt-2 flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">{patient.name}</h1>
            <p className="text-sm text-gray-500">{patient.email}</p>
            {patient.phone && <p className="text-sm text-gray-500">{patient.phone}</p>}
          </div>
          <Link
            href={`/appointments/new?patient_id=${patient.id}`}
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-500"
          >
            Book Appointment
          </Link>
        </div>
      </div>

      <div className="mb-4 flex items-center gap-4">
        <h2 className="text-lg font-semibold text-gray-900">Appointments</h2>
        <label className="flex items-center gap-2 text-sm text-gray-600">
          <input
            type="checkbox"
            checked={showUpcomingOnly}
            onChange={(e) => setShowUpcomingOnly(e.target.checked)}
            className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
          />
          Show upcoming only
        </label>
      </div>

      {appointments.length === 0 ? (
        <div className="rounded-md bg-gray-50 p-6 text-center">
          <p className="text-gray-500">No appointments found.</p>
        </div>
      ) : (
        <div className="overflow-hidden rounded-lg bg-white shadow">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Date</th>
                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Time</th>
                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Doctor</th>
                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {appointments.map((apt) => (
                <tr key={apt.id}>
                  <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-900">
                    {new Date(apt.start_time).toLocaleDateString()}
                  </td>
                  <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-500">
                    {new Date(apt.start_time).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
                  </td>
                  <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-500">{apt.doctor_name}</td>
                  <td className="whitespace-nowrap px-6 py-4">
                    <span className={getStatusBadge(apt.status)}>{apt.status}</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}