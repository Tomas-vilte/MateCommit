package config

import "log"

const (
	LangEN = "en"
	LangES = "es"
)

func GetLocaleConfig(lang string) string {
	switch lang {
	case LangEN:
		return LangEN
	case LangES:
		return LangES
	default:
		log.Printf("Idioma '%s' no soportado. Usando configuración por defecto (Inglés).", lang)
		return LangEN
	}
}
