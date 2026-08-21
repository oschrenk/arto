# arto

arto is a Claude companion

## Notify

Claude Code asks for permission before running tools like `Bash`, `Write`, and `Edit`. These prompts require you to be sitting at your terminal. If you step away, Claude Code stalls until you come back and press "y".

**arto** sends each permission prompt as a push notification to your phone via [ntfy.sh](https://ntfy.sh). You tap **Approve** or **Deny**, and Claude Code continues immediately — no terminal required.

See [docs/notify.md] for more details.

## Installation

### From source

```bash
go install github.com/oschrenk/arto/cmd/arto@latest
```

### With Homebrew

```bash
brew install oschrenk/made/arto
```

