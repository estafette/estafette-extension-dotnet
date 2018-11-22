package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/alecthomas/kingpin"
)

var (
	version   string
	branch    string
	revision  string
	buildDate string
	goVersion = runtime.Version()
)

var (
	// flags
	action                     = kingpin.Flag("action", "Any of the following actions: restore, build, test, unit-test, integration-test, publish, pack, push-nuget").Envar("ESTAFETTE_EXTENSION_ACTION").String()
	configuration              = kingpin.Flag("configuration", "The build configuration.").Envar("ESTAFETTE_EXTENSION_CONFIGURATION").Default("Release").String()
	buildVersion               = kingpin.Flag("buildVersion", "The build version.").Envar("ESTAFETTE_EXTENSION_BUILD_VERSION").String()
	project                    = kingpin.Flag("project", "The path to the project for which the tests/build should be run.").Envar("ESTAFETTE_EXTENSION_PROJECT").String()
	runtimeID                  = kingpin.Flag("runtimeId", "The publish runtime.").Envar("ESTAFETTE_EXTENSION_RUNTIME_ID").Default("linux-x64").String()
	forceRestore               = kingpin.Flag("forceRestore", "Execute the restore on every action.").Envar("ESTAFETTE_EXTENSION_FORCE_RESTORE").Default("false").Bool()
	forceBuild                 = kingpin.Flag("forceBuild", "Execute the build on every action.").Envar("ESTAFETTE_EXTENSION_FORCE_BUILD").Default("false").Bool()
	outputFolder               = kingpin.Flag("outputFolder", "The folder into which the publish output is generated.").Envar("ESTAFETTE_EXTENSION_OUTPUT_FOLDER").String()
	nugetServerURL             = kingpin.Flag("nugetServerUrl", "The URL of the NuGet server.").Envar("ESTAFETTE_EXTENSION_NUGET_SERVER_URL").String()
	nugetServerAPIKey          = kingpin.Flag("nugetServerApiKey", "The API key of the NuGet server.").Envar("ESTAFETTE_EXTENSION_NUGET_SERVER_API_KEY").String()
	nugetServerCredentialsJSON = kingpin.Flag("nugetServerCredentials", "NuGet Server credentials configured at server level, passed in to this trusted extension.").Envar("ESTAFETTE_CREDENTIALS_NUGET_SERVER").String()
	nugetServerName            = kingpin.Flag("nugetServerName", "The name of the preferred NuGet server from the preconfigured credentials.").Envar("ESTAFETTE_EXTENSION_NUGET_SERVER_NAME").String()
)

func main() {

	// parse command line parameters
	kingpin.Parse()

	workingDir, err := os.Getwd()

	if err != nil {
		log.Fatal("Couldn't determine current working directory.")
	}

	// log to stdout and hide timestamp
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))

	// log startup message
	log.Printf("Starting estafette-extension-dotnet version %v...", version)

	// set defaults
	builtInBuildVersion := os.Getenv("ESTAFETTE_BUILD_VERSION")
	if *buildVersion == "" {
		*buildVersion = builtInBuildVersion
	}

	solutionName, _ := getSolutionName()

	if solutionName == "" {
		log.Printf("Unknown solution")
	} else {
		log.Printf("Solution name: %s", solutionName)
	}

	switch *action {
	case "restore": // Restore package dependencies with dotnet restore.

		// Minimal example with defaults.
		// image: extensions/dotnet:stable
		// action: restore

		// build docker image
		log.Printf("Restoring pacakges...\n")
		args := []string{
			"restore",
			"--packages",
			".nuget/packages", // This is needed so the packages are restored into the working directory, so they're not lost between the stages.
		}

		runCommand("dotnet", args)

	case "build": // Build the solution.

		// Minimal example with defaults.
		// image: extensions/dotnet:stable
		// action: build

		// Customizations.
		// image: extensions/dotnet:stable
		// action: build
		// configuration: Debug
		// versionSuffix: 5

		log.Printf("Building the solution...\n")

		args := []string{
			"build",
			"--configuration",
			*configuration,
		}

		if *buildVersion != "" {
			args = append(args, fmt.Sprintf("/p:Version=%s", *buildVersion))
		}

		if !*forceRestore {
			args = append(args, "--no-restore")
		}

		runCommand("dotnet", args)

	case "test": // Run the tests for the whole solution.

		log.Printf("Running tests for every project in the ./test folder...\n")

		runTests("")

	case "unit-test": // Run the unit tests.

		log.Printf("Running tests for projects ending with UnitTests in the ./test folder...\n")

		runTests("UnitTests")

	case "integration-test": // Run the integration tests.

		log.Printf("Running tests for projects ending with IntegrationTests in the ./test folder...\n")

		runTests("IntegrationTests")

	case "publish": // Publish the final binaries.

		// Minimal example with defaults.
		// image: extensions/dotnet:stable
		// action: publish

		// Customizations.
		// image: extensions/dotnet:stable
		// action: publish
		// project: src/CustomProject
		// configuration: Debug
		// runtimteId: windows10-x64
		// outputFolder: ./binaries
		// versionSuffix: 5

		log.Printf("Publishing the binaries...\n")

		if *project == "" {
			*project = fmt.Sprintf("src/%s.WebService", solutionName)
		}

		if *outputFolder == "" {
			// A default sensible choice is to put the publish output directly under the working folder in a folder called "publish", so that its relative path doesn't depend on the project name.
			// This makes it easier to use in a generic way in followup steps of the build.
			*outputFolder = filepath.Join(workingDir, "publish")
		}

		args := []string{
			"publish",
			"--configuration",
			*configuration,
			"--runtime",
			*runtimeID,
			"--output",
			*outputFolder,
			*project,
		}

		if *buildVersion != "" {
			args = append(args, fmt.Sprintf("/p:Version=%s", *buildVersion))
		}

		if !*forceRestore {
			args = append(args, "--no-restore")
		}

		runCommand("dotnet", args)

	case "pack": // Pack the NuGet package.

		// Minimal example with defaults.
		// image: extensions/dotnet:stable
		// action: pack

		// Customizations.
		// image: extensions/dotnet:stable
		// action: pack
		// force-restore: true
		// force-build: true
		// configuration: Debug
		// versionSuffix: 5

		log.Printf("Packing the nuget package(s)...\n")

		args := []string{
			"pack",
			"--configuration",
			*configuration,
		}

		if *buildVersion != "" {
			args = append(args, fmt.Sprintf("/p:Version=%s", *buildVersion))
		}

		if !*forceRestore {
			args = append(args, "--no-restore")
		}

		if !*forceBuild {
			args = append(args, "--no-build")
		}

		runCommand("dotnet", args)

	case "push-nuget": // Pushes the package(s) to NuGet.

		// Minimal example with defaults.
		// image: extensions/dotnet:stable
		// action: push-nuget

		// Customizations.
		// image: extensions/dotnet:stable
		// action: push-nuget
		// nugetServerUrl: https://nuget.mycompany.com
		// nugetServerApikey: 3a4cdeca-3d5b-41a2-ac59-ae4b5c5eaece

		log.Printf("Publishing the nuget package(s)...\n")

		// Determine the NuGet server credentials
		// 1. If nugetServerURL and nugetServerAPIKey are explicitly specified, we use those.
		// 2. If we have the default credentials from the server level, and nugetServerName is explicitly specified, we look for the credential with the specified name.
		// 3. If we have the default credentials from the server level, and nugetServerName is not specified, we take the first credential. (This is the sensible default if we're using only one NuGet server.)

		if *nugetServerURL == "" || *nugetServerAPIKey == "" {
			if *nugetServerCredentialsJSON != "" {
				log.Printf("Unmarshalling credentials...")
				var credentials []NugetServerCredentials
				err := json.Unmarshal([]byte(*nugetServerCredentialsJSON), &credentials)
				if err != nil {
					log.Fatal("Failed unmarshalling credentials: ", err)
				}

				if len(credentials) == 0 {
					log.Fatal("There were no credentials specified.")
				}

				if *nugetServerName != "" {
					credential := GetNugetServerCredentialsByName(credentials, *nugetServerName)
					if credential == nil {
						log.Fatalf("The NuGet Server credential with the name %v does not exist.", *nugetServerName)
					}

					*nugetServerURL = credential.AdditionalProperties.APIURL
					*nugetServerAPIKey = credential.AdditionalProperties.APIKey
				} else {
					// Just pick the first
					credential := credentials[0]

					*nugetServerURL = credential.AdditionalProperties.APIURL
					*nugetServerAPIKey = credential.AdditionalProperties.APIKey
				}
			} else {
				log.Fatal("The NuGet server URL and API key have to be specified to push a package.")
			}
		}

		srcPath := filepath.Join(workingDir, "src")

		var files []string
		filepath.Walk(srcPath, func(path string, f os.FileInfo, _ error) error {
			if !f.IsDir() {
				if filepath.Ext(path) == ".nupkg" {
					files = append(files, path)
				}
			}
			return nil
		})

		if len(files) == 0 {
			log.Fatal("No .nupkg files were found.")
		}

		args := []string{
			"nuget",
			"push",
			"--source",
			*nugetServerURL,
			"--api-key",
			*nugetServerAPIKey,
		}

		for i := 0; i < len(files); i++ {
			argsForPackage := make([]string, len(args)+1)
			copy(argsForPackage, args)

			argsForPackage = append(argsForPackage, files[i])

			runCommand("dotnet", argsForPackage)
		}

	default:
		log.Fatal("Set `action: <action>` on this step to restore, build, test, unit-test, integration-test or publish.")
	}
}

// Returns the name of the .NET Core solution in this repository, based on the name of the solution file. If it cannot find a solution file, it returns an empty string.
func getSolutionName() (string, error) {
	files, err := ioutil.ReadDir(".")

	if err == nil {
		for _, f := range files {
			if strings.HasSuffix(f.Name(), ".sln") {
				return strings.TrimRight(f.Name(), ".sln"), nil
			}
		}

		return "", nil
	}

	return "", err
}

// Runs the unit tests for all projects in the ./test folder which have the passed in suffix in their name.
func runTests(projectSuffix string) {
	// Minimal example with defaults.
	// image: extensions/dotnet:stable
	// action: build

	// Customizations.
	// image: extensions/dotnet:stable
	// action: build
	// configuration: Debug
	// versionSuffix: 5

	args := []string{
		"test",
		"--configuration",
		*configuration,
	}

	if !*forceRestore {
		args = append(args, "--no-restore")
	}

	if !*forceBuild {
		args = append(args, "--no-build")
	}

	files, err := ioutil.ReadDir("./test")

	if err != nil {
		log.Fatal("Failed to read subdirectories under ./test.")
	}

	for _, f := range files {
		if f.IsDir() && strings.HasSuffix(f.Name(), projectSuffix) {
			log.Printf("Running tests for ./test/%s...\n", f.Name())

			argsForProject := make([]string, len(args)+1)
			copy(argsForProject, args)

			argsForProject = append(argsForProject, fmt.Sprintf("./test/%s", f.Name()))

			runCommand("dotnet", argsForProject)
		}
	}
}

func handleError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func runCommand(command string, args []string) {
	log.Printf("Running command '%v %v'...", command, strings.Join(args, " "))
	cmd := exec.Command(command, args...)
	cmd.Dir = "/estafette-work"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	handleError(err)
}
