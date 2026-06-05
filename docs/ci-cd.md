# CI/CD Pipeline

CasaDrop uses GitHub Actions for continuous integration and deployment.

## Workflows

### Build (`build.yml`)

Triggered on:
- Push to `main` or `develop`
- Pull requests to `main`

Jobs:
1. **build** - Compiles Go binary, builds Docker image, runs smoke test
2. **lint** - Runs golangci-lint for code quality
3. **security** - Runs Trivy vulnerability scanner

### Release (`release.yml`)

Triggered on:
- Push of version tags (`v*`)

Jobs:
1. **build-and-push** - Builds multi-arch images, pushes to registries
2. **build-sidecars** - Builds tunnel and tailscale sidecar images

## Creating a Release

```bash
# 1. Update CHANGELOG.md with release notes

# 2. Create and push a version tag
git tag v2.1.0
git push origin v2.1.0

# 3. GitHub Actions will automatically:
#    - Build linux/amd64 and linux/arm64 images
#    - Push to GHCR and Docker Hub
#    - Create a GitHub Release
```

## Docker Registries

### GitHub Container Registry (GHCR)

```bash
docker pull ghcr.io/USERNAME/casadrop:latest
docker pull ghcr.io/USERNAME/casadrop:2.0.0
```

### Docker Hub

```bash
docker pull USERNAME/casadrop:latest
docker pull USERNAME/casadrop:2.0.0
```

## Required Secrets

Configure these in your GitHub repository settings:

| Secret | Description |
|--------|-------------|
| `GITHUB_TOKEN` | Automatic - for GHCR access |
| `DOCKERHUB_USERNAME` | Docker Hub username |
| `DOCKERHUB_TOKEN` | Docker Hub access token |

## Image Tags

Each release creates multiple tags:

| Tag | Example | Description |
|-----|---------|-------------|
| `latest` | `casadrop:latest` | Most recent release |
| `{version}` | `casadrop:2.0.0` | Full version |
| `{major}.{minor}` | `casadrop:2.0` | Minor version |
| `{major}` | `casadrop:2` | Major version |

## Architectures

All images are built for:

- `linux/amd64` - Standard x86_64 servers
- `linux/arm64` - Raspberry Pi 4+, ARM servers

## Dependabot

Automatic dependency updates are configured for:

- Go modules (weekly)
- Docker base images (weekly)
- GitHub Actions (weekly)

PRs are automatically created and labeled.

## Local Testing

Test the CI workflow locally:

```bash
# Build like CI does
docker build -t casadrop:test .

# Run smoke test
docker run -d --name test -p 8080:8080 casadrop:test
sleep 5
curl -f http://localhost:8080/api/auth/status
docker stop test && docker rm test
```

## Troubleshooting

### Build fails on ARM

Ensure QEMU is set up:
```yaml
- uses: docker/setup-qemu-action@v3
```

### Push to Docker Hub fails

Check that `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` secrets are set.

### Cache issues

Clear the GitHub Actions cache:
1. Go to Actions > Caches
2. Delete relevant caches
