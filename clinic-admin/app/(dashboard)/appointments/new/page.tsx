"use client";

import { Suspense, useState, useEffect } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { listDoctors, getAvailability, listPatients, bookAppointment, Doctor, Slot, Patient } from "@/lib/api";

type Step = "doctor" | "date" | "slot" | "patient" | "confirm";

function NewAppointmentForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const preselectedPatientId = searchParams.get("patient_id");

  const [step, setStep] = useState<Step>("doctor");
  const [doctors, setDoctors] = useState<Doctor[]>([]);
  const [patients, setPatients] = useState<Patient[]>([]);
  const [selectedDoctorId, setSelectedDoctorId] = useState<string>("");
  const [selectedDate, setSelectedDate] = useState<string>("");
  const [slots, setSlots] = useState<Slot[]>([]);
  const [selectedSlot, setSelectedSlot] = useState<string>("");
  const [selectedPatientId, setSelectedPatientId] = useState<string>(preselectedPatientId || "");
  const [patientSearch, setPatientSearch] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    listDoctors().then(setDoctors).catch(() => {});
    listPatients().then(setPatients).catch(() => {});
  }, []);

  useEffect(() => {
    if (selectedDoctorId && selectedDate) {
      setLoading(true);
      setError(null);
      const tomorrow = new Date();
      tomorrow.setDate(tomorrow.getDate() + 1);
      const dateStr = selectedDate || tomorrow.toISOString().split("T")[0];

      getAvailability(selectedDoctorId, dateStr)
        .then((resp) => setSlots(resp.available_slots))
        .catch((err: any) => setError(err?.error?.message || "Failed to load slots"))
        .finally(() => setLoading(false));
    }
  }, [selectedDoctorId, selectedDate]);

  const filteredPatients = patients.filter(
    (p) =>
      p.name.toLowerCase().includes(patientSearch.toLowerCase()) ||
      p.email.toLowerCase().includes(patientSearch.toLowerCase())
  );

  const handleConfirm = async () => {
    setLoading(true);
    setError(null);
    try {
      await bookAppointment({
        doctor_id: selectedDoctorId,
        patient_id: selectedPatientId,
        start_time: selectedSlot,
      });
      router.push("/appointments");
    } catch (err: any) {
      setError(err?.error?.message || "Failed to book appointment.");
    } finally {
      setLoading(false);
    }
  };

  const selectedPatient = patients.find((p) => p.id === selectedPatientId);
  const selectedDoctor = doctors.find((d) => d.id === selectedDoctorId);

  return (
    <div className="max-w-2xl mx-auto">
      <h1 className="text-2xl font-bold text-gray-900 mb-6">Book Appointment</h1>

      {/* Step indicator */}
      <div className="mb-8 flex items-center gap-2 text-sm">
        {(["doctor", "date", "slot", "patient", "confirm"] as const).map((s, i) => (
          <span key={s} className={`flex items-center gap-2 ${step === s ? "text-blue-600 font-semibold" : "text-gray-400"}`}>
            <span className={`flex h-6 w-6 items-center justify-center rounded-full text-xs ${
              step === s ? "bg-blue-600 text-white" : "bg-gray-200 text-gray-500"
            }`}>
              {i + 1}
            </span>
            {s.charAt(0).toUpperCase() + s.slice(1)}
            {i < 4 && <span className="text-gray-300">→</span>}
          </span>
        ))}
      </div>

      {error && (
        <div className="mb-4 rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</div>
      )}

      {/* Step 1: Select Doctor */}
      {step === "doctor" && (
        <div className="rounded-lg bg-white p-6 shadow">
          <h2 className="text-lg font-semibold mb-4">Select Doctor</h2>
          <div className="space-y-2">
            {doctors.map((d) => (
              <button
                key={d.id}
                onClick={() => { setSelectedDoctorId(d.id); setStep("date"); }}
                className="w-full rounded-md border p-3 text-left hover:border-blue-500 hover:bg-blue-50"
              >
                <p className="font-medium">{d.name}</p>
                <p className="text-sm text-gray-500">{d.specialisation}</p>
              </button>
            ))}
          </div>
        </div>
      )}

      {/* Step 2: Select Date */}
      {step === "date" && (
        <div className="rounded-lg bg-white p-6 shadow">
          <h2 className="text-lg font-semibold mb-4">Select Date</h2>
          <input
            type="date"
            value={selectedDate}
            onChange={(e) => setSelectedDate(e.target.value)}
            min={new Date().toISOString().split("T")[0]}
            className="w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none"
          />
          <button
            onClick={() => selectedDate && setStep("slot")}
            disabled={!selectedDate}
            className="mt-4 rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-500 disabled:opacity-50"
          >
            Next
          </button>
        </div>
      )}

      {/* Step 3: Select Slot */}
      {step === "slot" && (
        <div className="rounded-lg bg-white p-6 shadow">
          <h2 className="text-lg font-semibold mb-4">Select Time Slot</h2>
          {loading && <p className="text-gray-500">Loading slots...</p>}
          {!loading && slots.length === 0 && <p className="text-gray-500">No available slots for this date.</p>}
          {!loading && slots.length > 0 && (
            <div className="grid grid-cols-4 gap-2">
              {slots.map((slot) => {
                const start = new Date(slot.start);
                const timeStr = start.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
                return (
                  <button
                    key={slot.start}
                    onClick={() => { setSelectedSlot(slot.start); setStep("patient"); }}
                    className={`rounded-md border p-2 text-sm ${
                      selectedSlot === slot.start
                        ? "border-blue-500 bg-blue-50 text-blue-700"
                        : "hover:border-blue-500"
                    }`}
                  >
                    {timeStr}
                  </button>
                );
              })}
            </div>
          )}
        </div>
      )}

      {/* Step 4: Select Patient */}
      {step === "patient" && (
        <div className="rounded-lg bg-white p-6 shadow">
          <h2 className="text-lg font-semibold mb-4">Select Patient</h2>
          <input
            type="text"
            placeholder="Search by name or email..."
            value={patientSearch}
            onChange={(e) => setPatientSearch(e.target.value)}
            className="w-full rounded-md border border-gray-300 px-3 py-2 mb-4 focus:border-blue-500 focus:outline-none"
          />
          <div className="space-y-2 max-h-64 overflow-y-auto">
            {filteredPatients.map((p) => (
              <button
                key={p.id}
                onClick={() => { setSelectedPatientId(p.id); setStep("confirm"); }}
                className={`w-full rounded-md border p-3 text-left ${
                  selectedPatientId === p.id
                    ? "border-blue-500 bg-blue-50"
                    : "hover:border-blue-500"
                }`}
              >
                <p className="font-medium">{p.name}</p>
                <p className="text-sm text-gray-500">{p.email}</p>
              </button>
            ))}
          </div>
        </div>
      )}

      {/* Step 5: Confirm */}
      {step === "confirm" && (
        <div className="rounded-lg bg-white p-6 shadow">
          <h2 className="text-lg font-semibold mb-4">Confirm Booking</h2>
          <div className="space-y-3">
            <div className="flex justify-between">
              <span className="text-gray-500">Doctor:</span>
              <span className="font-medium">{selectedDoctor?.name}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-500">Date:</span>
              <span className="font-medium">{new Date(selectedSlot).toLocaleDateString()}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-500">Time:</span>
              <span className="font-medium">{new Date(selectedSlot).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-500">Patient:</span>
              <span className="font-medium">{selectedPatient?.name} ({selectedPatient?.email})</span>
            </div>
          </div>
          <div className="mt-6 flex gap-3">
            <button
              onClick={handleConfirm}
              disabled={loading}
              className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-500 disabled:opacity-50"
            >
              {loading ? "Booking..." : "Confirm Booking"}
            </button>
            <button
              onClick={() => setStep("doctor")}
              className="rounded-md bg-gray-100 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-200"
            >
              Start Over
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

export default function NewAppointmentPage() {
  return (
    <Suspense fallback={<div className="max-w-2xl mx-auto p-8 text-center text-gray-500">Loading...</div>}>
      <NewAppointmentForm />
    </Suspense>
  );
}