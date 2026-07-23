"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { createDoctor } from "@/lib/api";

export default function NewDoctorPage() {
  const [name, setName] = useState("");
  const [specialisation, setSpecialisation] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const router = useRouter();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);

    try {
      const doctor = await createDoctor({ name, specialisation });
      router.push(`/doctors/${doctor.id}`);
    } catch (err: any) {
      setError(err?.error?.message || "Failed to create doctor.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Add Doctor</h1>
        <p className="mt-1 text-sm text-gray-500">Create a new doctor profile</p>
      </div>

      <form onSubmit={handleSubmit} className="max-w-md space-y-6">
        <div>
          <label htmlFor="name" className="block text-sm font-medium text-gray-700">
            Name
          </label>
          <input
            id="name"
            type="text"
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-blue-500 focus:outline-none focus:ring-blue-500"
            placeholder="Dr. Jane Smith"
          />
        </div>

        <div>
          <label htmlFor="specialisation" className="block text-sm font-medium text-gray-700">
            Specialisation
          </label>
          <input
            id="specialisation"
            type="text"
            required
            value={specialisation}
            onChange={(e) => setSpecialisation(e.target.value)}
            className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-blue-500 focus:outline-none focus:ring-blue-500"
            placeholder="General Practice"
          />
        </div>

        {error && (
          <div className="rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</div>
        )}

        <div className="flex items-center gap-4">
          <button
            type="submit"
            disabled={loading}
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-500 disabled:opacity-50"
          >
            {loading ? "Saving..." : "Save"}
          </button>
          <Link
            href="/doctors"
            className="text-sm font-medium text-gray-600 hover:text-gray-900"
          >
            Cancel
          </Link>
        </div>
      </form>
    </div>
  );
}