-- Add time_based_rules JSONB column for peak/off-peak pricing multipliers.
-- Format: [{"name":"高峰期","days":[1,2,3,4,5],"start_time":"09:00","end_time":"18:00","multiplier":1.5},...]
-- days: 0=Sunday, 1=Monday, ..., 6=Saturday.
-- start_time/end_time: "HH:MM" in Asia/Shanghai (UTC+8).
-- multiplier: applied to all price fields. 1.0 = no change.
-- When null or empty, billing uses base prices (multiplier=1.0, backward-compatible).
ALTER TABLE model_pricing ADD COLUMN time_based_rules JSONB;
