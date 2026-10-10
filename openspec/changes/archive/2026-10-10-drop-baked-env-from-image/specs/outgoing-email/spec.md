## MODIFIED Requirements

### Requirement: Instance mail server configuration

The instance mail server SHALL be configured through environment variables: `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM`, `SMTP_SECURITY`, `SMTP_HELO_NAME` and `SMTP_INSECURE_SKIP_VERIFY`. `SMTP_INSECURE_SKIP_VERIFY` has no effect in the `none` mode. A variable the operator did not set, or set to an empty value, SHALL count as unset.

When `SMTP_SECURITY` is not set, the mode SHALL be `tls` for port 465 and `starttls` for every other port. When `SMTP_USER` or `SMTP_PASSWORD` is not set, instance mail SHALL be delivered without authentication. The instance SHALL refuse to start, with a message naming the variable, when `SMTP_HOST` is set without a positive `SMTP_PORT`, when `SMTP_SECURITY` holds a value other than `tls`, `starttls` or `none`, when `SMTP_HELO_NAME` is set to something that is neither a host name nor an address literal, or when `SMTP_FROM` is set to something that is neither an email address nor a display name followed by an address in angle brackets.

#### Scenario: Security mode left unset on port 587

- **WHEN** the instance starts with `SMTP_HOST` set, `SMTP_PORT=587` and no `SMTP_SECURITY`
- **THEN** instance mail is delivered in the `starttls` mode

#### Scenario: Security mode left unset on port 465

- **WHEN** the instance starts with `SMTP_HOST` set, `SMTP_PORT=465` and no `SMTP_SECURITY`
- **THEN** instance mail is delivered in the `tls` mode

#### Scenario: Relay without credentials

- **WHEN** the published image starts with only `SMTP_HOST`, `SMTP_PORT=25` and `SMTP_FROM` set, against a relay that offers STARTTLS but no authentication
- **THEN** instance mail is delivered in the `starttls` mode without any authentication attempt

#### Scenario: Unknown security mode

- **WHEN** the instance starts with `SMTP_SECURITY=ssl`
- **THEN** it refuses to start and names `SMTP_SECURITY` and the accepted values

#### Scenario: Unreadable sender address

- **WHEN** the instance starts with `SMTP_FROM=noreply at example.com`
- **THEN** it refuses to start and names `SMTP_FROM`

#### Scenario: Invalid greeting name

- **WHEN** the instance starts with `SMTP_HELO_NAME=my relay`
- **THEN** it refuses to start and names `SMTP_HELO_NAME`

## ADDED Requirements

### Requirement: Instance mail settings come only from the operator

The instance mail server SHALL use only values the operator supplied, plus the defaults this specification defines. In the published image, no host, credential, sender or security mode SHALL come from a file shipped inside the image or from an environment file present on the machine that built it. In particular, an unset `SMTP_SECURITY` SHALL never resolve to `none`, so credentials are never sent unencrypted unless the operator chose `none` explicitly.

#### Scenario: Real credentials without a security mode

- **WHEN** the published image starts with `SMTP_HOST`, `SMTP_PORT=587`, `SMTP_USER` and `SMTP_PASSWORD` set and no `SMTP_SECURITY`
- **THEN** the connection is upgraded with STARTTLS before the credentials are sent, and if the server does not offer the upgrade, the delivery fails without sending them

#### Scenario: No mail settings at all

- **WHEN** the published image starts without any `SMTP_*` variable
- **THEN** the settings screen reports the mail server as not configured, the sign-in screen offers no password reset link, a test message from the settings screen is refused because no mail server is configured, and an invitation is logged as skipped instead of being sent

#### Scenario: Sender left unset

- **WHEN** the published image starts with `SMTP_HOST=relay.example.com`, `SMTP_PORT=25` and no `SMTP_USER` or `SMTP_FROM`
- **THEN** the envelope sender is `noreply@relay.example.com`, and `databasus.com` appears nowhere in the sender
