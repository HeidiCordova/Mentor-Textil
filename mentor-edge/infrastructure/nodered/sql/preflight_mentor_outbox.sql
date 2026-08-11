-- Read-only checks. Every duplicate query must return zero rows.

SELECT VERSION() AS version, DATABASE() AS database_name;

SHOW INDEX FROM mqtt_lecturas;

SELECT device, time, COUNT(*) AS duplicates
FROM mqtt_lecturas
GROUP BY device, time
HAVING COUNT(*) > 1
ORDER BY duplicates DESC
LIMIT 20;

SELECT device, COUNT(*) AS snapshot_rows
FROM mqtt_snapshot
GROUP BY device
HAVING COUNT(*) > 1
ORDER BY snapshot_rows DESC;

SELECT
    device,
    SUM(status = 0) AS mqtt_pending,
    SUM(restful = 0) AS rest_pending,
    MIN(CASE WHEN status = 0 THEN time END) AS oldest_mqtt_pending,
    MAX(time) AS newest_reading
FROM mqtt_lecturas
GROUP BY device
ORDER BY device;
