ARG SDK_VERSION_TAG
ARG DOCKER_REPOSITORY=microsoft/dotnet
FROM ${DOCKER_REPOSITORY}:${SDK_VERSION_TAG}

ARG OPENJDK_PACKAGE=openjdk-8-jre

LABEL maintainer="estafette.io" \
      description="The estafette-extension-dotnet component is an Estafette extension to build and publish .NET Core applications and libraries."

RUN apt-get update && apt-get install -y $OPENJDK_PACKAGE \
    && dotnet tool install --global --version 4.7.1 dotnet-sonarscanner

ENV PATH="$PATH:/root/.dotnet/tools" \
    ESTAFETTE_LOG_FORMAT="console"

COPY estafette-extension-dotnet /

ENTRYPOINT ["/estafette-extension-dotnet"]