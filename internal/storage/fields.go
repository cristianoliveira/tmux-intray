// Package storage provides the storage interface for tmux-intray.
package storage

// Field indices for the notification schema used in TSV output format:
// id, timestamp, state, session, window, pane, message, pane_created, level, read_timestamp.
// read_timestamp is RFC3339 when read, empty when unread.
const (
	FieldID = iota
	FieldTimestamp
	FieldState
	FieldSession
	FieldWindow
	FieldPane
	FieldMessage
	FieldPaneCreated
	FieldLevel
	FieldReadTimestamp
	NumFields
	MinFields = NumFields - 1
)

// Backward-compatible aliases used internally in storage package.
const (
	fieldID            = FieldID
	fieldTimestamp     = FieldTimestamp
	fieldState         = FieldState
	fieldSession       = FieldSession
	fieldWindow        = FieldWindow
	fieldPane          = FieldPane
	fieldMessage       = FieldMessage
	fieldPaneCreated   = FieldPaneCreated
	fieldLevel         = FieldLevel
	fieldReadTimestamp = FieldReadTimestamp
	numFields          = NumFields
	minFields          = MinFields
)
