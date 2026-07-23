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