package config

import "log"

const (
	LangEN = "en"
	LangES = "es"
)

var (
	EnglishConfig = CommitLocale{
		HeaderMsg: "Commit suggestions:",
		UsageMsg:  "To use a suggestion:",
		Lang:      LangEN,
		Types: map[string]CommitType{
			"feat": {
				Emoji:       "✨",
				Title:       "Features",
				Description: "A new feature",
				Examples:    []string{"Add login button", "Implement user authentication"},
			},
			"fix": {
				Emoji:       "🐛",
				Title:       "Bug Fixes",
				Description: "A bug fix",
				Examples:    []string{"Fix login validation", "Fix database connection issue"},
			},
			"docs": {
				Emoji:       "📚",
				Title:       "Documentation",
				Description: "Documentation only changes",
				Examples:    []string{"Update README", "Add API documentation"},
			},
			"style": {
				Emoji:       "💄",
				Title:       "Styles",
				Description: "Changes that do not affect the meaning of the code",
				Examples:    []string{"Fix indentation", "Remove whitespace"},
			},
			"refactor": {
				Emoji:       "♻️",
				Title:       "Code Refactoring",
				Description: "A code change that neither fixes a bug nor adds a feature",
				Examples:    []string{"Refactor authentication logic", "Restructure database queries"},
			},
			"test": {
				Emoji:       "✅",
				Title:       "Tests",
				Description: "Adding missing tests or correcting existing tests",
				Examples:    []string{"Add unit tests for login", "Fix integration tests"},
			},
			"chore": {
				Emoji:       "🔧",
				Title:       "Chores",
				Description: "Changes to the build process or auxiliary tools",
				Examples:    []string{"Update dependencies", "Configure CI pipeline"},
			},
			"perf": {
				Emoji:       "⚡️",
				Title:       "Performance Improvements",
				Description: "A code change that improves performance",
				Examples:    []string{"Optimize database queries", "Improve rendering performance"},
			},
			"ci": {
				Emoji:       "👷",
				Title:       "CI/CD",
				Description: "Changes to CI configuration files and scripts",
				Examples:    []string{"Add GitHub Actions workflow", "Update Jenkins pipeline"},
			},
			"build": {
				Emoji:       "📦",
				Title:       "Builds",
				Description: "Changes that affect the build system or external dependencies",
				Examples:    []string{"Update webpack config", "Add new dependency"},
			},
			"revert": {
				Emoji:       "⏪️",
				Title:       "Reverts",
				Description: "Reverts a previous commit",
				Examples:    []string{"Revert 'Add new feature'", "Rollback database migration"},
			},
		},
	}

	SpanishConfig = CommitLocale{
		Lang:      LangES,
		HeaderMsg: "Sugerencias de commit:",
		UsageMsg:  "Para usar una sugerencia",
		Types: map[string]CommitType{
			"feat": {
				Emoji:       "✨",
				Title:       "Funcionalidades",
				Description: "Una nueva funcionalidad",
				Examples:    []string{"Agregar botón de login", "Meter autenticación de usuarios"},
			},
			"fix": {
				Emoji:       "🐛",
				Title:       "Arreglos",
				Description: "Arreglo de un bug",
				Examples:    []string{"Arreglar validación del login", "Corregir problema de conexión con la base"},
			},
			"docs": {
				Emoji:       "📚",
				Title:       "Documentación",
				Description: "Cambios solo en documentación",
				Examples:    []string{"Actualizar README", "Agregar documentación de la API"},
			},
			"style": {
				Emoji:       "💄",
				Title:       "Estilos",
				Description: "Cambios que no afectan el significado del código",
				Examples:    []string{"Arreglar indentación", "Sacar espacios en blanco"},
			},
			"refactor": {
				Emoji:       "♻️",
				Title:       "Refactorización",
				Description: "Cambio de código que no arregla bugs ni agrega funcionalidades",
				Examples:    []string{"Refactorizar lógica de autenticación", "Reestructurar queries"},
			},
			"test": {
				Emoji:       "✅",
				Title:       "Tests",
				Description: "Agregar tests faltantes o corregir existentes",
				Examples:    []string{"Agregar tests unitarios del login", "Arreglar tests de integración"},
			},
			"chore": {
				Emoji:       "🔧",
				Title:       "Tareas",
				Description: "Cambios en el proceso de build o herramientas auxiliares",
				Examples:    []string{"Actualizar dependencias", "Configurar pipeline de CI"},
			},
			"perf": {
				Emoji:       "⚡️",
				Title:       "Rendimiento",
				Description: "Cambio de código que mejora el rendimiento",
				Examples:    []string{"Optimizar queries", "Mejorar rendimiento del renderizado"},
			},
			"ci": {
				Emoji:       "👷",
				Title:       "CI/CD",
				Description: "Cambios en archivos de configuración y scripts de CI",
				Examples:    []string{"Agregar workflow de GitHub Actions", "Actualizar pipeline de Jenkins"},
			},
			"build": {
				Emoji:       "📦",
				Title:       "Build",
				Description: "Cambios que afectan el sistema de build o dependencias externas",
				Examples:    []string{"Actualizar configuración de webpack", "Agregar nueva dependencia"},
			},
			"revert": {
				Emoji:       "⏪️",
				Title:       "Reverts",
				Description: "Revierte un commit anterior",
				Examples:    []string{"Revertir 'Agregar nueva feature'", "Rollback de migración de base de datos"},
			},
		},
	}
)

func GetLocaleConfig(lang string) CommitLocale {
	switch lang {
	case LangEN:
		return EnglishConfig
	case LangES:
		return SpanishConfig
	default:
		log.Printf("Idioma '%s' no soportado. Usando configuración por defecto (Inglés).", lang)
		return EnglishConfig
	}
}
