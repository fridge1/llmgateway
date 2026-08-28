-- Demote any back-office roles introduced by this release back to 'user'.
UPDATE users SET role = 'user' WHERE role IN ('support', 'finance', 'ops');
