package sqlutil

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Helper functions for converting between Go types and sql.Null* types

// ToSqlInt32 converts a Go int pointer to sql.NullInt32
func ToSqlInt32(val *int) sql.NullInt32 {
	if val == nil {
		return sql.NullInt32{Valid: false}
	}
	return sql.NullInt32{Int32: int32(*val), Valid: true}
}

// ToSqlString converts a Go string pointer to sql.NullString
func ToSqlString(val *string) sql.NullString {
	if val == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *val, Valid: true}
}

// ToSqlInt16 converts a Go int to sql.NullInt16
func ToSqlInt16(val int) sql.NullInt16 {
	return sql.NullInt16{Int16: int16(val), Valid: true}
}

// ToNullUUID converts a Go UUID pointer to uuid.NullUUID
func ToNullUUID(id *uuid.UUID) uuid.NullUUID {
	if id == nil {
		return uuid.NullUUID{Valid: false}
	}
	return uuid.NullUUID{UUID: *id, Valid: true}
}

// FromSqlInt32 converts sql.NullInt32 to Go int pointer
func FromSqlInt32(val sql.NullInt32) *int {
	if !val.Valid {
		return nil
	}
	i := int(val.Int32)
	return &i
}

// FromSqlString converts sql.NullString to Go string with default
func FromSqlString(val sql.NullString, defaultVal string) string {
	if !val.Valid {
		return defaultVal
	}
	return val.String
}

// FromSqlStringPtr converts sql.NullString to Go string pointer
func FromSqlStringPtr(val sql.NullString) *string {
	if !val.Valid {
		return nil
	}
	return &val.String
}

// FromSqlInt16 converts sql.NullInt16 to Go int
func FromSqlInt16(val sql.NullInt16) int {
	if !val.Valid {
		return 0
	}
	return int(val.Int16)
}

// FromNullUUID converts uuid.NullUUID to Go UUID pointer
func FromNullUUID(val uuid.NullUUID) *uuid.UUID {
	if !val.Valid {
		return nil
	}
	return &val.UUID
}

// ToSqlTime converts a Go time pointer to sql.NullTime
func ToSqlTime(val *time.Time) sql.NullTime {
	if val == nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *val, Valid: true}
}

// FromSqlTime converts sql.NullTime to Go time pointer
func FromSqlTime(val sql.NullTime) *time.Time {
	if !val.Valid {
		return nil
	}
	return &val.Time
}

// ToSqlInt32Direct converts a Go int to sql.NullInt32
func ToSqlInt32Direct(val int) sql.NullInt32 {
	return sql.NullInt32{Int32: int32(val), Valid: true}
}