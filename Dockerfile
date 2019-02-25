FROM microsoft/dotnet:2.2-sdk

LABEL maintainer="estafette.io" \
      description="The estafette-extension-dotnet component is an Estafette extension to build and publish .NET Core applications and libraries."

RUN apt-get update && apt-get install -y openjdk-8-jre

ENV PATH "$PATH:/root/.dotnet/tools"

COPY estafette-extension-dotnet /

ENTRYPOINT ["/estafette-extension-dotnet"]