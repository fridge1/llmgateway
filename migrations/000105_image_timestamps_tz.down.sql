ALTER TABLE image_tasks
  ALTER COLUMN created_at   TYPE timestamp USING created_at   AT TIME ZONE 'Asia/Shanghai',
  ALTER COLUMN started_at   TYPE timestamp USING started_at   AT TIME ZONE 'Asia/Shanghai',
  ALTER COLUMN completed_at TYPE timestamp USING completed_at AT TIME ZONE 'Asia/Shanghai';

ALTER TABLE image_generations
  ALTER COLUMN created_at TYPE timestamp USING created_at AT TIME ZONE 'Asia/Shanghai';
