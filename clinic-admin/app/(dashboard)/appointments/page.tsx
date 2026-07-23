"use client";

import { useState, useEffect } from "react";
import { listAppointments, cancelAppointment, Appointment } from "@/lib/api";

export default function AppointmentsPage() {
  const [appointments, setAppointments] = useState<Appointment[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState("");
  const [cancelModal, setCancelModal] = useState<{ id: string; reason: string } | null>(null);
  const [cancelling, setCancelling] = useState(false);

  useEffect(() => {
    loadAppointments();
  }, [statusFilter]);

  const loadAppointments = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await listAppointments(statusFilter ? { status: statusFilter } : undefined);
      setAppointments(data);
    } catch (err: any) {
      setError(err?.error?.message || "Failed to load appointments.");
    } finally {
      setLoading(false);
    }
  };

  const handleCancel = async () => {
    if (!cancelModal) return;
    setCancelling(true);
    setError(null);
    try {
      await cancelAppointment(cancelModal.id, cancelModal.reason);
      setCancelModal(null);
      loadAppointments();
    } catch (err: any) {
      setError(err?.error?.message || "Failed to cancel appointment.");
    } finally {
      setCancelling(false);
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

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">Appointments</h1>
        <div className="flex items-center gap-4">
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="rounded-md border border-gray-300 px-3 py-1.5 text-sm focus:border-blue-500 focus:outline-none"
          >
            <option value="">All Status</option>
            <option value="BOOKED">Booked</option>
            <option value="CANCELLED">Cancelled</option>
            <option value="RESCHEDULED">Rescheduled</option>
          </select>
          <a
            href="/appointments/new"
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-500"
          >
            Book Appointment
          </a>
        </div>
      </div>

      {error && (
        <div className="mb-4 rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</div>
      )}

      {loading && (
        <div className="space-y-4">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-12 animate-pulse rounded-lg bg-gray-100" />
          ))}
        </div>
      )}

      {!loading && !error && appointments.length === 0 && (
        <div className="rounded-md bg-gray-50 p-8 text-center">
          <p className="text-gray-500">No appointments found.</p>
        </div>
      )}

      {!loading && !error && appointments.length > 0 && (
        <div className="overflow-hidden rounded-lg bg-white shadow">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Date & Time</th>
                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Doctor</th>
                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Patient</th>
                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Status</th>
                <th className="px-6 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {appointments.map((apt) => (
                <tr key={apt.id}>
                  <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-900">
                    {new Date(apt.start_time).toLocaleString()}
                  </td>
                  <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-500">{apt.doctor_name}</td>
                  <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-500">{apt.patient_name}</td>
                  <td className="whitespace-nowrap px-6 py-4">
                    <span className={getStatusBadge(apt.status)}>{apt.status}</span>
                  </td>
                  <td className="whitespace-nowrap px-6 py-4 text-right text-sm">
                    {apt.status === "BOOKED" && (
                      <>
                        <button
                          onClick={() => setCancelModal({ id: apt.id, reason: "" })}
                          className="text-red-600 hover:text-red-900 mr-4"
                        >
                          Cancel
                        </button>
                        <button
                          className="text-blue-600 hover:text-blue-900"
                        >
                          Reschedule
                        </button>
                      </>
                    )}
                    {apt.status === "CANCELLED" && (
                      <span className="text-gray-400 text-xs">Already cancelled</span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Cancel Modal */}
      {cancelModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
          <div className="w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Cancel Appointment</h2>
            <textarea
              placeholder="Cancellation reason..."
              value={cancelModal.reason}
              onChange={(e) => setCancelModal({ ...cancelModal, reason: e.target.value })}
              className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
              rows={3}
            />
            <div className="mt-4 flex justify-end gap-3">
              <button
                onClick={() => setCancelModal(null)}
                className="rounded-md bg-gray-100 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-200"
              >
                Dismiss
              </button>
              <button
                onClick={handleCancel}
                disabled={cancelling || !cancelModal.reason.trim()}
                className="rounded-md bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-500 disabled:opacity-50"
              >
                {cancelling ? "Cancelling..." : "Confirm Cancellation"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}