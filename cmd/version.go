package cmd

import (
	"context"
	"fmt"

	"github.com/optiflow-os/tracelens-cli/pkg/exitcode"
	"github.com/optiflow-os/tracelens-cli/pkg/version"

	"github.com/spf13/viper"
)

func runVersion(_ context.Context, v *viper.Viper) (int, error) {
	if v.GetBool("verbose") {
		fmt.Printf(
			"wakatime-cli\n  Version: %s\n  Commit: %s\n  Built: %s\n  OS/Arch: %s/%s\n",
			version.Version,
			version.Commit,
			version.BuildDate,
			version.OS,
			version.Arch,
		)

		return exitcode.Success, nil
	}

	fmt.Println(version.Version)

	return exitcode.Success, nil
}
