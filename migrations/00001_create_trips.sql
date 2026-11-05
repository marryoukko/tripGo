CREATE TABLE trips (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    driver_id UUID NOT NULL,
    start_lat DOUBLE PRECISION NOT NULL,
    start_lon DOUBLE PRECISION NOT NULL,
    end_lat DOUBLE PRECISION NOT NULL,
    end_lon DOUBLE PRECISION NOT NULL,
    price BIGINT NOT NULL CHECK (price >= 0),
    status TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ,
    last_position_at TIMESTAMPTZ
);

DROP TABLE trips;