# AWS CLI Plus

AWS CLI Plus (`aws-clip`) makes multi-profile AWS CLI work safer. It finds the profiles AWS CLI v2 already knows, logs in through each profile's own path, checks the active account and principal, and guards risky commands by account.

It doesn't replace AWS CLI. AWS CLI v2 still handles configuration, credentials, IAM Identity Center, login, service coverage, retries, requests, and output. Use `aws` directly for ordinary one-off work. Use `aws-clip` when profile isolation, identity checks, production account guards, or explicit approval lower your risk.

## Install

You need AWS CLI v2 on `PATH` (or its path set in config) and Go 1.22 or newer to build.

```sh
make build
./bin/aws-clip doctor
```

Or install into your Go binary directory:

```sh
go install ./cmd/aws-clip
aws-clip doctor
```

## Start with a profile

List the profiles AWS CLI has already merged from its configuration:

```sh
aws-clip profiles
aws-clip profiles --format json
```

The text view shows Region, authentication path, SSO account and role when configured, selection, and protection status. It never prints access key IDs, credential-process commands, SSO start URLs, login-session names, or credential values. The JSON response has `schema_version: 1` and an ordered `profiles` array. An unavailable Region is `null`.

`aws-clip` doesn't create profiles. Create one with AWS CLI, then select it explicitly:

```sh
aws configure sso --profile development
aws-clip --profile development login
aws-clip --profile development context
```

You can also select a profile with `AWS_CLIP_PROFILE` or the configuration file. Login and guarded commands never fall back to AWS CLI's default profile.

## Log in and out

`login` asks STS whether the profile already has valid credentials and reuses them. Otherwise it follows the profile's metadata, including `source_profile` chains:

- IAM Identity Center profiles run `aws sso login` for the owning profile.
- AWS local login profiles run `aws login` for the owning profile.
- Role profiles use their source profile's login path.
- Configured profiles with no declared provider can bootstrap AWS local login after an identity probe fails, if your AWS CLI supports it.
- Static keys, credential processes, web identity, instance or container roles, and the default provider chain stay with their own provider.

After login, `aws-clip` checks STS identity again and reports the account and principal. It never exports or stores session credentials.

```sh
aws-clip --profile production login
aws-clip --profile development logout
```

Local-login logout affects only its login profile. AWS CLI's SSO logout clears every cached SSO session, so `aws-clip` asks you to say so:

```sh
aws-clip --profile development logout --approve all-sso-sessions
```

Profiles backed by external or static providers have no login or logout action. The message names the provider to refresh.

## Verify the active context

`context` calls `aws sts get-caller-identity` through the selected profile and reports non-secret context only:

```sh
aws-clip --profile production --region us-east-1 context
aws-clip --profile production context --format json
```

It reports profile, observed account ID, principal ARN, user ID, role or user name, effective Region, credential source, source profile, and account-guard state. JSON output is a stable `schema_version: 1` contract. Safety decisions use the observed STS account, not the profile name.

Region precedence: wrapper flag, `AWS_CLIP_REGION`, user configuration, `AWS_REGION`, `AWS_DEFAULT_REGION`, then the AWS profile. Unset retry and timeout settings defer to AWS CLI.

## Guarded commands

`exec` is stricter than `aws --profile`. It requires an explicit profile, rejects arguments that change context, classifies the operation, applies local policy, and prints the profile, Region, classification, and decision on stderr. Then it passes literal arguments and standard streams to AWS CLI:

```sh
aws-clip --profile development exec -- ec2 describe-instances --output json
```

The `--` separator is mandatory, and nothing after it goes through a shell. Set `--profile` and `--region` before `exec`. After the separator, `--profile`, `--region`, `--endpoint-url`, and `--no-sign-request` are rejected because they would change the resolved target.

Only documented non-mutating families count as read-only: `list-*`, `describe-*`, `get-*`, `head-*`, `lookup-*`, `search-*`, `validate-*`, and `wait`. Known mutations and unrecognized operations need confirmation. Interactive use prompts with the target context. Non-interactive use fails unless you pass `--yes`. Known credential-, token-, and secret-returning operations need the same confirmation even when the name looks read-only:

```sh
aws-clip --profile development exec -- ec2 terminate-instances --instance-ids i-example
aws-clip --profile development exec --yes -- \
  ec2 terminate-instances --instance-ids i-example
```

`--preview` is local. It resolves the executable path but starts no AWS CLI process, runs no identity check, and prints no option values. A Region inherited only from the AWS profile shows as unresolved, so pass `--region` if the preview must check a Region allowlist.

```sh
aws-clip --profile development --region us-east-1 exec --preview -- \
  secretsmanager put-secret-value --secret-id example --secret-string hidden
```

The preview shows only `secretsmanager:put-secret-value` and the option names `--secret-id` and `--secret-string`.

Classification is conservative and name-based, so a new, aliased, or ambiguous operation counts as mutating. Policy and confirmation add to IAM, service control policies, budgets, backups, and change management. They don't judge argument values or remote state.

### Restrict direct commands

Command policy lives in the file only. Deny rules always win. When `allow` isn't empty, the command must match it, and so must each non-empty target allowlist.

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

An account allowlist runs `sts get-caller-identity` just before confirmation and the command. `--yes` can't override an account mismatch. Without an account allowlist or protected profile, ordinary execution doesn't call STS. `--approve-account` also requests the identity check and requires a matching account. It doesn't replace `--yes` for non-interactive changes.

### Protect production profiles

Bind a high-risk profile to its expected 12-digit account in the configuration file:

```json
{
  "profile": "production",
  "region": "us-east-1",
  "protected_profiles": {
    "production": "123456789012"
  }
}
```

Every live `context` or `exec` check fails with the identity-mismatch status if `production` resolves to another account. On the right account, every non-read operation still needs interactive confirmation or `--yes`. Add `--approve-account 123456789012` for an extra identity-bound check. Protection is file-only, so an environment variable or flag can't weaken it.

For isolation, `context`, `login`, `exec`, and workflow runs block credential environment variables such as `AWS_ACCESS_KEY_ID`, `AWS_SESSION_TOKEN`, web-identity settings, and container credential endpoints. Only the names are reported. Clear them when the profile should be authoritative. If you want environment credentials, use AWS CLI directly.

## Doctor

```sh
aws-clip doctor
aws-clip --profile production doctor
```

`doctor` checks AWS CLI v2, resolves the aws-clip configuration, lists configured profiles, and looks for credential environment conflicts. With an explicit profile it also reports the authentication path and Region, verifies identity through STS, and validates any protected account binding. Failed checks name the next command to run. AWS diagnostic text and exit status stay available when a subprocess fails.

## Reviewed workflows

A Makefile or script is usually simpler for plain repeatability. Use an aws-clip workflow only when you want its strict schema, policy review, named approval, identity check, account guard, literal arguments, fail-fast execution, and per-step visibility.

A workflow is a versioned JSON file of named AWS CLI steps:

```json
{
  "schema_version": 1,
  "name": "release-check",
  "description": "Confirm identity and inspect the deployed stack",
  "steps": [
    {
      "name": "identity",
      "command": ["sts", "get-caller-identity"]
    },
    {
      "name": "stack",
      "command": [
        "cloudformation",
        "describe-stacks",
        "--stack-name",
        "application",
        "--output",
        "json"
      ]
    }
  ]
}
```

Preview is local and calls neither AWS CLI nor AWS:

```sh
aws-clip --profile production plan release-check.json
aws-clip --profile production plan --format json release-check.json
```

The plan shows effective context, protected account configuration, whether account approval is needed, each `service:operation`, and its policy decision. Argument values are omitted because they may hold sensitive data. JSON plans are a `schema_version: 1` contract with ordered steps.

Run it after reviewing the file and plan:

```sh
aws-clip --profile production run --approve release-check release-check.json
```

A run needs the exact workflow name, an explicit profile, and a live STS identity check. A destructive or sensitive-output step needs `--approve-account`, and so does any change on a protected profile. All policy and approvals are checked before the first step. Steps run in order and stop at the first failure. AWS output and diagnostics are kept, and the failing AWS status is returned unchanged. AWS CLI still handles request retries.

### Workflow policy

Read-oriented operations (`describe-*`, `get-*`, `head-*`, `list-*`, `lookup-*`, `search-*`, `validate-*`) are allowed by default. Known credential-, token-, and secret-returning operations are blocked even when the name looks read-only. Everything else needs a narrow allow rule. Deny rules always win:

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

Limits:

- Rules use lowercase `service:operation`, with `*` as the only wildcard.
- At most 100 allow and 100 deny rules.
- `max_steps` is 1 to 100 and defaults to 20.
- Workflow files are at most 1 MiB, commands at most 256 arguments and 64 KiB of argument data.
- Names and descriptions are printable single-line text.
- Unknown JSON fields and trailing data are rejected.

## Configure

The default file is `aws-clip/config.json` in your platform's user configuration directory:

- Linux and other Unix: `$XDG_CONFIG_HOME`, or `$HOME/.config`
- macOS: `$HOME/Library/Application Support`
- Windows: `%AppData%`

Use `--config PATH` or `AWS_CLIP_CONFIG_FILE` to pick another file. The default file may be absent, but an explicitly selected file must exist. Unknown keys and trailing JSON are errors.

Precedence, highest first: wrapper flag, `AWS_CLIP_*` environment, user configuration, wrapper default.

| Wrapper flag | Environment variable | JSON key | Effect |
| --- | --- | --- | --- |
| `--aws-binary PATH` | `AWS_CLIP_AWS_BINARY` | `aws_binary` | AWS CLI v2 executable, default `aws` |
| `--profile NAME` | `AWS_CLIP_PROFILE` | `profile` | Select AWS profile |
| `--region REGION` | `AWS_CLIP_REGION` | `region` | Set `AWS_REGION` for AWS CLI |
| `--retry-mode MODE` | `AWS_CLIP_RETRY_MODE` | `retry_mode` | `legacy`, `standard`, or `adaptive` |
| `--max-attempts NUMBER` | `AWS_CLIP_MAX_ATTEMPTS` | `max_attempts` | Set total AWS request attempts |
| `--connect-timeout SECONDS` | `AWS_CLIP_CONNECT_TIMEOUT` | `connect_timeout_seconds` | Set AWS socket connection timeout, `0` disables |
| `--read-timeout SECONDS` | `AWS_CLIP_READ_TIMEOUT` | `read_timeout_seconds` | Set AWS socket read timeout, `0` disables |

`command_policy`, `protected_profiles`, and `workflow_policy` are file-only safety settings. Each command-policy list takes at most 100 entries. For automation, pin and protect the configuration file.

When any standard stream isn't a terminal, `aws-clip` sets `AWS_PAGER` to an empty value and `AWS_CLI_AUTO_PROMPT=off` so automation can't block on a pager or prompt. Interactive runs leave both unchanged.

## Limits

- `aws-clip` never keeps, writes, prints, exports, or caches credential values.
- AWS command stdout and stderr belong to AWS and can contain sensitive service data. Protect them like direct AWS CLI output.
- Classification uses safe configuration metadata and operation names. IAM and AWS Organizations stay authoritative.
- Direct commands call STS only for an account allowlist, protected profile, or `--approve-account`. Workflows always check identity first.
- The identity check and the service command are separate AWS CLI processes, so an external credential provider can change between them.
- Environment-backed credentials are outside profile isolation.
- There's no interactive profile selector, browser-console launcher, or credential exporter.
- Workflows are sequential. They have no variables, conditionals, output chaining, concurrency, rollback, or remote orchestration.
- Multi-step workflow stdout is each command's output joined together. Use `exec` for one machine-readable response.

## Exit statuses

| Status | Meaning |
| --- | --- |
| `0` | Requested check or operation succeeded |
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
```

This checks formatting, runs static analysis, and runs a hermetic process-level suite. Tests use a local fake executable and never contact AWS.

Releases follow neon1's four [release levels](https://github.com/neon1llc/neon1-corpus/blob/main/corpus/standards/versioning.md#release-levels): Implemented, Verified, Released, and Qualified. CI's checks are the evidence for Verified. Qualified needs use outside the author's machine, following only this README.

## License

[MIT](LICENSE)
