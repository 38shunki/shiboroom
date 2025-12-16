package database

import (
	"database/sql"
	"fmt"
	"real-estate-portal/internal/models"
	"time"

	_ "github.com/lib/pq"
)

type DB struct {
	conn *sql.DB
}

func NewDB(host, port, user, password, dbname string) (*DB, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	conn, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(); err != nil {
		return nil, err
	}

	return &DB{conn: conn}, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

// InitSchema creates the properties table if it doesn't exist
func (db *DB) InitSchema() error {
	// Create table with all fields
	query := `
	CREATE TABLE IF NOT EXISTS properties (
		id VARCHAR(32) PRIMARY KEY,
		detail_url TEXT NOT NULL UNIQUE,
		title TEXT NOT NULL,
		image_url TEXT,

		-- Filter fields
		rent INTEGER,
		floor_plan VARCHAR(20),
		area DECIMAL(10, 2),
		walk_time INTEGER,
		station VARCHAR(100),
		address TEXT,
		building_age INTEGER,
		floor INTEGER,

		fetched_at TIMESTAMP NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	-- Create indexes for filtering
	CREATE INDEX IF NOT EXISTS idx_properties_created_at ON properties(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_properties_rent ON properties(rent);
	CREATE INDEX IF NOT EXISTS idx_properties_floor_plan ON properties(floor_plan);
	CREATE INDEX IF NOT EXISTS idx_properties_walk_time ON properties(walk_time);
	`
	_, err := db.conn.Exec(query)
	return err
}

// SaveProperty saves a property to the database
func (db *DB) SaveProperty(p *models.Property) error {
	query := `
	INSERT INTO properties (
		id, detail_url, title, image_url,
		rent, floor_plan, area, walk_time, station, address, building_age, floor,
		fetched_at, created_at
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	ON CONFLICT (detail_url) DO UPDATE SET
		title = EXCLUDED.title,
		image_url = EXCLUDED.image_url,
		rent = EXCLUDED.rent,
		floor_plan = EXCLUDED.floor_plan,
		area = EXCLUDED.area,
		walk_time = EXCLUDED.walk_time,
		station = EXCLUDED.station,
		address = EXCLUDED.address,
		building_age = EXCLUDED.building_age,
		floor = EXCLUDED.floor,
		fetched_at = EXCLUDED.fetched_at
	`
	_, err := db.conn.Exec(query,
		p.ID, p.DetailURL, p.Title, p.ImageURL,
		p.Rent, p.FloorPlan, p.Area, p.WalkTime, p.Station, p.Address, p.BuildingAge, p.Floor,
		p.FetchedAt, time.Now())
	return err
}

// GetAllProperties retrieves all properties from the database
func (db *DB) GetAllProperties() ([]models.Property, error) {
	query := `
		SELECT id, detail_url, title, image_url,
			   rent, floor_plan, area, walk_time, station, address, building_age, floor,
			   fetched_at, created_at
		FROM properties
		ORDER BY created_at DESC
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var properties []models.Property
	for rows.Next() {
		var p models.Property
		err := rows.Scan(
			&p.ID, &p.DetailURL, &p.Title, &p.ImageURL,
			&p.Rent, &p.FloorPlan, &p.Area, &p.WalkTime, &p.Station, &p.Address, &p.BuildingAge, &p.Floor,
			&p.FetchedAt, &p.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		properties = append(properties, p)
	}

	return properties, nil
}

// GetPropertyByID retrieves a property by ID
func (db *DB) GetPropertyByID(id string) (*models.Property, error) {
	query := `
		SELECT id, detail_url, title, image_url,
			   rent, floor_plan, area, walk_time, station, address, building_age, floor,
			   fetched_at, created_at
		FROM properties
		WHERE id = $1
	`

	var p models.Property
	err := db.conn.QueryRow(query, id).Scan(
		&p.ID, &p.DetailURL, &p.Title, &p.ImageURL,
		&p.Rent, &p.FloorPlan, &p.Area, &p.WalkTime, &p.Station, &p.Address, &p.BuildingAge, &p.Floor,
		&p.FetchedAt, &p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &p, nil
}
