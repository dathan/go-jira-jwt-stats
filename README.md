## Purpose

Generate a self-contained HTML stats dashboard for your team's open Jira work, scoped
across every project the team touches — not just the ones your team owns.

It answers three things at a glance:

* **What's open** — total unresolved issues, overdue count, unassigned count
* **Who is doing what** — open workload broken down by assignee
* **What's being asked of the team from outside your own project(s)** — issues assigned
  to your team members in projects other than your "home" project(s), surfaced separately
  from your own backlog

Authentication uses a headed Chrome browser session (SSO login, same pattern as scraping
any internal Atlassian instance) — no API token required. Session cookies are captured
once and reused via a saved curl file.

## Usage

```sh
go run ./cmd/go-jira-jwt-stats \
  --base-url https://jirasw.nvidia.com \
  --projects OPPEPROJ \
  --team-members jspragg,jcooper,agee,brusmith,alaing,avladimirov,brennenm,dcarr,dmateer,fengzhou,lgriffith,tflitsch,gvialetto \
  --open
```

On first run (no cached cookies), a Chrome window opens — log in via SSO, then the tool
captures your session and writes it to `conf/jira.curl` for reuse on future runs.

### Flags

| Flag | Env var | Description |
|---|---|---|
| `--base-url` | `JIRA_BASE_URL` | Jira base URL (default `https://jirasw.nvidia.com`) |
| `--team-members` | `JIRA_TEAM_MEMBERS` | Comma-separated Jira usernames to track |
| `--projects` | `JIRA_PROJECTS` | Comma-separated "home" project keys (e.g. `OPPEPROJ`) |
| `--jql` | `JIRA_JQL` | Full JQL override, replaces the generated query entirely |
| `--jql-extra` | `JIRA_JQL_EXTRA` | Extra clause ANDed onto the generated JQL |
| `--curl-file` | `JIRA_CURL_FILE` | Path to a curl command copied from DevTools (or `-` for stdin) |
| `--cookies` | `JIRA_COOKIES` | Raw cookie string, alternative to `--curl-file` |
| `--output` / `--output-dir` | | Where to write the dashboard HTML |
| `--open` | | Open the dashboard in a browser once written |

Without `--jql`, the tool builds:

```
assignee in (<team-members>) AND resolution = Unresolved ORDER BY due ASC, created ASC
```

Issues in a `--projects` project are shown as your team's own backlog; every other
project is treated as an external ask on the team.

### Auth without a browser

Paste a curl command copied from DevTools instead of doing an interactive login:

```sh
# In Chrome: DevTools → Network → any request → right-click → Copy as cURL
pbpaste > /tmp/jira.curl
go run ./cmd/go-jira-jwt-stats --curl-file /tmp/jira.curl --projects OPPEPROJ --team-members ... --open
```

## Features
* Makefile to build consistently in a local environment and remote environment
* Dockerfile for a generic image to build for
* Go Mod (which you should to your project path change)
* VS Code environment
* Generic docker push

## TODO
* Brew generic install [DONE]
* GITHUB Actions build and push to dockerhub [DONE]
* Production Builds with git tag

## Installing via brew
* `brew install --verbose --build-from-source brew/Formula/go-jira-jwt-stats.rb`
