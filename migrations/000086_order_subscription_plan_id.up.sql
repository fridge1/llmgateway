ALTER TABLE orders ADD COLUMN subscription_plan_id INTEGER REFERENCES subscription_plans(id);
