-- DB role-ууд ба өгөгдлийн сан. Superuser-ээр нэг удаа ажиллуулна:
--   psql -v owner_pw='...' -v app_pw='...' -v admin_pw='...' -v auth_pw='...' -f deploy/01-roles.sql
--
-- Зарчим (docs/01-lessons.md #3): апп хэзээ ч superuser/owner-оор
-- холбогдохгүй. nexus_owner нь зөвхөн миграцад; nexus_app-д RLS бүрэн
-- үйлчилнэ; nexus_admin нь nexus_platform гишүүн тул RLS бодлогууд
-- (pg_has_role) платформ гэж танина.

CREATE ROLE nexus_platform NOLOGIN;
CREATE ROLE nexus_owner LOGIN PASSWORD :'owner_pw';
CREATE ROLE nexus_app LOGIN PASSWORD :'app_pw';
CREATE ROLE nexus_admin LOGIN PASSWORD :'admin_pw';
-- nexus_auth — зөвхөн pre-auth SECURITY DEFINER функцууд (session, signup,
-- email хайлт, handover). Хүснэгтэд шууд хүрэхгүй; апп/модулийн SQL энэ
-- pool-д хүрдэггүй тул SQLi ч session үүсгэж чадахгүй (00010).
CREATE ROLE nexus_auth LOGIN PASSWORD :'auth_pw';
GRANT nexus_platform TO nexus_admin;
-- Definer функцууд owner-ийн эрхээр pg_has_role шалгахад owner өөрөө
-- platform байх шаардлагагүй — гэхдээ RLS-гүй унших нь owner-д угаас бий.

CREATE DATABASE nexus_mini OWNER nexus_owner;
-- Temp хүснэгтээр definer функцийн relation дарахаас (00010 H1).
REVOKE TEMPORARY ON DATABASE nexus_mini FROM PUBLIC;
