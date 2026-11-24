# Changelog

AWS CLIP releases use `v0.MINOR.PATCH` until v1.0.0, following neon1's [versioning standard](https://neon1.tech/knowledge/standards/versioning/). Each version below has an annotated tag on its commit. The release workflow adds missing tags and never moves one.

## Unreleased

### Changed
- The sensitive-output list is generated from AWS service models instead of kept by hand. Read-style operations that return a credential-like member now need confirmation or a workflow allow rule, including `cognito-idp describe-user-pool-client`, `ec2 describe-vpn-connections`, `lightsail get-relational-database-master-user-password`, `sts get-web-identity-token`, and others. A reviewed list of false positives, such as `rds describe-db-instances`, stays read-only.
- CI checks the list against the pinned `botocore` and compares operation names with botocore's own conversion. A weekly job fails when the newest models add an operation that needs review.

## v0.2.1 - 2026-11-20

### Changed
- Shorter README and `--help` text. The README no longer says there are no prebuilt binaries.

## v0.2.0 - 2026-11-19

### Added
- Each release carries macOS and Linux binaries (arm64 and amd64) and a `checksums.txt` file. The README shows how to check and install them.

## v0.1.5 - 2026-11-19

### Changed
- The Go module path is now `github.com/neon1llc/aws-clip`, so `go install github.com/neon1llc/aws-clip/cmd/aws-clip@latest` works.
- The README states the release status, says aws-clip isn't access control, and compares it with AWS CLI v2, Granted, aws-vault, and IAM.

### Fixed
- These credential-returning operations now need confirmation or a workflow allow rule: `redshift get-cluster-credentials`, `redshift-serverless get-credentials`, `ec2 get-password-data`, `ecr-public get-authorization-token`, `lightsail get-instance-access-details`, `connect get-federation-token`, and `gamelift get-instance-access`.

## v0.1.4 - 2026-11-15

### Changed
- Cut README wording that repeated a table, list, or linked page.
- Each release's changelog entry is published as its GitHub release.

## v0.1.3 - 2026-11-15

### Changed
- Shorter, plainer README and help text.

## v0.1.2 - 2026-11-14

### Changed
- The MIT license now names neon1 LLC as the copyright holder.

## v0.1.1 - 2026-11-13

### Changed
- Every pull request and push to main runs formatting, vet, build, and race-enabled tests, including tests that prove denied and unknown commands are refused. A version is tagged only after they pass.

## v0.1.0 - 2026-09-13
<!-- commit: c90f092292cd2dd60be89ea906777c667666b4c5 -->

### Added
- Direct commands follow a policy, and each allowed or refused command says which rule decided it.
- Documentation of the CLI's safety and workflow invariants.

## Before v0.1.0

AWS CLIP ran from main before its first tag. These pre-releases mark where each piece landed. Their tags were added on 2026-11-11.

### v0.1.0-alpha.6 - 2026-09-13
<!-- commit: 3a7a7007824344a4a70900d466c2e543781b103f -->

#### Fixed
- Costly S3 transfers are guarded.

### v0.1.0-alpha.5 - 2026-09-13
<!-- commit: d27b2768389aac1e6445b30c6bd4d4b2312aa295 -->

#### Added
- Profile operations bound to a verified account.

### v0.1.0-alpha.4 - 2026-09-13
<!-- commit: b53791920e58209c43d4410af824b4ee38db0b19 -->

#### Fixed
- More credential-returning operations are blocked in workflows.

### v0.1.0-alpha.3 - 2026-09-13
<!-- commit: b3ae032a7836b6642a893ca83675cc2a385b586a -->

#### Added
- Workflows with policy control.

### v0.1.0-alpha.2 - 2026-09-13
<!-- commit: 1118ba993957f2bd2c56e5f7f31ace50466a4d8f -->

#### Fixed
- Fixes to the execution core.

### v0.1.0-alpha.1 - 2026-09-13
<!-- commit: 37062ffa681f1c4b206e57435c60bd8921882380 -->

#### Added
- A configurable execution core around AWS CLI.

