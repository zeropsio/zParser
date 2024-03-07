# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v2.0.0] - 2024-03-07

### Added
- support for `OptionFunc` to configure the parser
- way to customize how multiline output is handled.
  - It can be either preserved, squashed to one line or preserved with indent (e.g. for yaml)

### Changed
- parser constructor now takes variadic parameter `OptionFunc`

### Removed
- `maxFunctionCount` parameter from constructor

### Fixed
- issue where multiline function output would not be indented if it was nested inside of text block.

## [v1.2.2] - 2024-02-28

### Changed
- GitHub actions should now use Makefile and correctly upload Assets for new tags

### Removed
- `bin` directory is now in `.gitignore` file, since binaries should be now available as part of a release

## [v1.2.1] - 2024-02-24

### Added
- `Argon2idConfig` struct for easier configuration of Argon2id hashing
- `DefaultArgon2idConf` and `Argon2IDPasswordHash` functions for Argon2id hashing
- `goimports` and `gci` linters

### Changed
- Updated to GO 1.19
- Updated dependencies to latest versions
- Updated `golangci-lint` to latest version and updated settings of some linters

### Removed
- `gitlab-ci.yaml` - not needed as repo was moved to GitHub

## [v1.2.0] - 2023-08-28

### Changed
- moved module from private GitLab repo to GitHub

## [v1.1.0] - 2022-12-15

### Changed
- `generateRandomString` and `generateRandomStringVar` now use `[0-9a-zA-Z_-.]` character set instead of `[0-9a-f]`

### Fixed 
- Typo in README file

## [v1.0.1] - 2022-12-09

### Fixed
- Small issues in README
- Typo in example file

### Added
- GitHub workflows
- Changelog file

## [v1.0.0] - 2022-11-29

### Added
- Initial release
