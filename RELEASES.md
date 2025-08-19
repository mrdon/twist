# Release Process

This project uses [GoReleaser](https://goreleaser.com/) with GitHub Actions for automated releases.

## Creating a Release

To create a new release, simply push a git tag:

```bash
# Create and push a new tag
git tag v0.2.0
git push origin v0.2.0
```

This will automatically:

1. **Trigger GitHub Actions** - The release workflow will start
2. **Build cross-platform binaries** - Linux, macOS, Windows (amd64 + arm64)
3. **Generate changelog** - Based on commits since the last tag
4. **Create GitHub release** - With binaries and release notes
5. **Upload assets** - Compressed archives for each platform

## What Gets Built

- **Linux** (amd64, arm64): `twist_Linux_x86_64.tar.gz`, `twist_Linux_arm64.tar.gz`
- **macOS** (amd64, arm64): `twist_Darwin_x86_64.tar.gz`, `twist_Darwin_arm64.tar.gz`  
- **Windows** (amd64): `twist_Windows_x86_64.zip`

## Version Display

The version information is automatically embedded in the binary during build:

- Displayed in the **status bar** of the TUI application
- Set via ldflags: `-X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}}`

## Development Builds

For local testing with version info:

```bash
# Regular build with dev version
make build

# Multi-platform builds (development snapshots)  
make build-all

# Check GoReleaser configuration
make release-check
```

## Changelog Format

The release notes are automatically generated from commit messages. For better changelogs, use conventional commits:

- `feat: add new feature` → **Features** section
- `fix: resolve bug` → **Bug fixes** section  
- `docs: update readme` → **Others** section

## Manual Release

If you need to create a release manually:

```bash
# Must have a git tag first
git tag v0.2.0
make release
```

## Workflow Files

- `.github/workflows/release.yml` - Automated releases on tag push
- `.github/workflows/test.yml` - CI testing and validation