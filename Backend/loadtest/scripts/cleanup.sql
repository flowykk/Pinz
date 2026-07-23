-- Сносит всех пользователей-сидов и каскадно — все их сессии, passkey, desired_places.
-- Запускать против БД auth-сервиса loadtest-стенда (LOADTEST_DB_URL).
-- Trips/pins/media и notification — отдельные БД; чистятся отдельным DELETE по тем же
-- email-префиксам через grpc-клиента или вручную, поскольку trip-service не хранит email.
DELETE FROM users WHERE is_test = true;
