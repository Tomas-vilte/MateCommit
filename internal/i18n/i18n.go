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

func NewTranslations(defaultLang string, localesPath string) (*Translations, error) {
	if defaultLang == "" {
		return nil, fmt.Errorf("default language cannot be empty")
	}

	files, err := filepath.Glob(filepath.Join(localesPath, "active.*.toml"))
	if err != nil {
		return nil, fmt.Errorf("no translation files found in directory: locales/")
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no translation files found")
	}

	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

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
