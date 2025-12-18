package ports

import (
	"context"

	"github.com/Tomas-vilte/MateCommit/internal/i18n"
)

// BinaryPackager define la interfaz para construir y empaquetar binarios
type BinaryPackager interface {
	BuildAndPackageAll(ctx context.Context) ([]string, error)
}

// BinaryBuilderFactory define la interfaz para crear nuevas instancias de BinaryPackager.
type BinaryBuilderFactory interface {
	NewBuilder(mainPath, binaryName, version, commit, date, buildDir string, trans *i18n.Translations) BinaryPackager
}
