ALTER TABLE image_tasks
  ALTER COLUMN created_at   TYPE timestamptz USING created_at   AT TIME ZONE 'Asia/Shanghai',
  ALTER COLUMN started_at   TYPE timestamptz USING started_at   AT TIME ZONE 'Asia/Shanghai',
  ALTER COLUMN completed_at TYPE timestamptz USING completed_at AT TIME ZONE 'Asia/Shanghai';

ALTER TABLE image_generations
  ALTER COLUMN created_at TYPE timestamptz USING created_at AT TIME ZONE 'Asia/Shanghai';
