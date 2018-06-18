package main

import (
	"code.cloudfoundry.org/cli/plugin"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

type MavenPushPlugin struct {
}

func (c *MavenPushPlugin) Run(cliConnection plugin.CliConnection, args []string) {
	if len(args) == 0 || args[0] != "maven-push" {
		os.Exit(0)
	}
	flags := flag.NewFlagSet("maven-push", flag.ContinueOnError)
	flags.SetOutput(ioutil.Discard)
	manifestPath := flags.String("f", "manifest.yml", "Path to manifest")
	flags.Parse(args[1:])

	fmt.Printf("using manifest file %s\n", *manifestPath)
	err := ValidateManifest(*manifestPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
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
