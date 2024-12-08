INSERT INTO chair_positions SELECT chair_locations.chair_id as chiar_id, chair_locations.latitude as latitude, chair_locations.longitude as longitude, t.total_distance as total_distance, t2.created_at as created_at, t.total_distance_updated_at as updated_at
FROM (SELECT chair_id,
           SUM(IFNULL(distance, 0)) AS total_distance,
           MAX(created_at)          AS total_distance_updated_at
    FROM (SELECT chair_id,
                 created_at,
                 ABS(latitude - LAG(latitude) OVER (PARTITION BY chair_id ORDER BY created_at)) +
                 ABS(longitude - LAG(longitude) OVER (PARTITION BY chair_id ORDER BY created_at)) AS distance
          FROM chair_locations) tmp
    GROUP BY chair_id) as t
INNER JOIN chair_locations ON t.chair_id = chair_locations.chair_id AND chair_locations.created_at = t.total_distance_updated_at
INNER JOIN (SELECT chair_id, MIN(created_at) as created_at FROM chair_locations GROUP BY chair_id) as t2 ON chair_locations.chair_id = t2.chair_id;
