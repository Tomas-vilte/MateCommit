<div style="text-align:center">
  <img src="./assets/logo.jpeg" alt="MateCommit Logo" width="1376">

# MateCommit

**I built this because I honestly couldn't be bothered to think of a name for every single commit.**

[Quick Start](#quick-start) • [Features](#what-it-actually-does) • [Documentation](./COMMANDS.md) • [Contributing](./CONTRIBUTING.md)

[![Go Report Card](https://goreportcard.com/badge/github.com/thomas-vilte/matecommit)](https://goreportcard.com/report/github.com/thomas-vilte/matecommit)
[![License](https://img.shields.io/github/license/thomas-vilte/matecommit)](https://opensource.org/licenses/MIT)
[![Build Status](https://github.com/thomas-vilte/matecommit/actions/workflows/ci.yml/badge.svg)](https://github.com/thomas-vilte/matecommit/actions)
</div>

---

## Why MateCommit?

You know that feeling when you've been coding for hours, your brain is fried, and you're staring at the terminal unable to describe what you just did? I do.

Writing good commit messages and PR descriptions is important, but it's often the last thing you want to do when you're tired. MateCommit was born out of that frustration. It reads your changes and suggests clear, meaningful messages using AI (Google Gemini), so you can focus on the code and leave the creative writing to the models.

It handles the boring parts of the workflow:
- Writing conventional commit messages.
- Summarizing PRs with test plans and breaking changes.
- Creating JIRA tickets from your changes.
- Managing releases, SemVer, and changelogs.

---

## Demo

<div style="text-align:center">
  <img src="./assets/demo_commits.gif" alt="MateCommit Demo" width="1843">
</div>

<details>
<summary>See example output</summary>

```bash
$ matecommit suggest

Analyzing changes...
Found 3 files changed, 127 insertions, 45 deletions

Suggestions:
1. feat(auth): implement JWT-based authentication with refresh tokens
2. feat: add user authentication system with JWT support
3. feat(api): integrate JWT authentication middleware for secure endpoints

Select a suggestion (1-3): 1
Committed: feat(auth): implement JWT-based authentication with refresh tokens
```

</details>

---

## Quick Start

### 1. Install
You'll need Go installed on your machine:
```bash
go install github.com/thomas-vilte/matecommit/cmd/matecommit@latest
```

### 2. Configure
Set up your Gemini API key (it takes 10 seconds):
```bash
matecommit config quick
```

### 3. Use it
Stage your changes and let the tool do the work:
```bash
matecommit suggest
```

---

## What it actually does

While there are other tools out there, I built MateCommit to be a complete workflow tool, not just a message generator.

| Feature                | MateCommit                               | Most Other Tools |
|------------------------|------------------------------------------|------------------|
| **Commit Messages**    | AI-powered, follows Conventional Commits | Yes              |
| **PR Summaries**       | Includes test plans and breaking changes | No               |
| **Issue Generation**   | Create from diff, PR, or description     | No               |
| **Release Automation** | Handles SemVer, Changelogs and Tags      | No               |
| **Jira Integration**   | Auto-updates and links tickets           | No               |
| **Cost Awareness**     | Built-in token and cost tracking         | No               |

### Main Features

*   **Smart Commits**: It analyzes the full diff context, not just filenames. It can also take context from a specific issue number to be more precise.
*   **PR Automation**: Use `matecommit spr <id>` to generate a full executive summary, test plan, and detect breaking changes automatically.
*   **Issue Management**: Generate issues directly from your code changes or descriptions. It even supports Jira integration and can auto-checkout branches for you.
*   **Releases**: An interactive wizard that analyzes your commits since the last tag, suggests the next version bump, and writes the changelog for you.
*   **Developer Experience**: Includes shell autocompletion (bash, zsh, fish) and a `doctor` command to make sure your integrations are working correctly.

---

## How It Works

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
│  AI Provider    │ ──▶ Google Gemini
└─────────┬───────┘
          │
          ▼
┌─────────────────┐
│  MateCommit     │ ──▶ Generates suggestions
└─────────┬───────┘
          │
          ▼
┌─────────────────┐
│  Your Choice    │ ──▶ Select and commit
└─────────────────┘
```

**Tech Stack:**
- **Language:** Go (fast, single binary).
- **AI:** Google Gemini (OpenAI and Claude support is coming).
- **Platforms:** GitHub (VCS) and Jira (Tickets).

---

## Roadmap

Stuff I'm planning to add soon:
- [ ] **Local LLMs**: Support for Ollama so you can use it for free/offline.
- [ ] **More Providers**: OpenAI and Claude integration.
- [ ] **Code Review**: AI-powered feedback before you even commit.
- [ ] **GitLab/Bitbucket**: Support for other platforms.

---

## Contributing

MateCommit is open source, and I'd love to have more people involved. Whether it's adding a new AI provider, fixing a bug, or just improving the templates, all help is welcome.

Check out [CONTRIBUTING.md](./CONTRIBUTING.md) to see how to get started.

---

## License

MIT License - see [LICENSE](./LICENSE) for details.

---

<div style="text-align:center">

**Made with 🧉 by developers, for developers**

[⬆ Back to top](#matecommit)

</div>