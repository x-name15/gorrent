# Security Policy

## Supported Versions

Gorrent is actively developed. Security fixes are generally applied only to the latest stable release.

| Version                | Supported      |
| ---------------------- | -------------- |
| Latest release         | ✅              |
| Previous major release | ⚠️ Best effort |
| Older releases         | ❌              |

If you are running an older version, please upgrade before reporting a vulnerability.

---

## Reporting a Vulnerability

If you believe you have found a security vulnerability in Gorrent, **please do not open a public GitHub Issue**.

Instead, report it privately by one of the following methods:

* GitHub Security Advisories (preferred): **Security → Report a vulnerability**

Please include as much information as possible:

* Gorrent version
* Operating system
* Configuration (if relevant)
* Steps to reproduce
* Proof-of-concept or exploit (if available)
* Impact assessment

---

## Response Process

After receiving a report, I will:

1. Acknowledge receipt within **72 hours**.
2. Investigate and reproduce the issue.
3. Work on a fix if the issue is confirmed.
4. Coordinate responsible disclosure.
5. Publish a patched release and security advisory when appropriate.

---

## Scope

Examples of issues that are considered security vulnerabilities include:

* Remote Code Execution (RCE)
* Authentication or authorization bypass
* Privilege escalation
* Information disclosure
* Persistent denial of service caused by malformed APIs
* Memory corruption or unsafe parsing vulnerabilities in the search engine

The following generally **are not** considered security vulnerabilities:

* Crashes caused only by local users with filesystem access
* Missing TLS termination when running behind a reverse proxy
* Performance issues without security impact
* Feature requests
* Configuration mistakes made by users (e.g. running the dashboard on `0.0.0.0` without setting `REDB_DASHBOARD_USER`/`REDB_DASHBOARD_PASS`)

---

## Supported Deployments

Gorrent is intended to run inside trusted environments such as:

* Docker
* Kubernetes
* Internal infrastructure
* Edge deployments
* Home labs

Users deploying Gorrent directly on the public Internet are responsible for providing appropriate security controls such as:

* TLS
* Authentication
* Firewalls
* Reverse proxies
* Network isolation

---

Thank you for helping make Gorrent more secure.