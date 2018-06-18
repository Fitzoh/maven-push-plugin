package main

import (
	"code.cloudfoundry.org/cli/plugin"
	"flag"
	"fmt"
	"github.com/go-yaml/yaml"
	"io/ioutil"
	"os"
	"strings"
)

type MavenPushPlugin struct {
}

type Manifest struct {
	Applications []Application `yaml:"applications"`
}

type Application struct {
	Name        string      `yaml:"name"`
	MavenConfig MavenConfig `yaml:"maven"`
}

func ParseManifest(f string) (Manifest, error) {
	raw, err := ioutil.ReadFile(f)
	if err != nil {
		return Manifest{}, fmt.Errorf("failed to read manifest file %s, %+v", f, err)
	}
	var manifest Manifest
	err = yaml.Unmarshal(raw, &manifest)
	if err != nil {
		return Manifest{}, fmt.Errorf("failed to umarshall manifest file %s, %+v", f, err)
	}
	if numApplications := len(manifest.Applications); numApplications != 1 {
		return Manifest{}, fmt.Errorf("single application manifest required, %d found", numApplications)
	}
	manifest.Applications[0].MavenConfig.SetDefaults()
	return manifest, nil
}

func (c *MavenPushPlugin) Run(cliConnection plugin.CliConnection, args []string) {
	if len(args) == 0 || args[0] != "maven-push" {
		os.Exit(0)
	}
	flags := flag.NewFlagSet("maven-push", flag.ContinueOnError)
	manifestPath := flags.String("f", "manifest.yml", "Path to manifest")

	parseArgsNoStdErr(flags, args)

	fmt.Printf("using manifest file %s\n", *manifestPath)
	manifest, err := ParseManifest(*manifestPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if numApplications := len(manifest.Applications); numApplications != 1 {
		fmt.Printf("single application manifest required, %d found", numApplications)
		os.Exit(1)
	}
	config := manifest.Applications[0].MavenConfig

	artifactDir, err := ioutil.TempDir("", "cf-maven-push")
	if err != nil {
		fmt.Printf("failed to create temp dir, %+v", err)
		os.Exit(1)
	}
	defer os.Remove(artifactDir)
	artifactFile := artifactDir + "/artifact"

	err = DownloadArtifact(config.ArtifactUrl(), artifactFile, config.RepoUsername, config.RepoPassword)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	args = append(args, "-p", artifactFile)
	args[0] = "push"

	fmt.Println("running: cf", strings.Join(args, " "))
	cliConnection.CliCommand(args...)
}

//don't show usage for an unknown flag
func parseArgsNoStdErr(flags *flag.FlagSet, args []string) {
	stderr := os.Stderr
	_, os.Stderr, _ = os.Pipe()
	flags.Parse(args[1:])
	os.Stderr = stderr
}

func (c *MavenPushPlugin) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "maven-push",
		Version: plugin.VersionType{
			Major: 0,
			Minor: 1,
			Build: 0,
		},
		MinCliVersion: plugin.VersionType{
			Major: 0,
			Minor: 0,
			Build: 0,
		},
		Commands: []plugin.Command{
			{
				Name:     "maven-push",
				HelpText: "Download and push an application based on maven coordinates defined in the manifest",

				// UsageDetails is optional
				// It is used to show help of usage of each command
				UsageDetails: plugin.Usage{
					Usage: "cf maven-push [-f MANIFEST_PATH]",
				},
			},
		},
	}
}

func main() {
	plugin.Start(new(MavenPushPlugin))
}
