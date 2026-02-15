package config

// setDedupDefaults installs default values for deduplication settings.
func setDedupDefaults() {
	setDefault("dedup.criteria", "message")
	setDefault("dedup.window", "")
}

// registerDedupValidators registers validators for deduplication settings.
func registerDedupValidators() {
	RegisterValidator("dedup.criteria", EnumValidator(map[string]bool{
		"message":        true,
		"message_level":  true,
		"message_source": true,
		"exact":          true,
	}))
	RegisterValidator("dedup.window", DurationValidator(true))
}
