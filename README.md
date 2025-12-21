<div align="center">
  <img src="/home/enano/.gemini/antigravity/brain/8d348096-a7bc-4507-b3d1-e0ceeee5f35f/matecommit_social_preview_1766175423747.png" alt="MateCommit Logo" width="640">

  # MateCommit

  **Effortless Git Intelligence**

  MateCommit leverage the power of Gemini AI to transform your raw code changes into meaningful, descriptive commit history. Stop struggling with naming; focus on building.

  [Go Report Card](https://goreportcard.com/report/github.com/Tomas-vilte/MateCommit) | [License](https://opensource.org/licenses/MIT) | [Build Status](https://github.com/Tomas-vilte/MateCommit/actions)

</div>

---

### Languages
*   [Official Documentation (English)](#)
*   [Traducción al Español (🇦🇷)](./docs/es/README.md)

---

## The Value Proposition
Writing high-quality commit messages is a critical yet time-consuming task. MateCommit eliminates this friction by analyzing your `git diff` and suggesting context-aware messages that follow conventional standards.

*   **Intelligent Suggestions**: Contextual analysis of logic changes, not just file names.
*   **Gemini Power**: Optimized for Flash 1.5 and 2.0 models for speed and accuracy.
*   **Issue Life-cycle**: Generate GitHub issues from code, PRs, or descriptions.
*   **Unified Releases**: Automated changelog generation, tagging, and GitHub publishing.
*   **PR Intelligence**: Instant summaries for complex Pull Requests.
*   **Usage Insights**: Track AI costs and usage statistics directly in your terminal.
*   **Efficiency Tools**: Shell autocompletion, local caching, and diagnostic tools.


---

## Quick Start

### 1. Installation
Install the latest binary using Go:

```bash
go install github.com/Tomas-vilte/MateCommit/cmd/matecommit@latest
```

### 2. Initial Setup
Run the guided configuration to set up your API keys and preferences:

```bash
matecommit config init
```

### 3. Usage
Stage your files and generate suggestions:

```bash
git add .
matecommit suggest
```

---

## Advanced Features
MateCommit is built for professional workflows, including:

*   **Jira Integration**: Automatic ticket linking based on branch context.
*   **PR Summarization**: Generates standardized Pull Request bodies.
*   **Release Automation**: Updates changelogs and pushes tags in one motion.

For a full technical overview of all commands, see [COMMANDS.md](./COMMANDS.md).

---

## Contributing
We value high-quality contributions. If you are interested in improving the codebase or documentation, please review our [Contribution Guidelines](./CONTRIBUTING.md).

---

## License
Distributed under the MIT License. See [LICENSE](./LICENSE) for more information.