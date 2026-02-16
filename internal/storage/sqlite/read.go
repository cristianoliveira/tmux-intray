package sqlite

import (
	"context"
	"fmt"

	"github.com/cristianoliveira/tmux-intray/internal/storage/sqlite/sqlcgen"
)

// MarkNotificationRead sets read_timestamp to current UTC time.
func (s *SQLiteStorage) MarkNotificationRead(id string) error {
	return s.markNotificationReadState(id, utcNow())
}

// MarkNotificationUnread clears read_timestamp.
func (s *SQLiteStorage) MarkNotificationUnread(id string) error {
	return s.markNotificationReadState(id, "")
}

// MarkNotificationReadWithTimestamp sets read_timestamp to the provided timestamp.
func (s *SQLiteStorage) MarkNotificationReadWithTimestamp(id, timestamp string) error {
	return s.markNotificationReadState(id, timestamp)
}

// MarkNotificationUnreadWithTimestamp clears read_timestamp (timestamp parameter is ignored, kept for consistency).
func (s *SQLiteStorage) MarkNotificationUnreadWithTimestamp(id, timestamp string) error {
	return s.markNotificationReadState(id, timestamp)
}

func (s *SQLiteStorage) markNotificationReadState(id, readTimestamp string) error {
	idInt, err := parseID(id)
	if err != nil {
		return err
	}

	res, err := s.queries.UpdateReadTimestampByID(context.Background(), sqlcgen.UpdateReadTimestampByIDParams{
		ReadTimestamp: readTimestamp,
		UpdatedAt:     utcNow(),
		ID:            idInt,
	})
	if err != nil {
		return fmt.Errorf("sqlite storage: update read state: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("sqlite storage: read rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("sqlite storage: mark read state: %w: id %s", ErrNotificationNotFound, id)
	}

	return nil
}
