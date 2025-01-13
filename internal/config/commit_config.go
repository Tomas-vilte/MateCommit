package config

type (
	CommitLocale struct {
		Lang      string
		Types     map[string]CommitType
		HeaderMsg string
		UsageMsg  string
	}

	CommitType struct {
		Emoji       string
		Description string
		Title       string
		Examples    []string
	}

	CommitConfig struct {
		Locale       CommitLocale
		MaxLength    int
		UseEmoji     bool
		CustomPrompt string
	}
)
