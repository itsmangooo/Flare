# syntax=docker/dockerfile:1.7

FROM mcr.microsoft.com/dotnet/sdk:10.0.300 AS build

WORKDIR /source

COPY global.json Directory.Build.props ./

COPY src/Flare.Contracts/Flare.Contracts.csproj src/Flare.Contracts/
COPY src/Flare.Api/Flare.Api.csproj src/Flare.Api/

RUN dotnet restore src/Flare.Api/Flare.Api.csproj

COPY src/Flare.Contracts/ src/Flare.Contracts/
COPY src/Flare.Api/ src/Flare.Api/

RUN dotnet publish src/Flare.Api/Flare.Api.csproj \
    --configuration Release \
    --no-restore \
    --output /app/publish \
    /p:UseAppHost=false


FROM mcr.microsoft.com/dotnet/aspnet:10.0.8-noble AS runtime

WORKDIR /app

ENV ASPNETCORE_URLS=http://+:8080 \
    ASPNETCORE_HTTP_PORTS=8080 \
    DOTNET_EnableDiagnostics=0

COPY --from=build --chown=$APP_UID:$APP_UID /app/publish ./

USER $APP_UID

EXPOSE 8080

HEALTHCHECK \
    --interval=30s \
    --timeout=5s \
    --start-period=10s \
    --retries=3 \
    CMD ["dotnet", "Flare.Api.dll", "--healthcheck"]

ENTRYPOINT ["dotnet", "Flare.Api.dll"]
