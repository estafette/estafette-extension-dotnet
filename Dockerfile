FROM docker:18.09.0

LABEL maintainer="estafette.io" \
      description="The estafette-extension-dotnet component is an Estafette extension to build and publish .NET Core applications and libraries."

COPY estafette-extension-dotnet /

ENTRYPOINT ["/estafette-extension-dotnet"]