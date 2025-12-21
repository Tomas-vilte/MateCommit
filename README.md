<div style="text-align:center">
  <img src="./assets/logo.jpeg" alt="MateCommit Logo" width="1376">

  # MateCommit

  **I built this because I honestly couldn't be bothered to think of a name for every single commit.**

  <img src="./assets/leny-pensando.jpg" alt="Lenny Thinking Meme" width="439">

  You know that feeling when you're staring at the terminal after hours of coding, and your brain just goes blank? Yeah, me too. MateCommit was born out of that exact frustration. It's an AI-powered CLI that reads your changes and suggests clear, meaningful commit messages so you can focus on the actual work and leave the creative writing to the LLMs.

  [![Go Report Card](https://goreportcard.com/badge/github.com/Tomas-vilte/MateCommit)](https://goreportcard.com/report/github.com/Tomas-vilte/MateCommit)
  [![License](https://img.shields.io/github/license/Tomas-vilte/MateCommit)](https://opensource.org/licenses/MIT)
  [![Build Status](https://github.com/Tomas-vilte/MateCommit/actions/workflows/ci.yml/badge.svg)](https://github.com/Tomas-vilte/MateCommit/actions)

</div>

---

### Languages
*   [Traducción al Español (🇦🇷)](./docs/es/README.md)

---

## Why MateCommit exists 🧉

Let's be real: writing good commit messages and PR descriptions is crucial, but when you're in the zone or just finished a massive task, the last thing you want to do is spend mental energy explaining a `diff`.

I built MateCommit to automate the boring parts of my Git workflow, but doing it the right way:

- **No more "fix", "update", or "changes"**: It uses advanced LLMs (like Google Gemini) to understand the *actual* context of your code.
- **Effortless conventions**: It follows *Conventional Commits* automatically, so your history stays pristine without you having to remember every prefix.
- **Modular by design**: It's built to integrate with different AI models and platforms, starting with GitHub and Jira.
- **Cost awareness**: I added a token tracker so you know exactly how much each request is costing you.

## What does it do for you?

- **Instant Suggestions**: Run a command and get message options based on what you actually changed.
- **Automatic PRs**: Generate structured Pull Request summaries with test plans and breaking change warnings.
- **Stress-free Releases**: It handles versioning, generates changelogs, and creates Git tags for you.
- **Great DX**: Built for dev productivity with shell autocompletion and diagnostic tools.

---

## Start now

### 1. Installation
If you have Go installed:

```bash
go install github.com/thomas-vilte/matecommit/cmd/matecommit@latest
```

### 2. Configuration
Set up your credentials and preferred providers:

```bash
matecommit config init
```

### 3. Usage
Stage your changes and let the AI do its magic:

```bash
git add .
matecommit suggest
```

#### Common Flags
- `-n` : How many suggestions you want to see (if you're feeling picky).
- `-l` : Force a specific language (e.g., if the repo is English but your config is Spanish).
- `-i` : Pass an issue number for even more precise suggestions.
- `--no-emoji` : For when the environment is serious, and you don't want little icons.

---

## Looking forward

MateCommit is designed to grow with the community:

*   **Modular AI**: Easily switch between different models as we add support.
*   **Custom Templates**: Tailor the output to exactly how your team likes it.

For a full technical deep dive into all commands, check out [COMMANDS.md](./COMMANDS.md).

---

## Contributing

I'd love your help! Whether it's adding a new AI model or a new VCS platform, check the [Contributing Guidelines](./CONTRIBUTING.md) to get started.

---

## License

Distributed under the MIT License. See [LICENSE](./LICENSE) for more information.