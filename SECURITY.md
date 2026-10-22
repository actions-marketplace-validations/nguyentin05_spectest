# Security Policy

SpecTest is committed to ensuring the security and privacy of our users, developers, and the open-source software ecosystem. We take all security vulnerabilities seriously.

## Supported Versions

Only the latest active release branch receives security updates and patches.

| Version | Supported          |
| ------- | ------------------ |
| 1.x     | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

If you discover a security vulnerability or potential threat within SpecTest, please report it **responsibly**. Do **NOT** disclose the vulnerability publicly via GitHub Issues, Discussions, or social media.

### Private Reporting Channels

1. **GitHub Private Vulnerability Reporting**: You can report security advisories directly through GitHub Security Advisories at [https://github.com/nguyentin05/spectest/security/advisories/new](https://github.com/nguyentin05/spectest/security/advisories/new).
2. **Direct Security Email**: Send encrypted or direct email detailing the findings to:
   - Primary: [tin.nguyen.cs05@gmail.com](mailto:tin.nguyen.cs05@gmail.com)
   - Subject: `[SECURITY] SpecTest Vulnerability Report - <Brief Summary>`

### What to Include in Your Report

To help us triage and resolve the issue quickly, please provide:
* Description of the vulnerability and its potential impact.
* Steps to reproduce the issue (PoC script, test payloads, or curl commands).
* Environment details (Operating system, Go version, SpecTest version).
* Any possible mitigations or suggestions.

### Our Commitment

* **Initial Response**: Within 48 hours of receiving the report, our core team will acknowledge receipt.
* **Assessment & Fix**: We will assess the severity, collaborate with the reporter on a timeline, and prepare a patched release.
* **Public Disclosure**: Once a patch is released, we will publish a security advisory and credit the reporter (unless anonymity is requested).

## Security Best Practices for Users

* **Credentials Storage**: By default, `spectest config` stores local credentials in `~/.spectest/credentials.yaml` with file permission `0600` (readable only by the current user).
* **Token Redaction**: SpecTest automatically masks sensitive headers such as `Authorization: Bearer ***` in public CLI logs and HTML reports.
* **Isolated Environment**: When testing untrusted services or fuzzing APIs, execute SpecTest inside containers or ephemeral CI environments.
