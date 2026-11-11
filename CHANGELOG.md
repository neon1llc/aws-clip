# Changelog

AWS CLIP releases use `v0.MINOR.PATCH` until the first complete release, v1.0.0, following neon1's [versioning standard](https://neon1.tech/knowledge/standards/versioning/). Each version below has an annotated tag on the commit it names; the release workflow adds missing tags and never moves one.

## v0.1.0 - 2026-09-13
<!-- commit: c90f092292cd2dd60be89ea906777c667666b4c5 -->

### Added
- Direct commands follow an explainable policy: each allowed or refused command says which rule decided it.
- Documentation of the CLI's safety and workflow invariants, with tighter explanations and contract-focused code comments.

## Before v0.1.0

AWS CLIP ran from main before its first tag. These pre-releases mark where each piece landed; their tags were added on 2026-11-11.

### v0.1.0-alpha.6 - 2026-09-13
<!-- commit: 3a7a7007824344a4a70900d466c2e543781b103f -->

#### Fixed
- Costly S3 transfers are guarded.

### v0.1.0-alpha.5 - 2026-09-13
<!-- commit: d27b2768389aac1e6445b30c6bd4d4b2312aa295 -->

#### Added
- Profile operations bound to a verified identity.

### v0.1.0-alpha.4 - 2026-09-13
<!-- commit: b53791920e58209c43d4410af824b4ee38db0b19 -->

#### Fixed
- More workflows that return credentials are blocked.

### v0.1.0-alpha.3 - 2026-09-13
<!-- commit: b3ae032a7836b6642a893ca83675cc2a385b586a -->

#### Added
- Policy-controlled AWS workflows.

### v0.1.0-alpha.2 - 2026-09-13
<!-- commit: 1118ba993957f2bd2c56e5f7f31ace50466a4d8f -->

#### Fixed
- Review fixes that strengthen the execution core.

### v0.1.0-alpha.1 - 2026-09-13
<!-- commit: 37062ffa681f1c4b206e57435c60bd8921882380 -->

#### Added
- A transparent, configurable execution core around the AWS CLI, for any install.

