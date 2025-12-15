package errors

import "fmt"

// ConfigError representa un error de configuración
type ConfigError struct {
	Field   string
	Message string
	Err     error
}

func (e *ConfigError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("config error [%s]: %s: %v", e.Field, e.Message, e.Err)
	}
	return fmt.Sprintf("config error [%s]: %s", e.Field, e.Message)
}

func (e *ConfigError) Unwrap() error {
	return e.Err
}

// NewConfigError crea un nuevo error de configuración
func NewConfigError(field, message string, err error) *ConfigError {
	return &ConfigError{
		Field:   field,
		Message: message,
		Err:     err,
	}
}

// AIProviderNotFoundError indica que un proveedor de IA no fue encontrado
type AIProviderNotFoundError struct {
	Provider string
}

func (e *AIProviderNotFoundError) Error() string {
	return fmt.Sprintf("Proveedor de IA '%s' no encontrado en el registro", e.Provider)
}

// NewAIProviderNotFoundError crea un nuevo error de proveedor no encontrado
func NewAIProviderNotFoundError(provider string) *AIProviderNotFoundError {
	return &AIProviderNotFoundError{Provider: provider}
}

// AIProviderNotConfiguredError indica que un proveedor de IA no está configurado
type AIProviderNotConfiguredError struct {
	Provider string
	Reason   string
}

func (e *AIProviderNotConfiguredError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("Proveedor IA '%s' no configurado: %s", e.Provider, e.Reason)
	}
	return fmt.Sprintf("Proveedor IA '%s' no configurado: %s", e.Provider, e.Reason)
}

// NewAIProviderNotConfiguredError crea un nuevo error de proveedor no configurado
func NewAIProviderNotConfiguredError(provider, reason string) *AIProviderNotConfiguredError {
	return &AIProviderNotConfiguredError{
		Provider: provider,
		Reason:   reason,
	}
}

// VCSConfigNotFoundError indica que la configuración de VCS no fue encontrada
type VCSConfigNotFoundError struct {
	Provider string
}

func (e *VCSConfigNotFoundError) Error() string {
	return fmt.Sprintf("Configuracion VCS para proveedor '%s' no encontrado", e.Provider)
}

// NewVCSConfigNotFoundError crea un nuevo error de config VCS no encontrada
func NewVCSConfigNotFoundError(provider string) *VCSConfigNotFoundError {
	return &VCSConfigNotFoundError{Provider: provider}
}

// VCSProviderNotConfiguredError indica que un proveedor VCS no está configurado
type VCSProviderNotConfiguredError struct {
	Provider string
}

func (e *VCSProviderNotConfiguredError) Error() string {
	return fmt.Sprintf("Proveedor VCS '%s' detectado pero no configurado", e.Provider)
}

// NewVCSProviderNotConfiguredError crea un nuevo error de proveedor VCS no configurado
func NewVCSProviderNotConfiguredError(provider string) *VCSProviderNotConfiguredError {
	return &VCSProviderNotConfiguredError{Provider: provider}
}

// VCSProviderNotSupportedError indica que un proveedor VCS no es soportado
type VCSProviderNotSupportedError struct {
	Provider string
}

func (e *VCSProviderNotSupportedError) Error() string {
	return fmt.Sprintf("Proveedor VCS '%s' no es soportado", e.Provider)
}

// NewVCSProviderNotSupportedError crea un nuevo error de proveedor no soportado
func NewVCSProviderNotSupportedError(provider string) *VCSProviderNotSupportedError {
	return &VCSProviderNotSupportedError{Provider: provider}
}
