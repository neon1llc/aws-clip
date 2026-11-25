# v1.0 readiness

These are the conditions for calling aws-clip v1.0. They describe a target, not a claim that it has been met.

## What the comparison tools establish

- AWS CLI v2 documents supported systems, installation, upgrades, and troubleshooting. aws-clip must be equally clear about its own install routes and the AWS CLI it wraps.
- Granted names macOS, Linux, and Windows support, while qualifying its Windows testing. aws-clip must name each supported operating system and architecture, and match those claims to CI and release artifacts.
- aws-vault focuses on credential storage and session use. aws-clip has a narrower job: every operation it cannot confidently classify must require confirmation, and its README must state that it is not access control.

Sources: [AWS CLI v2 guide](https://docs.aws.amazon.com/cli/latest/userguide/), [Granted](https://github.com/fwdcloudsec/granted), and [aws-vault](https://github.com/ByteNess/aws-vault).

## Exit criteria

- **Install:** A clean machine can install with the documented Go or binary route, verify the result, and run `aws-clip doctor`. Release archives include checksums.
- **Platforms:** Supported Go toolchains, operating systems, and architectures are explicit for source installs and release binaries. CI builds and tests every claimed platform; unsupported combinations fail clearly.
- **Commands and statuses:** The command, configuration, JSON, and exit-status contracts are documented and tested. Wrapper errors have stable meanings, while a started AWS CLI command preserves its status.
- **Errors:** Missing tools, invalid configuration, identity failures, denied operations, cancellation, and child-process failures give a useful next step without printing credential values or command argument values.
- **Safety:** Known mutations and sensitive-output operations require confirmation. Only exact operations verified against a reviewed AWS model may bypass confirmation; unknown, aliased, or malformed operations fail closed. Tests prove refused commands never start. The documented limits include direct AWS CLI bypass and the separation between identity preflight and the target request.
- **Upgrade:** Configuration and machine-readable schemas have explicit versions and a documented migration path before they are frozen. Install and upgrade instructions do not silently overwrite user configuration.
- **Docs:** Every README command works from a clean checkout or is clearly labeled as an example. The README states prerequisites, platform support, the command contract, security boundary, upgrade path, and known limits.
- **Maintenance:** Required CI checks, a secret scan, pinned workflow actions, and model-drift review pass before a change can merge.
- **Use:** A current sandbox-account verification is documented, three outside users have run the tool for a month and reported, and no open wrong-classification bugs remain.

## Current evidence

The project now generates exact read operations and credential-bearing reads from pinned botocore service models. The classifier also permits reviewed AWS CLI commands absent from those models, including S3 listing and waiters. Unknown operations, including unreviewed names with read-like prefixes, require confirmation. Tests cover direct execution and workflow allowance behavior. CI targets the two Go releases currently supported by the Go project. The project has not yet demonstrated installation or use against a real AWS account outside the author's machines. Passing CI will not establish that broader qualification.
