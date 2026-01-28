-- 1. Seed Roles (Essential System Data)
-- Using OR IGNORE to prevent duplicates on 'name' UNIQUE constraint
INSERT OR IGNORE INTO roles (name, description) VALUES
-- System
('super_admin', 'Full system access and organization management'),
('org_admin', 'Administrative access to a specific organization'),
-- Academic
('academic_coordinator', 'Manages courses, sections, and academic terms'),
('teacher', 'Manages assigned courses and students'),
('class_teacher', 'Primary teacher responsible for a section'),
('teaching_assistant', 'Assists teachers with academic activities'),
('student', 'Access to enrolled courses and personal metrics'),
-- Finance
('accountant', 'Manages invoices, payments, and financial reports'),
('fee_manager', 'Handles fee setup, concessions, and collections'),
-- Operations
('receptionist', 'Manages admissions, inquiries, and front desk tasks'),
('admissions_officer', 'Handles student onboarding and enrollment'),
('attendance_manager', 'Manages attendance records'),
-- External
('parent', 'Access to child academic progress'),
('guardian', 'Limited student-related access'),
-- Support
('it_support', 'Technical and system support');

-- 2. Seed Academic Terms (Initial Setup)
-- Set one term as active by default
INSERT OR IGNORE INTO academic_terms (name, start_date, end_date, is_active) VALUES 
('Academic Year 2025-2026', '2025-06-01', '2026-04-30', 1);

-- 3. Seed Primary Organization
-- Note: 'type' must match your CHECK constraint ('school' or 'institute')
INSERT INTO organizations (name, type, code) 
SELECT 'Everwise Crest Institute', 'institute', 'ECI-001'
WHERE NOT EXISTS (SELECT 1 FROM organizations WHERE code = 'ECI-001');

-- 4. Seed Courses for the specific Organization
-- We use a subquery to find the ID of Everwise Crest Institute 
-- This makes the script portable even if the ID isn't 1.
INSERT OR IGNORE INTO courses (organization_id, name, description)
SELECT id, 'Tamil Made Easy', 'Introductory course for learning Tamil language basics'
FROM organizations WHERE code = 'ECI-001';

INSERT OR IGNORE INTO courses (organization_id, name, description)
SELECT id, 'Game Development', 'Building 2D and 3D games using modern engines'
FROM organizations WHERE code = 'ECI-001';

INSERT OR IGNORE INTO courses (organization_id, name, description)
SELECT id, 'Chess', 'Strategic thinking and tournament-level chess tactics'
FROM organizations WHERE code = 'ECI-001';

INSERT OR IGNORE INTO courses (organization_id, name, description)
SELECT id, 'Phonics', 'Foundational reading skills and phonetic awareness'
FROM organizations WHERE code = 'ECI-001';

INSERT OR IGNORE INTO courses (organization_id, name, description)
SELECT id, 'Web Development', 'Full-stack web applications with HTML, CSS, and JS'
FROM organizations WHERE code = 'ECI-001';

INSERT OR IGNORE INTO courses (organization_id, name, description)
SELECT id, 'Foundations of Coding & Logic', 'Introduction to computational thinking and algorithms'
FROM organizations WHERE code = 'ECI-001';
