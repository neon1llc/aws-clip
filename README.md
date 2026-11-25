# AWS CLIP

AWS CLIP (AWS CLI Plus, `aws-clip`) sits in front of AWS CLI v2 for multi-profile work. It finds your profiles, logs in through each profile's own path, checks the active account and principal, and confirms risky commands against the account you expect. For one-off work, use `aws` directly.

**Status:** v0.x, early software. Tests run against a fake AWS CLI and never contact AWS. I've only used it on my own machines.

**Not access control.** Anyone with a shell can run `aws` directly. Real limits belong in IAM, permission boundaries, and service control policies.

## Where it fits

| Need | Use |
| --- | --- |
| Short-lived credentials, SSO login, profile config | AWS CLI v2 itself: [`aws configure sso`](https://docs.aws.amazon.com/cli/latest/userguide/sso-configure-profile-token.html) and [`aws login`](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-authentication.html) |
| Fast role switching and several consoles open at once | [Granted](https://github.com/fwdcloudsec/granted) |
| Storing long-lived IAM keys in an OS keystore | An actively maintained [aws-vault fork](https://github.com/ByteNess/aws-vault), since the [original](https://github.com/99designs/aws-vault) is abandoned. Better, move off long-lived keys |
| A prompt and account check at the moment you run a risky CLI command | **aws-clip** |

aws-clip works beside the others and stores no credentials. It catches a shell still pointed at production when you meant development.

## Install

Needs AWS CLI v2 on `PATH`. With Go 1.26 or newer:

```sh
go install github.com/neon1llc/aws-clip/cmd/aws-clip@latest
aws-clip doctor
```

Or download a macOS or Linux binary (arm64 or amd64) from the [latest release](https://github.com/neon1llc/aws-clip/releases/latest) and check it against `checksums.txt`:

```sh
# Linux amd64
sha256sum --check --ignore-missing checksums.txt
chmod +x aws-clip_linux_amd64
mv aws-clip_linux_amd64 ~/.local/bin/aws-clip
```

```sh
# macOS arm64
shasum -a 256 --check --ignore-missing checksums.txt
chmod +x aws-clip_darwin_arm64
mv aws-clip_darwin_arm64 ~/.local/bin/aws-clip
```

Then confirm that AWS CLI v2 is on `PATH` and run:

```sh
aws-clip doctor
```

Release binaries are provided for Linux and macOS on amd64 and arm64. Windows amd64 is supported through `go install`; prebuilt Windows binaries are not provided. CI tests Go 1.26 and 1.27 on Linux, macOS, and Windows.

Or build from a clone:

```sh
make build
./bin/aws-clip doctor
```

## Start with a profile

```sh
aws-clip profiles
aws-clip profiles --format json
```

Output never includes access key IDs, credential-process commands, SSO start URLs, login-session names, or credential values. JSON has `schema_version: 1`.

Create profiles with AWS CLI, then select one with `--profile`, `AWS_CLIP_PROFILE`, or the configuration file. Login and guarded commands never fall back to the default profile.

```sh
aws configure sso --profile development
aws-clip --profile development login
aws-clip --profile development context
```

## Log in and out

`login` asks STS whether the profile already has valid credentials and reuses them. Otherwise it follows the profile's metadata, including `source_profile` chains:

- IAM Identity Center profiles run `aws sso login`.
- AWS local login profiles run `aws login`.
- Role profiles use their source profile's login path.
- Static keys, credential processes, web identity, instance or container roles, and the default provider chain stay with their own provider. The message names the provider to refresh.

```sh
aws-clip --profile production login
aws-clip --profile development logout
```

SSO logout clears every cached SSO session, so it needs approval:

```sh
aws-clip --profile development logout --approve all-sso-sessions
```

## Verify the active context

`context` calls `aws sts get-caller-identity` through the selected profile and reports non-secret details: profile, account ID, principal ARN, user ID, effective Region, credential source, and account-guard state. Safety decisions use the observed STS account, never the profile name.

```sh
aws-clip --profile production --region us-east-1 context
aws-clip --profile production context --format json
```

## Guarded commands

`exec` requires an explicit profile, rejects arguments that change context, classifies the operation, applies local policy, and prints the profile, Region, classification, and decision on stderr. Then it passes literal arguments and standard streams to AWS CLI.

```sh
aws-clip --profile development exec -- ec2 describe-instances --output json
```

Nothing after `--` goes through a shell. Set `--profile` and `--region` before `exec`. After the separator, `--profile`, `--region`, `--endpoint-url`, and `--no-sign-request` are rejected.

Only `list-*`, `describe-*`, `get-*`, `head-*`, `lookup-*`, `search-*`, `validate-*`, and `wait` count as read-only. Known mutations, unrecognized operations, and known credential-, token-, and secret-returning operations need confirmation. Interactive use prompts. Non-interactive use fails unless you pass `--yes`.

```sh
aws-clip --profile development exec --yes -- \
  ec2 terminate-instances --instance-ids i-example
```

`--preview` is local. It starts no AWS CLI process, runs no identity check, and prints option names but no values. Pass `--region` if the preview must check a Region allowlist, since a Region inherited from the AWS profile shows as unresolved.

```sh
aws-clip --profile development --region us-east-1 exec --preview -- \
  secretsmanager put-secret-value --secret-id example --secret-string hidden
```

Classification uses exact service and operation names from reviewed AWS service models, plus a short reviewed list for AWS CLI commands that have no model entry. A newly introduced or unfamiliar operation requires confirmation even when its name starts with `get-` or `describe-`. Classification does not inspect argument values or remote state. Read-style operations that return credentials, such as `get-secret-value` or `describe-user-pool-client`, count as sensitive output. The sensitive-output list is generated from AWS service models: any read-style operation whose output has a credential-like member is on it, minus a short reviewed list of false positives such as `rds describe-db-instances`. A few commands the models can't show, such as `eks get-token`, are listed by hand. The models come from the PyPI `botocore` package, which can differ from the ones bundled in your AWS CLI, and a weekly CI job fails when newer models add an operation that needs review. The rule matches member names, so it can miss an operation with an unusual name. [Open an issue](https://github.com/neon1llc/aws-clip/issues) if you find one.

### Restrict direct commands

Command policy lives in the configuration file only. Deny rules always win. When `allow` isn't empty, the command must match it, and so must each non-empty allowlist.

```json
{
  "profile": "automation",
  "region": "us-east-1",
  "command_policy": {
    "allow": ["ec2:describe-*", "s3api:put-object"],
    "deny": ["*:delete-*", "ec2:terminate-*"],
    "allowed_profiles": ["automation"],
    "allowed_regions": ["us-east-1"],
    "allowed_account_ids": ["123456789012"]
  }
}
```

An account allowlist runs `sts get-caller-identity` just before confirmation, and `--yes` can't override a mismatch. Without an account allowlist or protected profile, `exec` doesn't call STS. `--approve-account` also requires a matching account but doesn't replace `--yes` for non-interactive changes.

### Protect production profiles

Bind a high-risk profile to its 12-digit account in the configuration file:

```json
{
  "profile": "production",
  "region": "us-east-1",
  "protected_profiles": {
    "production": "123456789012"
  }
}
```

If `production` resolves to another account, every live `context` or `exec` check fails with status `5`. On the right account, every non-read operation still needs confirmation or `--yes`. Protection is file-only, so no environment variable or flag can weaken it.

`context`, `login`, `exec`, and workflow runs block credential environment variables such as `AWS_ACCESS_KEY_ID` and `AWS_SESSION_TOKEN`, and report only their names. Clear them so the profile is authoritative.

## Doctor

```sh
aws-clip doctor
aws-clip --profile production doctor
```

`doctor` checks AWS CLI v2, configuration, profiles, and credential environment conflicts. With a profile it also verifies identity through STS and any protected account binding. Failures name the next command.

## Reviewed workflows

For plain repeatability, a Makefile or script is simpler. Use a workflow for policy review, named approval, an identity check, and fail-fast execution.

A workflow is a versioned JSON file of named AWS CLI steps:

```json
{
  "schema_version": 1,
  "name": "release-check",
  "description": "Confirm identity and inspect the deployed stack",
  "steps": [
    {"name": "identity", "command": ["sts", "get-caller-identity"]},
    {"name": "stack", "command": ["cloudformation", "describe-stacks", "--stack-name", "application"]}
  ]
}
```

`plan` is local and calls neither AWS CLI nor AWS. It shows effective context, protected account configuration, whether account approval is needed, and each `service:operation` with its policy decision, omitting argument values. JSON plans have `schema_version: 1` and ordered steps.

```sh
aws-clip --profile production plan release-check.json
aws-clip --profile production run --approve release-check release-check.json
```

A run needs the exact workflow name, an explicit profile, and a live STS identity check. A destructive or sensitive-output step needs `--approve-account`, and so does any change on a protected profile. All policy and approvals are checked before the first step. Steps stop at the first failure, returning the failing AWS status unchanged.

### Workflow policy

Read-only operations are allowed by default, except known credential-, token-, and secret-returning ones. Everything else needs an allow rule. Deny rules always win:

```json
{
  "workflow_policy": {
    "allow": [
      "cloudformation:deploy",
      "s3api:put-object"
    ],
    "deny": [
      "*:delete-*",
      "ec2:terminate-*"
    ],
    "max_steps": 20
  }
}
```

Limits: rules use lowercase `service:operation` with `*` as the only wildcard. At most 100 allow and 100 deny rules. `max_steps` is 1 to 100, default 20. Workflow files are at most 1 MiB.

## Configure

The default file is `aws-clip/config.json` in your platform's user configuration directory:

- Linux and other Unix: `$XDG_CONFIG_HOME`, or `$HOME/.config`
- macOS: `$HOME/Library/Application Support`
- Windows: `%AppData%`

Use `--config PATH` or `AWS_CLIP_CONFIG_FILE` for another file. The default file may be absent, but an explicit one must exist. Unknown keys and trailing JSON are errors.

Precedence, highest first: wrapper flag, `AWS_CLIP_*` environment, user configuration, wrapper default. For Region, `AWS_REGION`, `AWS_DEFAULT_REGION`, and the AWS profile come after that. Unset retry and timeout settings defer to AWS CLI.

| Wrapper flag | Environment variable | JSON key | Effect |
| --- | --- | --- | --- |
| `--aws-binary PATH` | `AWS_CLIP_AWS_BINARY` | `aws_binary` | AWS CLI v2 executable, default `aws` |
| `--profile NAME` | `AWS_CLIP_PROFILE` | `profile` | Select AWS profile |
| `--region REGION` | `AWS_CLIP_REGION` | `region` | Set `AWS_REGION` for AWS CLI |
| `--retry-mode MODE` | `AWS_CLIP_RETRY_MODE` | `retry_mode` | `legacy`, `standard`, or `adaptive` |
| `--max-attempts NUMBER` | `AWS_CLIP_MAX_ATTEMPTS` | `max_attempts` | Set total AWS request attempts |
| `--connect-timeout SECONDS` | `AWS_CLIP_CONNECT_TIMEOUT` | `connect_timeout_seconds` | Set AWS socket connection timeout, `0` disables |
| `--read-timeout SECONDS` | `AWS_CLIP_READ_TIMEOUT` | `read_timeout_seconds` | Set AWS socket read timeout, `0` disables |

`command_policy`, `protected_profiles`, and `workflow_policy` are file-only.

When any standard stream isn't a terminal, `aws-clip` sets `AWS_PAGER` to empty and `AWS_CLI_AUTO_PROMPT=off` so automation can't block.

## Limits

- AWS command output can contain sensitive service data. Protect it like direct AWS CLI output.
- The identity check and the service command are separate processes, so an external credential provider can change between them.
- Workflows are sequential, with no variables, conditionals, output chaining, concurrency, or rollback. Multi-step stdout is each command's output joined together. Use `exec` for one machine-readable response.

## Exit statuses

| Status | Meaning |
| --- | --- |
| `0` | The check or operation passed |
| AWS CLI status | An AWS CLI command started and failed, status unchanged |
| `2` | Invalid syntax, configuration, profile, workflow, or acknowledgement |
| `3` | Command or workflow policy, or credential isolation, blocked execution |
| `4` | Required command confirmation was refused or cancelled |
| `5` | The verified AWS account didn't match an account constraint |
| `126` | AWS CLI couldn't be validated or started, or returned invalid local data |
| `127` | AWS CLI executable not found |

## Develop

```sh
make check
make check-models   # needs Python; compares the sensitive-output list with the pinned AWS models
```

After a review changes `tools/genops/reviewed.json` or the pinned `botocore`, run `make generate`.

Releases follow neon1's [release levels](https://github.com/neon1llc/neon1-corpus/blob/main/corpus/standards/versioning.md#release-levels): Implemented, Verified, Released, and Qualified. CI is the evidence for Verified. Qualified needs use outside my machines, following only this README.

## License

[MIT](LICENSE)

## v1.0 readiness

[Exit criteria and current evidence](docs/v1-readiness.md).
