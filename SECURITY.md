# Security policy

## Reporting a vulnerability

Do not open a public issue for a suspected vulnerability. Use GitHub's private
vulnerability reporting for this repository so the report and any proof of
concept remain confidential while they are assessed.

Include the affected version or commit, operating system, configuration or
backend involved, expected impact, reproduction steps, and any proposed
mitigation. Remove access tokens, credentials, personal files, and production
endpoint details from the report.

Zclone is an independently maintained derivative of rclone. Report issues in
Zclone-specific code or behavior here. If the issue is demonstrably present in
an unmodified current rclone release, follow the upstream project's security
policy instead. Do not submit the same uncoordinated public disclosure to both
projects.

## Supported versions

This project currently supports the latest commit on `main`. There is no
separate long-term-support release line.

## Security boundaries

Zclone can read, copy, replace, and delete data using the permissions granted to
its configured storage credentials. Treat configuration files, RC credentials,
logs, and generated support bundles as sensitive. Preview destructive commands
with `--dry-run`, bind network services to trusted interfaces, and enable
authentication before exposing the RC API or a protocol server.
