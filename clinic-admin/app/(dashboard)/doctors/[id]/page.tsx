"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { getDoctor, setWorkingHours, Doctor, WorkingHours } from "@/lib/api";

const DAY_NAMES = ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"];

const TIME_OPTIONS: string[] = [];
for (let h = 0; h < 24; h++) {
  TIME_OPTIONS.push(`${String(h).padStart(2, "0")}:00`);
  TIME_OPTIONS.push(`${String(h).padStart(2, "0")}:30`);
}

export default function DoctorSchedulePage() {
  const params = useParams();
  const router = useRouter();
  const [doctor, setDoctor] = useState<Doctor | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saveMessage, setSaveMessage] = useState<string | null>(null);
  const [hasUnsaved, setHasUnsaved] = useState(false);

  // Local working hours state
  const [workingHours, setWorkingHours] = useState<Record<number, { enabled: boolean; start: string; end: string }>>({});

  const initialRef = useRef<string>("");

  useEffect(() => {
    loadDoctor();
  }, [params.id]);

  useEffect(() => {
    const serialized = JSON.stringify(workingHours);
    if (initialRef.current && initialRef.current !== serialized) {
      setHasUnsaved(true);
    }
  }, [workingHours]);

  const loadDoctor = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await getDoctor(params.id as string);
      setDoctor(data);

      const whMap: Record<number, { enabled: boolean; start: string; end: string }> = {};
      for (let i = 0; i < 7; i++) {
        whMap[i] = { enabled: false, start: "09:00", end: "17:00" };
      }

      if (data.working_hours) {
        for (const wh of data.working_hours) {
          whMap[wh.day_of_week] = {
            enabled: true,
            start: wh.start_time.substring(0, 5),
            end: wh.end_time.substring(0, 5),
          };
        }
      }

      setWorkingHours(whMap);
      initialRef.current = JSON.stringify(whMap);
    } catch (err: any) {
      setError(err?.error?.message || "Failed to load doctor.");
    } finally {
      setLoading(false);
    }
  };

  const handleToggleDay = (day: number) => {
    setWorkingHours((prev) => ({
      ...prev,
      [day]: { ...prev[day], enabled: !prev[day].enabled },
    }));
  };

  const handleTimeChange = (day: number, field: "start" | "end", value: string) => {
    setWorkingHours((prev) => ({
      ...prev,
      [day]: { ...prev[day], [field]: value },
    }));
  };

  const handleSave = async () => {
    setSaving(true);
    setSaveMessage(null);
    setError(null);

    const hours: WorkingHours[] = [];
    for (let i = 0; i < 7; i++) {
      const wh = workingHours[i];
      if (wh.enabled) {
        hours.push({
          day_of_week: i,
          start_time: wh.start + ":00",
          end_time: wh.end + ":00",
        });
      }
    }

    try {
      await setWorkingHours(params.id as string, hours);
      setSaveMessage("Working hours saved.");
      initialRef.current = JSON.stringify(workingHours);
      setHasUnsaved(false);
    } catch (err: any) {
      setError(err?.error?.message || "Failed to save working hours.");
    } finally {
      setSaving(false);
    }
  };

  // Warn on unsaved changes
  useEffect(() => {
    const handler = (e: BeforeUnloadEvent) => {
      if (hasUnsaved) {
        e.preventDefault();
      }
    };
    window.addEventListener("beforeunload", handler);
    return () => window.removeEventListener("beforeunload", handler);
  }, [hasUnsaved]);

  if (loading) {
    return (
      <div className="space-y-4">
        <div className="h-8 w-64 animate-pulse rounded bg-gray-100" />
        <div className="h-96 animate-pulse rounded-lg bg-gray-100" />
      </div>
    );
  }

  if (error && !doctor) {
    return (
      <div className="rounded-md bg-red-50 p-4">
        <p className="text-sm text-red-700">{error}</p>
        <Link href="/doctors" className="mt-2 inline-block text-sm font-medium text-blue-600">
          Back to Doctors
        </Link>
      </div>
    );
  }

  if (!doctor) return null;

  return (
    <div>
      <div className="mb-6">
        <Link href="/doctors" className="text-sm text-blue-600 hover:text-blue-900">
          &larr; Back to Doctors
        </Link>
        <h1 className="mt-2 text-2xl font-bold text-gray-900">{doctor.name}</h1>
        <p className="text-sm text-gray-500">{doctor.specialisation}</p>
        <p className="text-sm text-gray-500">Slot duration: {doctor.slot_duration_minutes} minutes</p>
      </div>

      {saveMessage && (
        <div className="mb-4 rounded-md bg-green-50 p-3 text-sm text-green-700">{saveMessage}</div>
      )}
      {error && (
        <div className="mb-4 rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</div>
      )}

      <div className="overflow-hidden rounded-lg bg-white shadow">
        <div className="px-6 py-4 border-b">
          <h2 className="text-lg font-semibold text-gray-900">Weekly Schedule</h2>
          <p className="text-sm text-gray-500">Set the doctor's available hours for each day of the week</p>
        </div>

        <div className="divide-y divide-gray-200">
          {DAY_NAMES.map((dayName, i) => {
            const wh = workingHours[i] || { enabled: false, start: "09:00", end: "17:00" };
            return (
              <div key={i} className="flex items-center gap-4 px-6 py-4">
                <div className="flex items-center gap-3 w-40">
                  <input
                    type="checkbox"
                    checked={wh.enabled}
                    onChange={() => handleToggleDay(i)}
                    className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                  />
                  <span className={`text-sm font-medium ${wh.enabled ? "text-gray-900" : "text-gray-400"}`}>
                    {dayName}
                  </span>
                </div>

                {wh.enabled && (
                  <div className="flex items-center gap-2">
                    <select
                      value={wh.start}
                      onChange={(e) => handleTimeChange(i, "start", e.target.value)}
                      className="rounded-md border border-gray-300 px-2 py-1 text-sm focus:border-blue-500 focus:outline-none"
                    >
                      {TIME_OPTIONS.map((t) => (
                        <option key={t} value={t}>{t}</option>
                      ))}
                    </select>
                    <span className="text-gray-500">to</span>
                    <select
                      value={wh.end}
                      onChange={(e) => handleTimeChange(i, "end", e.target.value)}
                      className="rounded-md border border-gray-300 px-2 py-1 text-sm focus:border-blue-500 focus:outline-none"
                    >
                      {TIME_OPTIONS.map((t) => (
                        <option key={t} value={t}>{t}</option>
                      ))}
                    </select>
                  </div>
                )}
              </div>
            );
          })}
        </div>

        <div className="border-t px-6 py-4">
          <button
            onClick={handleSave}
            disabled={saving || !hasUnsaved}
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-500 disabled:opacity-50"
          >
            {saving ? "Saving..." : "Save Schedule"}
          </button>
        </div>
      </div>
    </div>
  );
}