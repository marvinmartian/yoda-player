// internal/db/db.go
package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3" // Assuming you are using SQLite as an example
)

// DB represents the database connection
type DB struct {
	conn *sql.DB
}

// NewDB creates a new database connection
func NewDB(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	return &DB{conn: conn}, nil
}

// Close closes the database connection
func (d *DB) Close() error {
	return d.conn.Close()
}

// CreateTable creates a sample table in the database
func (d *DB) CreateTable() error {
	_, err := d.conn.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			age INTEGER
		);
	`)
	return err
}

// InsertUser inserts a new user into the database
func (d *DB) InsertUser(name string, age int) error {
	_, err := d.conn.Exec("INSERT INTO users (name, age) VALUES (?, ?)", name, age)
	return err
}

// GetAllUsers retrieves all users from the database
func (d *DB) GetAllUsers() ([]string, error) {
	rows, err := d.conn.Query("SELECT name FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		users = append(users, name)
	}

	return users, nil
}
