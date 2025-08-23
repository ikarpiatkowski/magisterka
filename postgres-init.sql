-- Enable pg_stat_statements extension for query performance monitoring
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

CREATE TABLE IF NOT EXISTS my_data (
    id SERIAL PRIMARY KEY,
    data JSONB
);

-- Dodaj indeks GIN na kolumnie 'data' dla efektywnego przeszukiwania JSONB
CREATE INDEX IF NOT EXISTS idx_gin_my_data_data ON my_data USING GIN (data jsonb_path_ops);
