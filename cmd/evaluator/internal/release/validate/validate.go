package validate

import (
	"fmt"
	"os"

	"github.com/laurentsimon/slsa-policy/cli/evaluator/internal/utils"
	"github.com/laurentsimon/slsa-policy/pkg/release"
	"github.com/laurentsimon/slsa-policy/pkg/utils/iterator/files_reader"
)

func usage(cli string) {
	msg := "" +
		"Usage: %s release validate orgPath projectsPath\n" +
		"\n" +
		"Example:\n" +
		"%s release validate ./path/to/policy/org ./path/to/policy/projects\n" +
		"\n"
	fmt.Fprintf(os.Stderr, msg, cli, cli)
	os.Exit(1)
}

type PolicyValidator struct{}

func (v *PolicyValidator) ValidatePackage(pkg release.ValidationPackage) error {
	return utils.ValidatePolicyPackage(pkg.Name, pkg.Environment.AnyOf)
}

func Run(cli string, args []string) error {
	// We need 2 paths:
	// 1. Path to org policy
	// 2. Path to project policy.
	if len(args) != 2 {
		usage(cli)
	}
	orgPath := args[0]
	projectsPath, err := utils.ReadFiles(args[1], orgPath)
	if err != nil {
		return err
	}
	// Create a policy. This will validate the files.
	projectsReader := files_reader.FromPaths(projectsPath)
	organizationReader, err := os.Open(orgPath)
	_, err = release.PolicyNew(organizationReader, projectsReader, &utils.PackageHelper{}, release.SetValidator(&PolicyValidator{}))
	if err != nil {
		return err
	}
	return nil
}
