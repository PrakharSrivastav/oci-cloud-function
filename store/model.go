package store

import (
	"database/sql"
)

type ScheduledSteps struct {
	ID          int64          `db:"ID"`
	SchID       int64          `db:"SCH_ID"`
	Seq         int64          `db:"SEQ"`
	Type        string         `db:"TYPE"`
	Description sql.NullString `db:"DESCRIPTION"`
}

func (ss *ScheduledSteps) ToHistory(status, description string) ScheduledHistory {
	return ScheduledHistory{
		SchID:       ss.SchID,
		Seq:         ss.Seq,
		Status:      status,
		Type:        ss.Type,
		Description: sql.NullString{String: description, Valid: true},
	}
}

type ScheduledHistory struct {
	ID          int64          `db:"ID"`
	SchID       int64          `db:"SCH_ID"`
	Seq         int64          `db:"SEQ"`
	Status      string         `db:"STATUS"`
	Type        string         `db:"TYPE"`
	CreatedAt   sql.NullTime   `db:"CREATED_AT"`
	UpdatedAt   sql.NullTime   `db:"UPDATED_AT"`
	Description sql.NullString `db:"DESCRIPTION"`
}
