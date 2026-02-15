package humans

import (
	"database/sql"
	"time"
)

type HumanStore struct {
	db *sql.DB
}

func NewHumanStore(db *sql.DB) *HumanStore {
	return &HumanStore{
		db: db,
	}
}

func (s *HumanStore) GetHumans() ([]*Human, error) {
	humans := make([]*Human, 0)
	stmt := `SELECT first_name, last_name, date_of_birth, has_allergies, bio FROM humans`
	rows, err := s.db.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		pet, err := scanRowsIntoHuman(rows)
		if err != nil {
			return nil, err
		}
		humans = append(humans, pet)
	}
	return humans, nil
}

func scanRowsIntoHuman(rows *sql.Rows) (*Human, error) {
	Human := new(Human)
	err := rows.Scan(
		&Human.FirstName,
		&Human.LastName,
		&Human.DateOfBirth,
		&Human.HasAllergies,
		&Human.Bio,
	)

	if err != nil {
		return nil, err
	}
	return Human, nil
}

func (s *HumanStore) SeedHumans(db *sql.DB) error {
	humans := []Human{
		{"Alice", "Walker", "1990-03-12", false, "Loves long walks and dogs."},
		{"Bob", "Smith", "1985-07-22", true, "Allergic but determined."},
		{"Charlie", "Johnson", "1992-11-02", false, "Works from home."},
		{"Diana", "Brown", "1988-01-17", false, "Very active lifestyle."},
		{"Ethan", "Davis", "1995-09-30", false, "Enjoys hiking."},
		{"Fiona", "Miller", "1991-04-05", true, "Cat person trying dogs."},
		{"George", "Wilson", "1983-12-11", false, "Has a big yard."},
		{"Hannah", "Moore", "1998-06-19", false, "First-time pet owner."},
		{"Ian", "Taylor", "1987-08-08", false, "Experienced with rescues."},
		{"Julia", "Anderson", "1993-10-25", false, "Looking for a running buddy."},
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		INSERT INTO humans 
		(first_name, last_name, date_of_birth, has_allergies, bio, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()

	for _, h := range humans {
		_, err := stmt.Exec(
			h.FirstName,
			h.LastName,
			h.DateOfBirth,
			boolToInt(h.HasAllergies),
			h.Bio,
			now,
			now,
		)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
