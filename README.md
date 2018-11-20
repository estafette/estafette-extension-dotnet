# estafette-extension-dotnet

This extension allows you to build and publish .NET Core applications and libraries.

On every stage, we have to specify the `action` label, which can have the following values: `restore`, `build`, `test`, `unit-test`, `integration-test`, `publish`, `pack`, `push-nuget`.

If we don't specify any other labels, then the extension executes an opinionated build with sensible defaults.

## Example

A full build and publish process of an API looks like this.

```
  restore:
    image: extensions/dotnet:stable
    action: restore

  build:
    image: extensions/dotnet:stable
    action: build

  tests:
    image: extensions/dotnet:stable
    action: test

  publish:
    image: extensions/dotnet:stable
    action: publish
```

## Actions
