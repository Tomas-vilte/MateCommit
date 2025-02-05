package i18n

import (
	"embed"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"os"
	"path/filepath"
)

//go:embed locales/*
var localesFS embed.FS

type Translations struct {
	bundle   *i18n.Bundle
	localize *i18n.Localizer
}

func NewTranslations(defaultLang string, localesPath string) (*Translations, error) {
	if defaultLang == "" {
		return nil, fmt.Errorf("default language cannot be empty")
	}

	var files []os.DirEntry
	var err error

	// Si localesPath está vacío, usamos el sistema embebido
	if localesPath == "" {
		files, err = readEmbeddedLocales()
	} else {
		files, err = readLocalesFromFileSystem(localesPath)
	}

	if err != nil {
		return nil, err
	}

	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	// Cargar archivos de traducción
	for _, file := range files {
		var data []byte
		if localesPath == "" {
			data, err = localesFS.ReadFile(filepath.Join("locales", file.Name()))
		} else {
			data, err = os.ReadFile(filepath.Join(localesPath, file.Name()))
		}

		if err != nil {
			return nil, fmt.Errorf("error reading file %s: %w", file.Name(), err)
		}

		bundle.MustParseMessageFileBytes(data, file.Name())
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

func readEmbeddedLocales() ([]os.DirEntry, error) {
	return localesFS.ReadDir("locales")
}

func readLocalesFromFileSystem(localesPath string) ([]os.DirEntry, error) {
	return os.ReadDir(localesPath)
}
