# mp

Mixpanel CLI - query analytics from your terminal.

`mp` wraps the entire Mixpanel read API into a single command-line tool. Export raw events, run segmentation queries, inspect user profiles, and browse project metadata - all from your terminal with JSON, table, or CSV output.

## Installation

### Homebrew (macOS and Linux)

```bash
brew install aviadshiber/tap/mp
```

### Go Install

```bash
go install github.com/aviadshiber/mp/cmd/mp@latest
```

### Download Binary

Download the latest release from [GitHub Releases](https://github.com/aviadshiber/mp/releases/latest).

**macOS (Apple Silicon):**
```bash
curl -sL https://github.com/aviadshiber/mp/releases/latest/download/mp_darwin_arm64.tar.gz | tar xz
sudo mv mp /usr/local/bin/
```

**macOS (Intel):**
```bash
curl -sL https://github.com/aviadshiber/mp/releases/latest/download/mp_darwin_amd64.tar.gz | tar xz
sudo mv mp /usr/local/bin/
```

**Linux (amd64):**
```bash
curl -sL https://github.com/aviadshiber/mp/releases/latest/download/mp_linux_amd64.tar.gz | tar xz
sudo mv mp /usr/local/bin/
```

### From Source

```bash
git clone https://github.com/aviadshiber/mp
cd mp
make install
```

## Quick Start

### 1. Configure authentication

Create a [Service Account](https://developer.mixpanel.com/reference/service-accounts) in Mixpanel, then:

```bash
mp config set project_id YOUR_PROJECT_ID
mp config set service_account YOUR_SA_USERNAME
mp config set service_secret YOUR_SA_SECRET
```

Or use environment variables:
```bash
export MP_TOKEN="username:secret"
export MP_PROJECT_ID="12345"
```

### 2. Query your data

```bash
# Export raw events
mp export events --from 2024-01-01 --to 2024-01-31 --limit 100

# Segmentation query
mp query segmentation --event "Signup" --from 2024-01-01 --to 2024-01-31

# Aggregate event counts
mp query events --event "Signup,Login" --type general --unit day --from 2024-01-01 --to 2024-01-31

# User profiles
mp profiles query --where 'user["$city"]=="San Francisco"' --limit 10

# List cohorts
mp cohorts list
```

## Commands

### Export
| Command | Description |
|---------|-------------|
| `mp export events` | Export raw event data as JSONL |

### Query (Analytics)
| Command | Description |
|---------|-------------|
| `mp query segmentation` | Event segmentation (Insights report equivalent) |
| `mp query events` | Aggregate event counts over time |
| `mp query properties` | Event property breakdown |
| `mp query funnels` | Funnel conversion analysis |
| `mp query funnels list` | List saved funnels |
| `mp query retention` | User retention analysis |
| `mp query frequency` | Event frequency analysis |
| `mp query insights` | Query a saved Insights report |

### Profiles
| Command | Description |
|---------|-------------|
| `mp profiles query` | Query user profiles |
| `mp profiles groups` | Query group profiles |

### Metadata
| Command | Description |
|---------|-------------|
| `mp activity` | User activity stream |
| `mp cohorts list` | List cohorts |
| `mp annotations list` | List annotations |
| `mp annotations get` | Get annotation by ID |
| `mp schemas list` | List event/profile schemas |
| `mp schemas get` | Get schema details |
| `mp lookup-tables list` | List lookup tables |
| `mp pipelines list` | List data pipeline jobs |
| `mp pipelines status` | Get pipeline status |

## Output Formats

Every command supports the `--json`, `--jq`, and `--template` flags:

```bash
# JSON output
mp cohorts list --json

# Select specific fields
mp cohorts list --json id,name,count

# Filter with jq
mp cohorts list --json --jq '.[].name'

# Format with Go templates
mp cohorts list --json --template '{{range .}}{{.name}}: {{.count}}{{"\n"}}{{end}}'
```

Default output is a human-readable table in terminals, or JSON when piped.

## Configuration

Config file: `~/.config/mp/config.yaml`

```bash
mp config set <key> <value>
mp config get <key>
mp config list
```

| Key | Description | Env Variable |
|-----|-------------|-------------|
| `project_id` | Mixpanel project ID | `MP_PROJECT_ID` |
| `region` | API region (us, eu, in) | `MP_REGION` |
| `service_account` | Service account username | `MP_TOKEN` (user:secret) |
| `service_secret` | Service account secret | `MP_TOKEN` (user:secret) |

**Precedence**: flags > environment variables > config file > defaults

## EU and India Data Residency

```bash
mp config set region eu    # EU data residency
mp config set region in    # India data residency
```

Or per-command: `mp query segmentation --region eu --event "Signup" ...`

## License

MIT
