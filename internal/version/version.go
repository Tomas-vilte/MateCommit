package version

// Version es la versión actual de MateCommit
// Esta versión debe actualizarse en cada release
const Version = "1.4.0"

// FullVersion retorna la versión con el prefijo v
func FullVersion() string {
	return "v" + Version
}
