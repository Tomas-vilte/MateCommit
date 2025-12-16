package completion_helper

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

// DefaultFlagComplete imprime todos los flags del comando actual para facilitar la finalización del shell.
// Esto se utiliza para garantizar que se sugieran flags incluso cuando la finalización predeterminada de urfave/cli pueda fallar.
func DefaultFlagComplete(_ context.Context, cmd *cli.Command) {
	for _, f := range cmd.Flags {
		for _, name := range f.Names() {
			if len(name) == 1 {
				fmt.Println("-" + name)
			} else {
				fmt.Println("--" + name)
			}
		}
	}
}
