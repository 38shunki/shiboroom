-- Normalize floor plan values in existing properties
-- ワンルーム → 1R, etc.

UPDATE properties
SET floor_plan = '1R'
WHERE floor_plan LIKE '%ワンルーム%' OR floor_plan LIKE '%1R%';

UPDATE properties
SET floor_plan = '1K'
WHERE floor_plan LIKE '%1K%';

UPDATE properties
SET floor_plan = '1DK'
WHERE floor_plan LIKE '%1DK%';

UPDATE properties
SET floor_plan = '1LDK'
WHERE floor_plan LIKE '%1LDK%';

UPDATE properties
SET floor_plan = '1SDK'
WHERE floor_plan LIKE '%1SDK%';

UPDATE properties
SET floor_plan = '1SLDK'
WHERE floor_plan LIKE '%1SLDK%';

UPDATE properties
SET floor_plan = '2K'
WHERE floor_plan LIKE '%2K%';

UPDATE properties
SET floor_plan = '2DK'
WHERE floor_plan LIKE '%2DK%';

UPDATE properties
SET floor_plan = '2LDK'
WHERE floor_plan LIKE '%2LDK%';

UPDATE properties
SET floor_plan = '2SDK'
WHERE floor_plan LIKE '%2SDK%';

UPDATE properties
SET floor_plan = '2SLDK'
WHERE floor_plan LIKE '%2SLDK%';

UPDATE properties
SET floor_plan = '3K'
WHERE floor_plan LIKE '%3K%';

UPDATE properties
SET floor_plan = '3DK'
WHERE floor_plan LIKE '%3DK%';

UPDATE properties
SET floor_plan = '3LDK'
WHERE floor_plan LIKE '%3LDK%';

UPDATE properties
SET floor_plan = '3SDK'
WHERE floor_plan LIKE '%3SDK%';

UPDATE properties
SET floor_plan = '3SLDK'
WHERE floor_plan LIKE '%3SLDK%';

UPDATE properties
SET floor_plan = '4K'
WHERE floor_plan LIKE '%4K%';

UPDATE properties
SET floor_plan = '4DK'
WHERE floor_plan LIKE '%4DK%';

UPDATE properties
SET floor_plan = '4LDK'
WHERE floor_plan LIKE '%4LDK%';

-- Normalize building type values
-- マンション → apartment, etc.

UPDATE properties
SET building_type = 'apartment'
WHERE building_type LIKE '%マンション%' OR building_type LIKE '%アパート%';

UPDATE properties
SET building_type = 'house'
WHERE building_type LIKE '%一戸建%' OR building_type LIKE '%戸建%';

UPDATE properties
SET building_type = 'terrace_house'
WHERE building_type LIKE '%テラスハウス%';

UPDATE properties
SET building_type = 'town_house'
WHERE building_type LIKE '%タウンハウス%';

UPDATE properties
SET building_type = 'share_house'
WHERE building_type LIKE '%シェアハウス%';
