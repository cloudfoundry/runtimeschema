package models

import (
	"crypto/md5"
	"flag"
	"fmt"
	"path"
	"strings"
)

type CircusTailorConfig struct {
	*flag.FlagSet

	values map[string]*string

	buildpacksDir  *string
	appDir         *string
	ExecutablePath string

	buildArtifactsCacheDir    *string
	outputDroplet             *string
	outputMetadataDir         *string
	outputBuildArtifactsCache *string
	buildpackOrder            *string
}

const (
	circusTailorAppDirFlag                    = "appDir"
	circusTailorOutputDropletFlag             = "outputDroplet"
	circusTailorOutputMetadataDirFlag         = "outputMetadataDir"
	circusTailorOutputBuildArtifactsCacheFlag = "outputBuildArtifactsCache"
	circusTailorBuildpacksDirFlag             = "buildpacksDir"
	circusTailorBuildArtifactsCacheDirFlag    = "buildArtifactsCacheDir"
	circusTailorBuildpackOrderFlag            = "buildpackOrder"
)

var circusTailorDefaults = map[string]string{
	circusTailorAppDirFlag:                    "/app",
	circusTailorOutputDropletFlag:             "/tmp/droplet",
	circusTailorOutputMetadataDirFlag:         "/tmp/result",
	circusTailorOutputBuildArtifactsCacheFlag: "/tmp/output-cache",
	circusTailorBuildpacksDirFlag:             "/tmp/buildpacks",
	circusTailorBuildArtifactsCacheDirFlag:    "/tmp/cache",
}

func NewCircusTailorConfig(buildpacks []string) CircusTailorConfig {
	flagSet := flag.NewFlagSet("tailor", flag.ExitOnError)

	appDir := flagSet.String(
		circusTailorAppDirFlag,
		circusTailorDefaults[circusTailorAppDirFlag],
		"directory containing raw app bits",
	)

	outputDroplet := flagSet.String(
		circusTailorOutputDropletFlag,
		circusTailorDefaults[circusTailorOutputDropletFlag],
		"file where compressed droplet should be written",
	)

	outputMetadataDir := flagSet.String(
		circusTailorOutputMetadataDirFlag,
		circusTailorDefaults[circusTailorOutputMetadataDirFlag],
		"directory in which to write the app metadata",
	)

	outputBuildArtifactsCache := flagSet.String(
		circusTailorOutputBuildArtifactsCacheFlag,
		circusTailorDefaults[circusTailorOutputBuildArtifactsCacheFlag],
		"file where compressed contents of new cached build artifacts should be written",
	)

	buildpacksDir := flagSet.String(
		circusTailorBuildpacksDirFlag,
		circusTailorDefaults[circusTailorBuildpacksDirFlag],
		"directory containing the buildpacks to try",
	)

	buildArtifactsCacheDir := flagSet.String(
		circusTailorBuildArtifactsCacheDirFlag,
		circusTailorDefaults[circusTailorBuildArtifactsCacheDirFlag],
		"directory where previous cached build artifacts should be extracted",
	)

	buildpackOrder := flagSet.String(
		circusTailorBuildpackOrderFlag,
		strings.Join(buildpacks, ","),
		"comma-separated list of buildpacks, to be tried in order",
	)

	return CircusTailorConfig{
		FlagSet: flagSet,

		ExecutablePath:            "/tmp/circus/tailor",
		appDir:                    appDir,
		outputDroplet:             outputDroplet,
		outputMetadataDir:         outputMetadataDir,
		outputBuildArtifactsCache: outputBuildArtifactsCache,
		buildpacksDir:             buildpacksDir,
		buildArtifactsCacheDir:    buildArtifactsCacheDir,
		buildpackOrder:            buildpackOrder,

		values: map[string]*string{
			circusTailorAppDirFlag:                    appDir,
			circusTailorOutputDropletFlag:             outputDroplet,
			circusTailorOutputMetadataDirFlag:         outputMetadataDir,
			circusTailorOutputBuildArtifactsCacheFlag: outputBuildArtifactsCache,
			circusTailorBuildpacksDirFlag:             buildpacksDir,
			circusTailorBuildArtifactsCacheDirFlag:    buildArtifactsCacheDir,
			circusTailorBuildpackOrderFlag:            buildpackOrder,
		},
	}
}

func (s CircusTailorConfig) Path() string {
	return s.ExecutablePath
}

func (s CircusTailorConfig) Args() []string {
	argv := []string{}

	s.FlagSet.VisitAll(func(flag *flag.Flag) {
		argv = append(argv, fmt.Sprintf("-%s=%s", flag.Name, *s.values[flag.Name]))
	})

	return argv
}

func (s CircusTailorConfig) Validate() error {
	var missingFlags []string

	s.FlagSet.VisitAll(func(flag *flag.Flag) {
		schemaFlag, ok := s.values[flag.Name]
		if !ok {
			return
		}

		value := *schemaFlag
		if value == "" {
			missingFlags = append(missingFlags, "-"+flag.Name)
		}
	})

	if len(missingFlags) > 0 {
		return fmt.Errorf("missing flags: %s", strings.Join(missingFlags, ", "))
	}

	return nil
}

func (s CircusTailorConfig) AppDir() string {
	return *s.appDir
}

func (s CircusTailorConfig) BuildpackPath(buildpackName string) string {
	return path.Join(s.BuildpacksDir(), fmt.Sprintf("%x", md5.Sum([]byte(buildpackName))))
}

func (s CircusTailorConfig) BuildpackOrder() []string {
	return strings.Split(*s.buildpackOrder, ",")
}

func (s CircusTailorConfig) BuildpacksDir() string {
	return *s.buildpacksDir
}

func (s CircusTailorConfig) BuildArtifactsCacheDir() string {
	return *s.buildArtifactsCacheDir
}

func (s CircusTailorConfig) OutputDroplet() string {
	return *s.outputDroplet
}

func (s CircusTailorConfig) OutputMetadataDir() string {
	return *s.outputMetadataDir
}

func (s CircusTailorConfig) OutputMetadataPath() string {
	return path.Join(s.OutputMetadataDir(), "result.json")
}

func (s CircusTailorConfig) OutputBuildArtifactsCache() string {
	return *s.outputBuildArtifactsCache
}
