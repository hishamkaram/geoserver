-- Minimal seed used by the integration test TestPublishPostgisLayer (and any
-- other suites that publish a postgis-backed layer). The table is named
-- "lbldyt" to match the historical fixture from the v1.0 era; the GeoServer
-- PublishPostgisLayer endpoint requires the table to exist with at least one
-- column besides the geometry column or it 400s with
--   "Trying to create new feature type inside the store, but no attributes
--    were specified"
--
-- This file lives in /docker-entrypoint-initdb.d/ inside the postgis
-- container; the postgres image runs *.sql files alphabetically on first
-- boot of an empty data volume. To re-run after a schema change, recreate
-- the volume: `docker compose down -v && docker compose up -d --wait`.

\connect gis;

CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE IF NOT EXISTS public.lbldyt (
    gid    SERIAL PRIMARY KEY,
    name   TEXT,
    label  TEXT,
    geom   geometry(Geometry, 4326)
);

CREATE INDEX IF NOT EXISTS lbldyt_geom_gist ON public.lbldyt USING GIST (geom);

INSERT INTO public.lbldyt (name, label, geom) VALUES
    ('alpha', 'first label',  ST_SetSRID(ST_MakePoint(  0.0,  0.0), 4326)),
    ('beta',  'second label', ST_SetSRID(ST_MakePoint( 10.0, 20.0), 4326)),
    ('gamma', 'third label',  ST_SetSRID(ST_MakePoint(-30.0, 40.0), 4326))
ON CONFLICT DO NOTHING;

GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO golang;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO golang;
