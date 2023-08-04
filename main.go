package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/alecthomas/kingpin"
	foundation "github.com/estafette/estafette-foundation"
	"github.com/rs/zerolog/log"
)

var (
	appgroup  string
	app       string
	version   string
	branch    string
	revision  string
	buildDate string
	goVersion = runtime.Version()
)

var (
	// flags
	action                             = kingpin.Flag("action", "Any of the following actions: restore, build, test, unit-test, integration-test, publish, pack, push-nuget").Envar("ESTAFETTE_EXTENSION_ACTION").String()
	configuration                      = kingpin.Flag("configuration", "The build configuration.").Envar("ESTAFETTE_EXTENSION_CONFIGURATION").Default("Release").String()
	buildVersion                       = kingpin.Flag("buildVersion", "The build version.").Envar("ESTAFETTE_EXTENSION_BUILD_VERSION").String()
	project                            = kingpin.Flag("project", "The path to the project for which the tests/build should be run.").Envar("ESTAFETTE_EXTENSION_PROJECT").String()
	runtimeID                          = kingpin.Flag("runtimeId", "The publish runtime.").Envar("ESTAFETTE_EXTENSION_RUNTIME_ID").Default("linux-x64").String()
	forceRestore                       = kingpin.Flag("forceRestore", "Execute the restore on every action.").Envar("ESTAFETTE_EXTENSION_FORCE_RESTORE").Default("false").Bool()
	forceBuild                         = kingpin.Flag("forceBuild", "Execute the build on every action.").Envar("ESTAFETTE_EXTENSION_FORCE_BUILD").Default("false").Bool()
	outputFolder                       = kingpin.Flag("outputFolder", "The folder into which the publish output is generated.").Envar("ESTAFETTE_EXTENSION_OUTPUT_FOLDER").String()
	nugetSources                       = kingpin.Flag("nugetSources", "String array of nuget sources to restore from.").Envar("ESTAFETTE_EXTENSION_SOURCES").String()
	nugetServerURL                     = kingpin.Flag("nugetServerUrl", "The URL of the NuGet server.").Envar("ESTAFETTE_EXTENSION_NUGET_SERVER_URL").String()
	nugetServerAPIKey                  = kingpin.Flag("nugetServerApiKey", "The API key of the NuGet server.").Envar("ESTAFETTE_EXTENSION_NUGET_SERVER_API_KEY").String()
	nugetServerCredentialsJSONPath     = kingpin.Flag("nugetServerCredentials-path", "Path to file with NuGet Server credentials configured at server level, passed in to this trusted extension.").Default("/credentials/nuget_server.json").String()
	nugetServerName                    = kingpin.Flag("nugetServerName", "The name of the preferred NuGet server from the preconfigured credentials.").Envar("ESTAFETTE_EXTENSION_NUGET_SERVER_NAME").Default("github-nuget").String()
	nugetSkipDuplicate                 = kingpin.Flag("nugetSkipDuplicate", "Treat 409 Conflict response as a warning.").Envar("ESTAFETTE_EXTENSION_NUGET_SKIP_DUPLICATE").Default("false").Bool()
	publishReadyToRun                  = kingpin.Flag("publishReadyToRun", "Sets PublishReadyToRun parameter for the publish action when true.").Envar("ESTAFETTE_EXTENSION_PUBLISH_READY_TO_RUN").Default("false").Bool()
	publishSingleFile                  = kingpin.Flag("publishSingleFile", "Sets PublishSingleFile parameter for the publish action when true.").Envar("ESTAFETTE_EXTENSION_PUBLISH_SINGLE_FILE").Default("false").Bool()
	publishTrimmed                     = kingpin.Flag("publishTrimmed", "Sets PublishTrimmed parameter for the publish action when true.").Envar("ESTAFETTE_EXTENSION_PUBLISH_TRIMMED").Default("false").Bool()
	sonarQubeServerURL                 = kingpin.Flag("sonarQubeServerUrl", "The URL of the SonarQube Server to which we send analysis reports.").Envar("ESTAFETTE_EXTENSION_SONARQUBE_SERVER_URL").String()
	sonarQubeServerCredentialsJSONPath = kingpin.Flag("sonarQubeServerCredentials-path", "Path to file with SonarQube Server credentials configured at server level, passed in to this trusted extension.").Default("/credentials/sonarqube_server.json").String()
	sonarQubeServerName                = kingpin.Flag("sonarQubeServerName", "The name of the preferred SonarQube server from the preconfigured credentials.").Envar("ESTAFETTE_EXTENSION_SONARQUBE_SERVER_NAME").String()
	sonarQubeCoverageExclusions        = kingpin.Flag("sonarQubeCoverageExclusions", "The path for the code to be excluded on SonarQube Scan.").Envar("ESTAFETTE_EXTENSION_SONARQUBE_COVERAGE_EXCLUSIONS").String()
)

func main() {

	// parse command line parameters
	kingpin.Parse()

	// init log format from envvar ESTAFETTE_LOG_FORMAT
	foundation.InitLoggingFromEnv(appgroup, app, version, branch, revision, buildDate)

	// create context to cancel commands on sigterm
	ctx := foundation.InitCancellationContext(context.Background())

	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatal().Err(err).Msg("Couldn't determine current working directory.")
	}

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

		if foundation.FileExists("nuget.config") {
			log.Printf("WARNING: NuGet.config was found in the repository, deleting it.\n")
			log.Printf("The NuGet.config should be deleted from the repository, to make sure that only the common default sources are used.\n")
			os.Remove("nuget.config")
		}

		// If the NuGet server URL and credentials are explicitly specified, we use those. Otherwise we retrieve them from the Estafette credentials.
		if *nugetServerURL == "" || *nugetServerAPIKey == "" {
			// use mounted credential file if present instead of relying on an envvar
			if runtime.GOOS == "windows" {
				*nugetServerCredentialsJSONPath = "C:" + *nugetServerCredentialsJSONPath
			}

			if foundation.FileExists(*nugetServerCredentialsJSONPath) {
				*nugetServerURL, *nugetServerAPIKey = getNugetServerCredentialsFromFile(*nugetServerCredentialsJSONPath, *nugetServerName)
			}
		}

		if *nugetServerURL != "" && *nugetServerAPIKey != "" {
			log.Printf("Adding the NuGet source.\n")

			foundation.RunCommandWithArgs(ctx, "dotnet", []string{"nuget", "add", "source", "--username", "travix-tooling-bot", "--password", *nugetServerAPIKey, "--store-password-in-clear-text", "--name", "travix", *nugetServerURL})
		} else {
			log.Printf("No custom NuGet credentials were found.\n")
		}

		// build docker image
		log.Printf("Restoring packages...\n")
		args := []string{
			"restore",
			"--packages",
			".nuget/packages", // This is needed so the packages are restored into the working directory, so they're not lost between the stages.
		}

		if *nugetSources != "" {
			nugetSourcesArray := strings.Split(*nugetSources, ",")

			for _, source := range nugetSourcesArray {
				args = append(args, "--source", source)
			}
		}

		foundation.RunCommandWithArgs(ctx, "dotnet", args)

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

		foundation.RunCommandWithArgs(ctx, "dotnet", args)

	case "test": // Run the tests for the whole solution.

		log.Printf("Running tests for every project in the ./test folder...\n")

		runTests(ctx, "")

	case "unit-test": // Run the unit tests.

		log.Printf("Running tests for projects ending with UnitTests in the ./test folder...\n")

		runTests(ctx, "UnitTests")

	case "integration-test": // Run the integration tests.

		log.Printf("Running tests for projects ending with IntegrationTests in the ./test folder...\n")

		runTests(ctx, "IntegrationTests")

	case "analyze-sonarqube": // Run the SonarQube analysis.

		// Minimal example with defaults.
		// image: extensions/dotnet:stable
		// action: analyze-sonarqube

		// Customizations.
		// image: extensions/dotnet:stable
		// action: analyze-sonarqube
		// sonarQubeServerUrl: https://my-sonar-server.example.com
		// sonarQubeCoverageExclusions: **Tests.cs

		log.Printf("Running the SonarQube analysis...\n")

		// Determine the SonarQube server credentials
		// 1. If sonarQubeServerURL is explicitly specified, we use that.
		// 2. If we have the default credentials from the server level, and sonarQubeServerName is explicitly specified, we look for the credential with the specified name.
		// 3. If we have the default credentials from the server level, and sonarQubeServerName is not specified, we take the first credential. (This is the sensible default if we're using only one SonarQube server.)
		if *sonarQubeServerURL == "" {

			if runtime.GOOS == "windows" {
				*sonarQubeServerCredentialsJSONPath = "C:" + *sonarQubeServerCredentialsJSONPath
			}
			if foundation.FileExists(*sonarQubeServerCredentialsJSONPath) {
				log.Printf("Unmarshalling credentials...")

				log.Info().Msgf("Reading credentials from file at path %v...", *sonarQubeServerCredentialsJSONPath)
				credentialsFileContent, err := ioutil.ReadFile(*sonarQubeServerCredentialsJSONPath)
				if err != nil {
					log.Fatal().Msgf("Failed reading credential file at path %v.", *sonarQubeServerCredentialsJSONPath)
				}

				var credentials []SonarQubeServerCredentials
				err = json.Unmarshal(credentialsFileContent, &credentials)
				if err != nil {
					log.Fatal().Err(err).Msg("Failed unmarshalling credentials")
				}

				if len(credentials) == 0 {
					log.Fatal().Msg("There were no credentials specified.")
				}

				if *sonarQubeServerName != "" {
					credential := GetSonarQubeServerCredentialsByName(credentials, *sonarQubeServerName)
					if credential == nil {
						log.Fatal().Msgf("The NuGet Server credential with the name %v does not exist.", *sonarQubeServerName)
					}

					*sonarQubeServerURL = credential.AdditionalProperties.APIURL
				} else {
					// Just pick the first
					credential := credentials[0]

					*sonarQubeServerURL = credential.AdditionalProperties.APIURL
				}
			} else {
				log.Fatal().Msg("The SonarQube server URL has to be specified to run the analysis.")
			}
		}
		if *sonarQubeCoverageExclusions == "" {
			*sonarQubeCoverageExclusions = "**Tests.cs"
		}

		// dotnet sonarscanner begin /k:"Travix.Core.ShoppingCart" /d:sonar.host.url=https://sonarqube.travix.com /d:sonar.cs.opencover.reportsPaths="**\coverage.opencover.xml" /d:sonar.coverage.exclusions="**Tests.cs"
		args := []string{
			"sonarscanner",
			"begin",
			fmt.Sprintf("/key:%s", solutionName),
			fmt.Sprintf("/d:sonar.host.url=%s", *sonarQubeServerURL),
			"/d:sonar.cs.opencover.reportsPaths=\"**\\coverage.opencover.xml\"",
			fmt.Sprintf("/d:sonar.coverage.exclusions=\"%s\"", *sonarQubeCoverageExclusions),
		}

		if *buildVersion != "" {
			args = append(args, fmt.Sprintf("/version:%s", *buildVersion))
		}

		foundation.RunCommandWithArgs(ctx, "dotnet", args)

		// dotnet build
		args = []string{"build"}

		if *buildVersion != "" {
			args = append(args, fmt.Sprintf("/p:Version=%s", *buildVersion))
		}

		if !*forceRestore {
			args = append(args, "--no-restore")
		}

		foundation.RunCommandWithArgs(ctx, "dotnet", args)

		// Run unit tests with the extra arguments for coverage.
		*forceBuild = true
		runTests(ctx, "UnitTests", "/p:CollectCoverage=true", "/p:CoverletOutputFormat=opencover", "/p:CopyLocalLockFileAssemblies=true")

		// dotnet sonarscanner end
		args = []string{
			"sonarscanner",
			"end",
		}

		foundation.RunCommandWithArgs(ctx, "dotnet", args)

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
		// buildVersion: 1.5.0
		// forceRestore: true

		log.Printf("Publishing the binaries...\n")

		// The solution is called Acme.FooApi, then we by default look for a project called Acme.FooApi.WebService, and if that doesn't exist, we fall back to simply Acme.FooApi
		if *project == "" {
			*project = fmt.Sprintf("src/%s.WebService", solutionName)
			if _, err := os.Stat(*project); os.IsNotExist(err) {
				*project = fmt.Sprintf("src/%s", solutionName)
				if _, err := os.Stat(*project); os.IsNotExist(err) {
					log.Fatal().Err(err).Msg("The project to be published can not be found. Please specify it with the 'project' label.")
				}
			}
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

		if *publishReadyToRun {
			args = append(args, "/p:PublishReadyToRun=true", "/p:PublishReadyToRunShowWarnings=true")
		}
		if *publishSingleFile {
			args = append(args, "/p:PublishSingleFile=true")
		}
		if *publishTrimmed {
			args = append(args, "/p:PublishTrimmed=true")
		}

		if !*forceRestore {
			args = append(args, "--no-restore")
		}

		foundation.RunCommandWithArgs(ctx, "dotnet", args)

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

		foundation.RunCommandWithArgs(ctx, "dotnet", args)

	case "push-nuget": // Pushes the package(s) to NuGet.

		// Minimal example with defaults.
		// image: extensions/dotnet:stable
		// action: push-nuget

		// Customizations.
		// image: extensions/dotnet:stable
		// action: push-nuget
		// nugetServerUrl: https://nuget.mycompany.com
		// nugetServerApikey: 3a4cdeca-3d5b-41a2-ac59-ae4b5c5eaece
		// nugetSkipDuplicate: true

		log.Printf("Publishing the nuget package(s)...\n")

		type nugetCredentials struct {
			url string
			key string
		}

		var nugetPushCredentials []nugetCredentials
		// Determine the NuGet server credentials
		// If nugetServerURL and nugetServerAPIKey are explicitly specified, we use those.
		// Otherwise we automatically push to both GitHub and MyGet. This is temporary, until we finish the transition to GitHub packages.
		if *nugetServerURL == "" || *nugetServerAPIKey == "" {
			// use mounted credential file if present instead of relying on an envvar
			if runtime.GOOS == "windows" {
				*nugetServerCredentialsJSONPath = "C:" + *nugetServerCredentialsJSONPath
			}

			if foundation.FileExists(*nugetServerCredentialsJSONPath) {
				url, key := getNugetServerCredentialsFromFile(*nugetServerCredentialsJSONPath, "github-nuget")
				nugetPushCredentials = append(nugetPushCredentials, nugetCredentials{url: url, key: key})
				url, key = getNugetServerCredentialsFromFile(*nugetServerCredentialsJSONPath, "myget")
				nugetPushCredentials = append(nugetPushCredentials, nugetCredentials{url: url, key: key})
			} else {
				log.Fatal().Msg("The NuGet server URL and API key have to be specified to push a package.")
			}
		} else {
			nugetPushCredentials = append(nugetPushCredentials, nugetCredentials{url: *nugetServerURL, key: *nugetServerAPIKey})
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
			log.Fatal().Msg("No .nupkg files were found.")
		}

		args1 := []string{
			"nuget",
			"push",
		}

		if *nugetSkipDuplicate {
			args1 = append(args1, "--skip-duplicate")
		}

		for i := 0; i < len(files); i++ {
			argsForPackage := []string{}
			argsForPackage = append(argsForPackage, args1...)
			argsForPackage = append(argsForPackage, files[i])

			for _, cred := range nugetPushCredentials {
				argsForServer := []string{}
				argsForServer = append(argsForServer, argsForPackage...)
				argsForServer = append(argsForServer, "--source", cred.url, "--api-key", cred.key)

				foundation.RunCommandWithArgs(ctx, "dotnet", argsForServer)
			}
		}

	default:
		log.Fatal().Msg("Set `action: <action>` on this step to restore, build, test, unit-test, integration-test or publish.")
	}
}

func getNugetServerCredentialsFromFile(credentialsFilePath string, serverName string) (serverURL string, APIKey string) {
	log.Printf("Unmarshalling credentials...")

	log.Info().Msgf("Reading credentials from file at path %v...", credentialsFilePath)
	credentialsFileContent, err := ioutil.ReadFile(credentialsFilePath)
	if err != nil {
		log.Fatal().Msgf("Failed reading credential file at path %v.", credentialsFilePath)
	}

	var credentials []NugetServerCredentials
	err = json.Unmarshal(credentialsFileContent, &credentials)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed unmarshalling credentials")
	}

	if len(credentials) == 0 {
		log.Fatal().Msg("There are no credentials specified.")
	}

	if serverName != "" {
		credential := GetNugetServerCredentialsByName(credentials, serverName)
		if credential == nil {
			log.Fatal().Msgf("The NuGet Server credential with the name %v does not exist.", serverName)
		}

		serverURL = credential.AdditionalProperties.APIURL
		APIKey = credential.AdditionalProperties.APIKey
	} else {
		// Just pick the first
		credential := credentials[0]

		serverURL = credential.AdditionalProperties.APIURL
		APIKey = credential.AdditionalProperties.APIKey
	}

	return
}

// Returns the name of the .NET Core solution in this repository, based on the name of the solution file. If it cannot find a solution file, it returns an empty string.
func getSolutionName() (string, error) {
	files, err := ioutil.ReadDir(".")

	if err == nil {
		for _, f := range files {
			if strings.HasSuffix(f.Name(), ".sln") {
				return strings.TrimSuffix(f.Name(), ".sln"), nil
			}
		}

		return "", nil
	}

	return "", err
}

// Runs the unit tests for all projects in the ./test folder which have the passed in suffix in their name.
func runTests(ctx context.Context, projectSuffix string, extraArgs ...string) {
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

	args = append(args, extraArgs...)

	files, err := ioutil.ReadDir("./test")

	if err == nil {
		for _, f := range files {
			if f.IsDir() && strings.HasSuffix(f.Name(), projectSuffix) {
				log.Printf("Running tests for ./test/%s...\n", f.Name())

				argsForProject := make([]string, len(args)+1)
				copy(argsForProject, args)

				argsForProject = append(argsForProject, fmt.Sprintf("./test/%s", f.Name()))

				foundation.RunCommandWithArgs(ctx, "dotnet", argsForProject)
			}
		}
	} else if !os.IsNotExist(err) { // If we got an error just because the "test" folder doesn't exist, that's fine, we can ignore. We only fail with an error if it was something else.
		log.Fatal().Err(err).Msg("Failed to read subdirectories under ./test.")
	}
}
