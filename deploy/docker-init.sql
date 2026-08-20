-- Docker Compose-ийн Postgres анх асахад ажиллана (docker-entrypoint-initdb.d).
-- Локал/жишээ орчны нууц үгүүд — production-д deploy/01-roles.sql-ийг өөрийн
-- нууц үгтэй ажиллуулна.
CREATE ROLE nexus_platform NOLOGIN;
CREATE ROLE nexus_owner LOGIN PASSWORD 'nexus-dev';
CREATE ROLE nexus_app LOGIN PASSWORD 'nexus-dev';
CREATE ROLE nexus_admin LOGIN PASSWORD 'nexus-dev';
GRANT nexus_platform TO nexus_admin;
CREATE DATABASE nexus_mini OWNER nexus_owner;
