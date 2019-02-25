# estafette-extension-dotnet

This extension allows you to build and publish .NET Core applications and libraries.

On every stage, we have to specify the `action` label, which can have the following values: `restore`, `build`, `test`, `unit-test`, `integration-test`, `publish`, `pack`, `push-nuget`.

If we don't specify any other labels, then the extension executes an opinionated build with sensible defaults.

## Example

A full build and publish process (with the default setting everywhere) of an API looks like this.

```
stages:
  restore:
    image: extensions/dotnet:2.2-stable
    action: restore

  build:
    image: extensions/dotnet:2.2-stable
    action: build

  tests:
    image: extensions/dotnet:2.2-stable
    action: test

  publish:
    image: extensions/dotnet:2.2-stable
    action: publish
```

And for a library:

```
stages:
  restore:
    image: extensions/dotnet:2.2-stable
    action: restore

  build:
    image: extensions/dotnet:2.2-stable
    action: build

  tests:
    image: extensions/dotnet:2.2-stable
    action: test

  pack:
    image: extensions/dotnet:2.2-stable
    action: pack

  push-nuget:
    image: extensions/dotnet:2.2-stable
    action: push-nuget
```

## Actions

This section describes the supported actions and their configuration arguments.

### Common configuration arguments

The following  arguments can be used in multiple actions.

 - `buildVersion`: Instead of using the version of the Estafette build, we'll use this explicitly specified version during the `build`, `publish` and `pack` steps.
 - `configuration`: Instead of `Release`, we'll use this configuration during the compilation.
 - `forceRestore`: We force executing the package restore on every step, not just on `restore`.
 - `forceBuild`: We force executing the build on every step, not just on `build`.

### build

Builds all the projects in the solution by executing `dotnet build` in the root.

Syntax:

```
  build:
    image: extensions/dotnet:2.2-stable
    action: build
    configuration: Debug
    buildVersion: 1.5.0
    forceRestore: true
```

### test

Runs the tests for every test project in the `./test` folder.

Syntax:

```
  test:
    image: extensions/dotnet:2.2-stable
    action: test
    configuration: Debug
    forceRestore: true
    forceBuild: true
```

### unit-test

The same as `test`, but only runs the tests for projects ending with `UnitTests`.

### integration-test

The same as `test`, but only runs the tests for projects ending with `IntegrationTests`.

### analyze-sonarqube

Runs the SonarQube analysis on the whole solution, and sends the analysis report to the Sonar server.  
It also collects test coverage, if the `coverlet.msbuild` package is added to the Unit test projects as a package dependency.

```
  test:
    image: extensions/dotnet:2.2-stable
    action: analyze-sonarqube
```

The URL of the Sonar server to use can be customized with the `sonarQubeServerUrl` field.

### publish

Generates the final binaries by executing `dotnet publish`.

By default it tries to publish a project in the `./src` folder with the name `<SolutionName>.WebService`. You can override this by explicitly specifying the `project` field.

If the `outputFolder` is not specified, it puts the binaries in the `./publish` folder *directly the root*.

The default runtime identifier is `linux-x64`, this can be overridden with the `runtimeId` field.

Syntax:

```
  publish:
    image: extensions/dotnet:2.2-stable
    action: publish
    forceRestore: true
    forceBuild: true
    project: src/CustomProject
    configuration: Debug
    runtimeId: windows10-x64
    outputFolder: ./binaries
    buildVersion: 1.5.0
```

### pack

Creates the NuGet package by executing `nuget pack`.

Syntax:

```
  pack:
    image: extensions/dotnet:2.2-stable
    action: pack
    configuration: Debug
    forceRestore: true
    forceBuild: true
    buildVersion: 1.5.0
```

### push-nuget

Pushes all the NuGet packages build with the `pack` action to a NuGet server.  
It finds the packages by finding all the files under the `src` folder which have the `nupkg` extension.

If we don't specify the NuGet server in any way.

```
  push-nuget:
    image: extensions/dotnet:2.2-stable
    action: push-nuget
```

Then we'll try to pick the first one from the default server credentials configured in the Estafette CI server.

If we have multiple credentials configured, then we can also pick one by its name.

```
  push-nuget:
    image: extensions/dotnet:2.2-stable
    action: push-nuget
    nugetServerName: my-configured-server
```

Or we can explicitly configure the URL and the API Key, that way we're not using the default credentials.

```
  push-nuget:
    image: extensions/dotnet:2.2-stable
    action: push-nuget
    nugetServerUrl: https://nuget.mycompany.com
    nugetServerApiKey: 3a4cdeca-3d5b-41a2-ac59-ae4b5c5eaece
```
