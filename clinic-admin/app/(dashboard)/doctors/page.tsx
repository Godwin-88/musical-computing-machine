"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { listDoctors, Doctor } from "@/lib/api";

export default function DoctorsPage() {
  const [doctors, setDoctors] = useState<Doctor[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadDoctors();
  }, []);

  const loadDoctors = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await listDoctors();
      setDoctors(data);
    } catch (err: any) {
      setError(err?.error?.message || "Failed to load doctors.");
    } finally {
      setLoading(false);
    }
  };

  const getConfiguredDays = (doctor: Doctor): string => {
    if (!doctor.working_hours || doctor.working_hours.length === 0) return "None";
    const days = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];
    return doctor.working_hours
      .map((wh) => days[wh.day_of_week])
      .join(", ");
  };

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">Doctors</h1>
        <Link
          href="/doctors/new"
          className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-500"
        >
          Add Doctor
        </Link>
      </div>

      {loading && (
        <div className="space-y-4">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-16 animate-pulse rounded-lg bg-gray-100" />
          ))}
        </div>
      )}

      {error && (
        <div className="rounded-md bg-red-50 p-4">
          <p className="text-sm text-red-700">{error}</p>
          <button
            onClick={loadDoctors}
            className="mt-2 text-sm font-medium text-red-700 underline"
          >
            Try again
          </button>
        </div>
      )}

      {!loading && !error && doctors.length === 0 && (
        <div className="rounded-md bg-gray-50 p-8 text-center">
          <p className="text-gray-500">No doctors configured yet. Add your first doctor.</p>
          <Link
            href="/doctors/new"
            className="mt-4 inline-block rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white"
          >
            Add Doctor
          </Link>
        </div>
      )}

      {!loading && !error && doctors.length > 0 && (
        <div className="overflow-hidden rounded-lg bg-white shadow">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Name</th>
                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Specialisation</th>
                <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Configured Days</th>
                <th className="px-6 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {doctors.map((doctor) => (
                <tr key={doctor.id}>
                  <td className="whitespace-nowrap px-6 py-4 text-sm font-medium text-gray-900">{doctor.name}</td>
                  <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-500">{doctor.specialisation}</td>
                  <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-500">{getConfiguredDays(doctor)}</td>
                  <td className="whitespace-nowrap px-6 py-4 text-right text-sm">
                    <Link href={`/doctors/${doctor.id}`} className="text-blue-600 hover:text-blue-900 mr-4">
                      View Schedule
                    </Link>
                    <Link href={`/doctors/${doctor.id}`} className="text-gray-600 hover:text-gray-900">
                      Edit
                    </Link>
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