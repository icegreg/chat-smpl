$currentDir = Get-Location
docker run --rm -v "${currentDir}:/app" -w /app/services/admin golang:1.22-alpine sh -c "go mod download && go mod tidy"
