# Outgoing Email Specification

## Purpose

Describes how Databasus delivers email: the instance-wide mail server that sends invitations, password reset codes and sign-in codes, and the email notification channels that report backup events. It covers connection security, sender identity, message format, configuration, and the tools an administrator has to check the instance mail server.

## Requirements

### Requirement: One delivery behavior for instance mail and notification channels

The instance mail server and every email notification channel SHALL deliver mail with the same behavior. Given the same server, port, credentials, security mode, sender and greeting name, both SHALL open the same kind of connection, authenticate the same way, build the message the same way and report failures the same way.

#### Scenario: Same settings, same outcome

- **WHEN** the instance mail server and an email notification channel are configured with identical connection settings and the mail server rejects the credentials
- **THEN** both deliveries fail, and both report the mail server's rejection

### Requirement: Explicit connection security mode

Every delivery SHALL use one of three connection security modes:

- `tls`: the connection is encrypted from its first byte.
- `starttls`: the connection starts unencrypted and SHALL be upgraded to an encrypted one before any credential or message is sent. If the mail server does not offer the upgrade, the delivery SHALL fail.
- `none`: the connection is never encrypted.

In the `tls` and `starttls` modes the mail server's certificate SHALL be verified unless the operator has explicitly turned verification off.

#### Scenario: Encrypted from the start

- **WHEN** a delivery uses `tls` against a server that expects encryption from the first byte
- **THEN** the message is delivered over an encrypted connection

#### Scenario: Upgrade offered

- **WHEN** a delivery uses `starttls` and the server offers the upgrade
- **THEN** the connection is upgraded before authentication, and the message is delivered

#### Scenario: Upgrade not offered

- **WHEN** a delivery uses `starttls` and the server does not offer the upgrade
- **THEN** the delivery fails with an error saying the server does not offer STARTTLS, and nothing after the greeting is sent to the server, not even `QUIT`

#### Scenario: Deliberately unencrypted

- **WHEN** a delivery uses `none`
- **THEN** the message is delivered without encryption, with authentication if credentials are configured

#### Scenario: Untrusted certificate

- **WHEN** a delivery uses `tls` or `starttls` and the server presents a certificate that does not verify, and verification has not been turned off
- **THEN** the delivery fails before any credential is sent

### Requirement: Credentials and message content never cross an unencrypted connection by accident

A username, a password or any part of a message SHALL NOT be sent over an unencrypted connection unless the operator selected the `none` mode for that delivery. This SHALL hold even when an attacker between the instance and the mail server removes the server's offer to upgrade the connection.

#### Scenario: Upgrade offer stripped in transit

- **WHEN** a delivery configured for `starttls` with credentials reaches a server whose upgrade offer an attacker has removed from the conversation
- **THEN** the delivery fails, and neither the username nor the password nor any part of the message has been sent

### Requirement: Delivery errors never reveal the mail server password

An error produced by a delivery SHALL NOT contain the mail server password, wherever the error is shown: an API response, a notification channel's last-send error, the audit log or the server log.

#### Scenario: Rejected credentials

- **WHEN** the mail server rejects the configured credentials
- **THEN** the reported error states that authentication failed and includes the server's response, and it does not include the password

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

### Requirement: One answer to whether the instance mail server is configured

The instance mail server SHALL count as configured exactly when `SMTP_HOST` is set. `DATABASUS_URL` SHALL NOT affect that answer. The settings screen, the second-factor switch, the sign-in screen's password reset link and the backend's own checks SHALL all rely on this one answer.

#### Scenario: Mail server without a public address

- **WHEN** the instance runs with `SMTP_HOST` and `SMTP_PORT` set and `DATABASUS_URL` unset
- **THEN** the settings screen reports the mail server as configured, and the sign-in screen offers the password reset link

#### Scenario: No mail server

- **WHEN** the instance runs without `SMTP_HOST`
- **THEN** the settings screen reports the mail server as not configured, and the sign-in screen does not offer the password reset link

### Requirement: Sender identity

Every message SHALL name its sender as a display name and an address. The envelope sender that the mail server sees SHALL be the address alone.

A sender SHALL be accepted either as a bare address (`noreply@example.com`) or as a display name followed by an address in angle brackets (`Acme Backups <noreply@example.com>`). When the sender carries no display name, the display name SHALL be `Databasus`.

For instance mail the sender SHALL be `SMTP_FROM`; when it is not set, `SMTP_USER` if that is a bare email address; otherwise `noreply@` followed by `SMTP_HOST`. For a notification channel it SHALL follow the same order over the channel's own sender, username and host. A channel whose sender is set but cannot be read in either accepted form SHALL be refused when it is saved.

#### Scenario: Display name in the sender

- **WHEN** the instance runs with `SMTP_FROM` set to `Acme Backups <noreply@example.com>`
- **THEN** recipients see the sender as `Acme Backups <noreply@example.com>`, and the mail server receives `noreply@example.com` alone as the envelope sender

#### Scenario: Bare sender address

- **WHEN** the instance runs with `SMTP_FROM=noreply@example.com`
- **THEN** recipients see the sender as `Databasus <noreply@example.com>`

#### Scenario: Username that is not an address

- **WHEN** the instance runs with `SMTP_HOST=smtp.sendgrid.net`, `SMTP_USER=apikey` and no `SMTP_FROM`
- **THEN** the envelope sender is `noreply@smtp.sendgrid.net`, and `apikey` appears nowhere in the sender

#### Scenario: Non-ASCII display name

- **WHEN** the display name in the sender contains characters outside ASCII
- **THEN** recipients see the name exactly as configured

#### Scenario: Notification channel with an unreadable sender

- **WHEN** a user saves an email notification channel whose sender is `ops at example`
- **THEN** the save is refused with a message that the sender must be an address, optionally preceded by a display name

### Requirement: Greeting name

Every delivery SHALL introduce itself to the mail server with a greeting name. That name SHALL be the configured greeting name (`SMTP_HELO_NAME` for instance mail, the channel's own setting for a notification channel). When none is configured, it SHALL be the host name from `DATABASUS_URL`, or the machine's host name when `DATABASUS_URL` is not set, or `localhost` when the host name is unavailable. When the name is an IP address, it SHALL be sent as an address literal: `[192.0.2.1]` for IPv4 and `[IPv6:2001:db8::1]` for IPv6. The greeting SHALL use the same name in every security mode.

A configured greeting name SHALL be either a host name or an address literal in that form.

#### Scenario: Greeting derived from the public address

- **WHEN** the instance runs with `DATABASUS_URL=https://backup.example.com` and no greeting name is configured
- **THEN** deliveries greet the mail server as `backup.example.com`

#### Scenario: Explicit greeting name

- **WHEN** `SMTP_HELO_NAME=mail.example.com` is set
- **THEN** instance mail greets the mail server as `mail.example.com`

#### Scenario: Public address given as an IPv6 address

- **WHEN** the instance runs with `DATABASUS_URL=http://[2001:db8::1]:4005` and no greeting name is configured
- **THEN** deliveries greet the mail server as `[IPv6:2001:db8::1]`

### Requirement: Delivery time limit

A single delivery SHALL finish or fail within 30 seconds of its start, counting the whole conversation with the mail server and not only opening the connection.

#### Scenario: Server goes silent

- **WHEN** the mail server accepts the connection and then stops responding
- **THEN** the delivery fails with a timeout error within 30 seconds, and the request that triggered it completes

### Requirement: Well-formed messages

Every message SHALL carry a `From`, `To`, `Subject`, `Date`, `Message-ID`, `MIME-Version`, `Content-Type` and `Content-Transfer-Encoding` header. The `Message-ID` SHALL be unique per message. A subject or a name containing characters outside ASCII SHALL reach the recipient intact without requiring the server to support UTF-8 addressing. The body SHALL be encoded so that no line on the wire exceeds the limit mail servers enforce, whatever the length of the lines in the original text.

#### Scenario: Instance mail carries a message identifier

- **WHEN** the instance sends an invitation
- **THEN** the delivered message has a `Message-ID` header that differs from every other message the instance has sent

#### Scenario: Long body line

- **WHEN** a message body contains a single line of 5,000 characters
- **THEN** the recipient reads the body exactly as it was written

### Requirement: Header values cannot add headers

No value placed in a header SHALL be able to add a header or end the header block. This covers subjects built from user-supplied text such as a workspace name, the recipient address, and the sender's display name and address.

#### Scenario: Line break in a workspace name

- **WHEN** a workspace is named `Ops\r\nBcc: attacker@example.com` and an invitation to it is sent
- **THEN** the delivered message has no `Bcc` header, and the recipient list holds only the invited address

### Requirement: Failed instance mail is logged

When instance mail cannot be delivered for an invitation, the invitation SHALL still be created, and the failure SHALL be written to the server log with the reason. When the instance mail server is not configured, an attempted send SHALL be written to the server log as skipped.

#### Scenario: Invitation with a broken mail server

- **WHEN** an administrator invites a user while the mail server rejects every message
- **THEN** the invitation is created, and the server log records that the invitation email failed and why

### Requirement: Notification channel connection options

An email notification channel SHALL let its owner choose the connection security mode and the greeting name. These options SHALL sit in the channel form's advanced settings next to the existing certificate verification switch. The default mode for a port SHALL be `tls` for 465 and `starttls` for every other port. When the port changes in the form, the security mode SHALL follow the new port's default only if it equals the previous port's default; any other mode SHALL be kept.

A request that omits the security mode SHALL be accepted: a new channel SHALL be saved with its port's default, and an updated channel SHALL keep its stored mode. A request naming an unknown mode SHALL be refused. A channel SHALL be refused on save when its recipient is not a single email address or its greeting name is neither a host name nor an address literal.

Channels that existed before this capability SHALL be given `tls` if their port is 465 and `starttls` otherwise, with no greeting name.

#### Scenario: New channel on port 465

- **WHEN** a user creates an email notification channel, enters port 465 and does not touch the security mode
- **THEN** the form proposes `tls`, and the channel is saved with `tls`

#### Scenario: Port change keeps a deliberate choice

- **WHEN** the owner of a channel stored with `none` on port 25 changes its port to 2525 and saves
- **THEN** the channel is saved with `none`

#### Scenario: API client that does not know the security mode

- **WHEN** a client creates an email notification channel on port 587 without a security mode, and later updates it without one
- **THEN** the channel is created with `starttls`, and the update keeps `starttls`

#### Scenario: Existing channel after upgrade

- **WHEN** an instance upgrades with an email notification channel on port 587
- **THEN** the channel uses `starttls`, and its advanced settings show that mode

#### Scenario: Existing channel pointed at a server without STARTTLS

- **WHEN** an upgraded channel on port 25 points at a server that does not offer STARTTLS
- **THEN** its next notification fails, the channel shows the failure as its last send error, and it delivers again once its owner selects `none`

### Requirement: Mail server status on the settings screen

The settings screen SHALL show administrators a mail server block directly above the audit logs. The block SHALL say whether the instance mail server is configured and link to the SMTP section of the published configuration documentation. It SHALL state that instance mail covers invitations, password reset codes and sign-in codes, and that email notification channels use SMTP settings of their own. It SHALL NOT show the host, port, username, sender or any other part of the configuration.

#### Scenario: Configured

- **WHEN** an administrator opens the settings screen on an instance with `SMTP_HOST` set
- **THEN** the mail server block says the mail server is configured

#### Scenario: Not configured

- **WHEN** an administrator opens the settings screen on an instance without `SMTP_HOST`
- **THEN** the mail server block says the mail server is not configured and links to the documentation that explains how to connect one

### Requirement: Test message from the settings screen

The mail server block SHALL offer a button that sends a test message to the signed-in administrator's own address. The button SHALL be unavailable while the mail server is not configured. After a send, the screen SHALL either confirm the delivery and name the address it went to, or show the error the delivery produced, including the mail server's response.

The instance SHALL refuse a test send, and say why, when the mail server is not configured or when the administrator's account has no valid email address. It SHALL NOT fall back to skipping the send silently.

#### Scenario: Successful test

- **WHEN** an administrator presses the test button with a working mail server
- **THEN** a test message arrives at the administrator's address, and the screen confirms which address it was sent to

#### Scenario: Mail server rejects the login

- **WHEN** an administrator presses the test button and the mail server rejects the credentials
- **THEN** the screen shows that authentication failed together with the mail server's response

#### Scenario: Mail server not configured

- **WHEN** a test send is requested on an instance without `SMTP_HOST`
- **THEN** the request is refused with an answer saying the mail server is not configured, and nothing is sent

#### Scenario: Account without a usable address

- **WHEN** an administrator whose account holds no valid email address requests a test send
- **THEN** the request is refused with an answer saying the account has no address to send to

### Requirement: Test message access, abuse limits and audit

Only administrators SHALL be able to request a test message. Each administrator SHALL be limited to five test messages in ten minutes. Every test send the instance attempts, whatever stage it fails at, SHALL be recorded in the audit log with the administrator, the recipient address and whether it was delivered.

#### Scenario: Member requests a test message

- **WHEN** a signed-in user who is not an administrator requests a test send
- **THEN** the request is refused as forbidden, and nothing is sent

#### Scenario: Too many test messages

- **WHEN** an administrator requests a sixth test send within ten minutes
- **THEN** the request is refused as rate-limited, and nothing is sent

#### Scenario: Failed test in the audit log

- **WHEN** an administrator's test send fails at the mail server
- **THEN** the audit log shows an entry naming the administrator, the recipient address and that the delivery failed

### Requirement: Published mail server documentation

The SMTP section of the published configuration documentation SHALL describe every instance mail variable, the three security modes and the default mode for each port, the rule that decides whether the mail server counts as configured, and the difference between the instance mail server and an email notification channel. It SHALL warn that `none` sends credentials and messages unencrypted. It SHALL warn that an instance requiring the emailed second factor admits nobody through password sign-in while its mail server cannot deliver, tell operators on a relay without STARTTLS to set `SMTP_SECURITY` before upgrading, and name the console command that switches the second factor off. The section SHALL be published in every language the website ships and keep the same anchor in each.

#### Scenario: Operator migrating a plain relay

- **WHEN** an operator whose relay on port 25 does not offer STARTTLS reads the SMTP section in any published language
- **THEN** the section tells them to set `SMTP_SECURITY=none` before upgrading, warns what that setting exposes, and warns that with the second factor on a missed setting locks password sign-in until it is fixed or the second factor is switched off
