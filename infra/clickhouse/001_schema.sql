-- Имитационный справочник инфраструктуры (остановки). Не смешивать с default/таблицами analytics.
CREATE DATABASE IF NOT EXISTS its_infra_sim;

CREATE TABLE IF NOT EXISTS its_infra_sim.municipalities
(
    `municipality_id` LowCardinality(String) COMMENT 'Код города для API и телеметрии (msk, spb, …)',
    `name_ru` String COMMENT 'Наименование для UI',
    `center_lat` Float64 COMMENT 'Центр карты WGS84',
    `center_lon` Float64 COMMENT 'Центр карты WGS84',
    `default_zoom` UInt8 COMMENT 'Масштаб Leaflet по умолчанию',
    `updated_at` DateTime64(3, 'UTC') COMMENT 'Время обновления строки'
)
ENGINE = MergeTree
ORDER BY municipality_id
COMMENT 'Справочник населённых пунктов для map_portal (имитация)';

CREATE TABLE IF NOT EXISTS its_infra_sim.bus_stops
(
    `stop_id` UUID COMMENT 'Стабильный идентификатор остановки',
    `stop_code` LowCardinality(String) COMMENT 'Код в НСИ/внутренний (для связи с маршрутами и телеметрией)',
    `name` String COMMENT 'Полное наименование остановки',
    `name_short` String COMMENT 'Краткое имя для табло',
    `lat` Float64 COMMENT 'Широта WGS84',
    `lon` Float64 COMMENT 'Долгота WGS84',
    `municipality_id` LowCardinality(String) COMMENT 'Идентификатор муниципалитета/зоны моделирования',
    `address_note` String COMMENT 'Условный адрес или ориентир',
    `bearing_deg` Nullable(Float32) COMMENT 'Азимут направления движения маршрута, градусы',
    `road_name` String COMMENT 'Условное имя улицы/дороги',
    `lane_count` UInt8 COMMENT 'Число полос в сторону движения (оценка для симуляции)',
    `bay_type` LowCardinality(String) COMMENT 'inline | lay_by | unknown — заездной карман (см. ГОСТ Р 52766)',
    `distance_to_junction_m` Nullable(Float32) COMMENT 'Расстояние до ближайшего пересечения (тема ГОСТ Р 58653)',
    `has_shelter` UInt8 COMMENT 'Навес/павильон ожидания',
    `has_seating` UInt8 COMMENT 'Места для сидения',
    `has_trash_bin` UInt8 COMMENT 'Урна',
    `has_schedule_info` UInt8 COMMENT 'Информация о расписании (табло/схема)',
    `has_lighting` UInt8 COMMENT 'Освещение',
    `boarding_width_m` Nullable(Float32) COMMENT 'Ширина посадочной зоны, м (ориентир по ГОСТ Р 52766)',
    `platform_surface` LowCardinality(String) COMMENT 'Тип покрытия площадки ожидания',
    `wheelchair_accessible` UInt8 COMMENT 'Доступность для МГН (колясочная доступность)',
    `tactile_guiding` UInt8 COMMENT 'Тактильная навигация',
    `signage_set_id` LowCardinality(String) COMMENT 'Условный код набора знаков (логически ГОСТ Р 58287)',
    `source` LowCardinality(String) COMMENT 'Источник записи, например simulation',
    `gost_reference_note` String COMMENT 'Текстовая отсылка к нормам (не юридическая интерпретация)',
    `valid_from` Date COMMENT 'Начало действия версии справочника',
    `valid_to` Nullable(Date) COMMENT 'Окончание действия; NULL — актуально',
    `updated_at` DateTime64(3, 'UTC') COMMENT 'Время последнего обновления строки'
)
ENGINE = MergeTree
ORDER BY (municipality_id, stop_code, stop_id)
COMMENT 'Остановочные пункты автобуса (имитация, поля согласованы с тематикой ГОСТ Р 52766, 58287, 58653)';

CREATE TABLE IF NOT EXISTS its_infra_sim.bus_stop_routes
(
    `stop_id` UUID COMMENT 'Ссылка на bus_stops.stop_id',
    `route_number` LowCardinality(String) COMMENT 'Номер маршрута',
    `direction` LowCardinality(String) COMMENT 'Направление: forward | backward | loop | unknown',
    `sequence` Nullable(UInt16) COMMENT 'Порядковый номер остановки на маршруте',
    `updated_at` DateTime64(3, 'UTC')
)
ENGINE = MergeTree
ORDER BY (route_number, direction, stop_id)
COMMENT 'Связь остановка — маршрут (имитация)';
