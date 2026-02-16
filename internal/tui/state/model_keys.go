package state

// model_keys.go is intentionally minimal.
// All key handling methods have been split into:
// - model_keys_core.go: Main dispatch and routing
// - model_key_handlers.go: Individual key type reactions
// - model_movement.go: Cursor navigation
// - model_actions.go: Notification operations
// - model_settings.go: Settings management
// - messages.go: errorMsg struct and errorMsgAfter function
