package i18n

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"path/filepath"
)

type Translations struct {
	bundle   *i18n.Bundle
	localize *i18n.Localizer
}

func NewTranslations(defaultLang string) (*Translations, error) {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	bundle.MustParseMessageFileBytes([]byte(defaultMessages), "default.en.toml")

	files, err := filepath.Glob("locales/active.*.toml")
	if err != nil {
		return nil, fmt.Errorf("error reading locales: %w", err)
	}

	for _, file := range files {
		if _, err := bundle.LoadMessageFile(file); err != nil {
			return nil, fmt.Errorf("error loading locale file %s: %w", file, err)
		}
	}

	localize := i18n.NewLocalizer(bundle, defaultLang)

	return &Translations{
		bundle:   bundle,
		localize: localize,
	}, nil
}

func (t *Translations) SetLanguage(lang string) error {
	for _, tag := range t.bundle.LanguageTags() {
		if tag.String() == lang {
			t.localize = i18n.NewLocalizer(t.bundle, lang)
			return nil
		}
	}
	return fmt.Errorf("language '%s' not supported", lang)
}

func (t *Translations) GetMessage(messageID string, count int, templateData map[string]interface{}) string {
	localized, err := t.localize.Localize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID: messageID,
		},
		PluralCount:  count,
		TemplateData: templateData,
	})
	if err != nil {
		return "Translation missing: " + messageID
	}
	return localized
}

var defaultMessages = `
	[suggest_command_description]
	description = "Description for the suggest command"
	other = "Analyze your changes and suggest appropriate commit messages"
	
	[no_staged_changes]
	other = "No staged changes to commit.\nUse 'git add' to stage your changes first"
	
	[analyzing_changes]
	other = "Analyzing changes..."
	
	[invalid_suggestions_count]
	other = "Number of suggestions must be between {{.Min}} and {{.Max}}"
	
	[current_config]
	other = "Current configuration"
	
	[select_option]
	other = "Select an option:"
	
	[enter_selection]
	other = "Enter your selection:"
	
	[operation_cancelled]
	other = "Operation cancelled"
	
	[commit_created]
	other = "Commit created successfully with message:"
	
	[modified_files_count]
	one = "{{.Count}} file modified"
	other = "{{.Count}} files modified"
	`
