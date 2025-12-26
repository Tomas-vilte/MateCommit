<div align="center">
  <img src="./assets/logo.jpeg" alt="MateCommit Logo" width="600">

  # MateCommit 🧉

  **AI-powered Git workflow automation: from commits to releases**

  Stop wasting time on commit messages, PR descriptions, and release notes.
  Let AI handle the boring parts while you focus on code.

  [![Go Report Card](https://goreportcard.com/badge/github.com/thomas-vilte/matecommit)](https://goreportcard.com/report/github.com/thomas-vilte/matecommit)
  [![License](https://img.shields.io/github/license/thomas-vilte/matecommit)](https://opensource.org/licenses/MIT)
  [![Build Status](https://github.com/thomas-vilte/matecommit/actions/workflows/ci.yml/badge.svg)](https://github.com/thomas-vilte/matecommit/actions)

  [Quick Start](#-quick-start-60-seconds) • [Features](#-what-makes-it-different) • [Documentation](./COMMANDS.md) • [Contributing](./CONTRIBUTING.md)

</div>

---

## 🎯 The Problem

You've spent 4 hours coding. Your brain is fried. Now you need to:
- ✍️ Write meaningful commit messages
- 📝 Summarize your PR with test plans and breaking changes
- 🎫 Create JIRA tickets from your changes
- 🚀 Manage releases with SemVer and changelogs

**MateCommit does all of this in seconds.**

---

## 🎬 Demo

<div align="center">
  <img src="./assets/demo.gif" alt="MateCommit Demo" width="800">
</div>

<details>
<summary>📝 See example output</summary>

```bash
$ git add .
$ matecommit suggest

🧉 Analyzing changes...
✓ Found 3 files changed, 127 insertions, 45 deletions

Suggestions:
1. feat(auth): implement JWT-based authentication with refresh tokens
2. feat: add user authentication system with JWT support
3. feat(api): integrate JWT authentication middleware for secure endpoints

Select a suggestion (1-3): 1
✓ Committed: feat(auth): implement JWT-based authentication with refresh tokens
```

</details>

---

## ⚡ Quick Start (60 seconds)

### 1. Install
```bash
go install github.com/thomas-vilte/matecommit/cmd/matecommit@latest
```

### 2. Configure (one-time setup)
```bash
matecommit config quick
# Enter your Gemini API key and you're done
```

### 3. Use it
```bash
git add .
matecommit suggest
```

Done. ✅

---

## 🚀 What Makes It Different

MateCommit isn't just another commit message generator. It's a **complete Git workflow automation platform**.

| Feature | MateCommit | Other Tools* |
|---------|------------|--------------|
| **Commit Messages** | ✅ AI-powered, Conventional Commits | ✅ |
| **PR Summaries** | ✅ With test plans + breaking changes | ❌ |
| **Issue Generation** | ✅ From diff, PR, or description | ❌ |
| **Release Automation** | ✅ SemVer + Changelog + Tags | ❌ |
| **Jira Integration** | ✅ Ticket linking + auto-updates | ❌ |
| **Multi-language** | ✅ English + Spanish | ⚠️ Limited |
| **Token Tracking** | ✅ Cost awareness built-in | ❌ |
| **Templates** | ✅ Customizable issue templates | ❌ |

<sub>*Compared to aicommits, OpenCommit, aicommit2</sub>

---

## 💎 Core Features

### 🧠 Intelligent Commit Messages
```bash
matecommit suggest -n 5          # Get 5 suggestions
matecommit suggest -i 123        # Include context from issue #123
matecommit suggest -l es         # Generate in Spanish
```

**Smart features:**
- Analyzes full diff context, not just file names
- Follows Conventional Commits automatically
- Learns from issue context when provided
- Handles large diffs with intelligent truncation

---

### 📋 PR Automation
```bash
matecommit spr 456               # Summarize PR #456
```

Generates:
- Executive summary of changes
- Detailed test plan
- Breaking change detection
- Auto-updates PR description on GitHub

---

### 🎫 Issue Management
```bash
matecommit issue generate -d                    # Generate from diff
matecommit issue generate -m "Add dark mode"    # From description
matecommit issue generate --from-pr 123         # From existing PR
matecommit issue generate -d -c                 # Generate + auto-checkout branch
```

**Includes:**
- Auto-generated title and description
- Smart label suggestions
- Jira integration support
- Automatic branch creation and checkout

---

### 🚀 Release Automation
```bash
matecommit release                              # Interactive release wizard
```

**Handles everything:**
- Analyzes commits since last release
- Suggests version bump (patch/minor/major)
- Generates changelog from conventional commits
- Creates Git tags
- Publishes GitHub releases with AI-generated notes

---

### 🔧 Developer Experience
```bash
matecommit config doctor        # Health check for all integrations
matecommit config show          # View current configuration
matecommit stats                # Track token usage and costs
```

**Built for productivity:**
- Shell autocompletion (bash, zsh, fish)
- Comprehensive error messages
- Diagnostic tools for debugging
- Token usage tracking to monitor AI costs

---

## 🎨 Use Cases

### For Solo Developers
- Never think about commit messages again
- Professional PR descriptions without effort
- Automated release notes

### For Teams
- Consistent commit history across contributors
- Standardized PR format
- JIRA ticket integration
- Release coordination

### For Open Source
- High-quality commit messages attract contributors
- Professional PR summaries
- Clear release notes for users

---

## 📚 Documentation

- [**Commands Reference**](./COMMANDS.md) - Deep dive into all commands
- [**Contributing Guide**](./CONTRIBUTING.md) - Help improve MateCommit
- [**Español**](./docs/es/README.md) - Documentación en español

---

## 🏗️ How It Works

```
┌─────────────┐
│  Your Code  │
└──────┬──────┘
       │
       ▼
┌─────────────────┐
│  Git Diff       │ ──▶ Analyzes changes
└─────────┬───────┘
          │
          ▼
┌─────────────────┐
│  AI Provider    │ ──▶ Gemini (OpenAI/Claude coming soon)
│  (Gemini)       │
└─────────┬───────┘
          │
          ▼
┌─────────────────┐
│  MateCommit     │ ──▶ Generates suggestions
│  Engine         │
└─────────┬───────┘
          │
          ▼
┌─────────────────┐
│  Your Choice    │ ──▶ Select and commit
└─────────────────┘
```

**Tech Stack:**
- **Language:** Go (fast, single binary, cross-platform)
- **AI:** Google Gemini (OpenAI, Claude, Ollama coming soon)
- **VCS:** GitHub (GitLab, Bitbucket planned)
- **Tickets:** Jira, GitHub Issues

---

## 🛣️ Roadmap

### Coming Soon
- [ ] **Ollama Support** - Use local models for free, private commits
- [ ] **OpenAI & Claude** - More AI provider options
- [ ] **Code Review** - AI-powered review before commit
- [ ] **Test Generation** - Auto-generate unit tests from changes
- [ ] **GitLab/Bitbucket** - Support more VCS platforms

### Under Consideration
- [ ] Watch mode - Smart auto-commit on logical checkpoints
- [ ] Team templates - Share configurations across teams
- [ ] Slack/Discord notifications
- [ ] Custom AI prompts

**Have ideas?** [Open an issue](https://github.com/thomas-vilte/matecommit/issues/new) or join the discussion!

---

## 🤝 Contributing

MateCommit is open source and welcomes contributions!

**Good first issues:**
- Add support for new AI providers (OpenAI, Claude, Ollama)
- Improve commit message templates
- Add translations (French, German, Portuguese)
- Write tests for uncovered code

See [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

---

## 🙏 Acknowledgments

Inspired by the frustration of writing commit messages at 2 AM.

Built with:
- [Google Gemini](https://ai.google.dev/) - AI provider
- [urfave/cli](https://github.com/urfave/cli) - CLI framework
- [charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea) - TUI components

---

## 📄 License

MIT License - see [LICENSE](./LICENSE) for details.

---

## ⭐ Support

If MateCommit saves you time, consider:
- Starring the repo ⭐
- Sharing with other developers
- [Contributing](./CONTRIBUTING.md) new features
- [Sponsoring development](https://github.com/sponsors/thomas-vilte) (if available)

---

<div align="center">

**Made with 🧉 by developers, for developers**

[⬆ Back to top](#matecommit-)

</div>