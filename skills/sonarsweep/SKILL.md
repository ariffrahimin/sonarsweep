---
name: sonarsweep
description: Fetch, triage, and remediate SonarQube issues, code smells, bugs, and security vulnerabilities without needing sonar-scanner CLI. Use when the user mentions SonarQube, sonarsweep, quality gate failures, sonar issues, or fixing static analysis findings.
---

# SonarSweep — AI SonarQube Issue Triage & Remediation

Extract open SonarQube issues and rule guidance directly from the SonarQube server API (via `sonarsweep` or `curl`) without running `sonar-scanner`, then remediate findings in the local repository.

## Step 1: Discover Project & Fetch Issues

Resolve the target project and retrieve open issues as structured JSON:

1. **Primary path (`sonarsweep` installed)**:
   - Run `sonarsweep --json -s BLOCKER,HIGH` (auto-detects `sonar.projectKey` and `sonar.host.url` from `sonar-project.properties`, or uses `~/.config/sonarsweep/config.json` and the OS keychain token).
   - Pass `-p <project_key>` to target a specific project, `-b <branch>` for a branch, `--pr <id>` for a pull request, or `--overall` to include issues outside the new code period.
   - If `BLOCKER,HIGH` returns 0 issues and the user asked for all severities, re-run with `-s BLOCKER,HIGH,MEDIUM,LOW`.

2. **Fallback path (`sonarsweep` not installed)**:
   - Resolve `SONAR_URL` and `PROJECT_KEY` from `sonar-project.properties` or `~/.config/sonarsweep/config.json`.
   - Resolve `USER_TOKEN` from the environment or OS keychain:
     - macOS: `security find-generic-password -s sonarsweep -a user_token -w 2>/dev/null`
     - Linux: `secret-tool lookup service sonarsweep username user_token 2>/dev/null`
   - Query the SonarQube Web API directly:
     ```bash
     curl -s -u "${USER_TOKEN}:" \
       "${SONAR_URL}/api/issues/search?componentKeys=${PROJECT_KEY}&statuses=OPEN,CONFIRMED&impactSeverities=BLOCKER,HIGH&ps=500"
     ```

**Completion criterion**: You have the list of open issues with repo-relative `file` paths, `line` numbers, `rule` keys, `impact_severity`, `software_quality`, and `message`.

## Step 2: Triage by Severity, Quality & File Locality

1. Order issues by impact severity (`BLOCKER` → `HIGH` → `MEDIUM` → `LOW`) and software quality (`SECURITY` → `RELIABILITY` → `MAINTAINABILITY`).
2. Group issues by `file` so all findings in a single file are fixed together in one pass.
3. For any unfamiliar `rule` key where the required fix is non-obvious from `message`, fetch the rule remediation spec before editing:
   - With `sonarsweep`: `sonarsweep --explain-rule <rule_key>`
   - Fallback with `curl`: `curl -s -u "${USER_TOKEN}:" "${SONAR_URL}/api/rules/show?key=<rule_key>"`

**Completion criterion**: Every targeted issue is grouped by file and its rule requirement is understood.

## Step 3: Remediate & Verify

For each grouped file:

1. Read the target `file` around the reported `line`.
2. Apply the minimal code change that satisfies the SonarQube rule without altering external behavior.
3. Run the repository's test suite and linter/typechecker after each logical batch.

**Completion criterion**: Every targeted SonarQube issue is fixed in code with all local tests and build checks passing, or explicitly documented as an intentional exception with rationale.
