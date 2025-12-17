# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in Postgresus, please report it responsibly. **Do not create a public GitHub issue for security vulnerabilities.**

### How to Report

1. **Email** (preferred): Send details to [info@postgresus.com](mailto:info@postgresus.com)
2. **Telegram**: Contact [@rostislav_dugin](https://t.me/rostislav_dugin)
3. **GitHub Security Advisories**: Use the [private vulnerability reporting](https://github.com/RostislavDugin/postgresus/security/advisories/new) feature

### What to Include

- Description of the vulnerability
- Steps to reproduce the issue
- Potential impact and severity assessment
- Any suggested fixes (optional)

## Supported Versions

| Version | Supported |
| ------- | --------- |
| Latest  | Yes       |

We recommend always using the latest version of Postgresus. Security patches are applied to the most recent release.

### PostgreSQL Compatibility

Postgresus supports PostgreSQL versions 12, 13, 14, 15, 16, 17 and 18.

## Response Timeline

- **Acknowledgment**: Within 48-72 hours
- **Initial Assessment**: Within 1 week
- **Fix Timeline**: Depends on severity, but we aim to address critical issues as quickly as possible

We follow a coordinated disclosure policy. We ask that you give us reasonable time to address the vulnerability before any public disclosure.

## Security Features

Postgresus is designed with security in mind. For full details, see our [security documentation](https://postgresus.com/security).

Key features include:

- **AES-256-GCM Encryption**: Enterprise-grade encryption for backup files and sensitive data
- **Read-Only Database Access**: Postgresus uses read-only access by default and warns if write permissions are detected
- **Role-Based Access Control**: Assign viewer, member, admin or owner roles within workspaces
- **Audit Logging**: Track all system activities and changes made by users
- **Zero-Trust Storage**: Encrypted backups are safe even in shared cloud storage

## License

Postgresus is licensed under [Apache 2.0](../LICENSE).