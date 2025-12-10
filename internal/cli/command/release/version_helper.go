package release

import (
	"fmt"
	"strconv"
	"strings"
)

// getPreviousVersion Calcula la versión anterior a partir de una versión dada
// por ej: v1.3.0 -> v1.2.0
func getPreviousVersion(version string) (string, error) {
	version = strings.TrimPrefix(version, "v")

	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("formato de versión no válido: %s", version)
	}

	// Parse parts
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", fmt.Errorf("versión major no válida: %s", parts[0])
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", fmt.Errorf("versión minor no válida: %s", parts[1])
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", fmt.Errorf("versión de parche no válida: %s", parts[2])
	}

	if patch > 0 {
		patch--
	} else if minor > 0 {
		minor--
		patch = 0
	} else if major > 0 {
		major--
		minor = 0
		patch = 0
	} else {
		return "", fmt.Errorf("no se puede decrementar la versión v0.0.0")
	}

	return fmt.Sprintf("v%d.%d.%d", major, minor, patch), nil
}
