package completion_helper

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

// DefaultFlagComplete prints all flags of the current command to facilitate shell completion.
// This is used to ensure flags are suggested even when the default urfave/cli completion might fail.
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
