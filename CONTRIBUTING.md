# Contributing to MateCommit

Thank you for your interest in contributing to MateCommit! We welcome contributions of all kinds, from bug fixes and new features to documentation improvements.

While the project is currently maintained by a small team, we strive to follow best practices to ensure code quality and project sustainability.

---

## Getting Started

1.  **Fork the repository** on GitHub.
2.  **Clone your fork** locally:
    ```bash
    git clone https://github.com/YOUR_USERNAME/MateCommit.git
    ```
3.  **Sync dependencies**:
    ```bash
    go mod tidy
    ```
4.  **Create a feature branch**:
    ```bash
    git checkout -b feature/your-awesome-feature
    ```

---

## Contribution Guidelines

### 1. Code Quality & Standards
- Ensure your code follows standard Go formatting (`go fmt`).
- Run tests before submitting a Pull Request:
  ```bash
  go test ./...
  ```
- If you're adding a new feature, please include corresponding tests.

### 2. Commit Messages
We practice what we preach! Please use clear and descriptive commit messages. We recommend the [Conventional Commits](https://www.conventionalcommits.org/) format:
- `feat: add support for new AI model`
- `fix: resolve issue with Jira integration`
- `docs: update installation instructions`

### 3. Pull Request Process
- Provide a clear description of the changes and the problem they solve.
- Link any related issues in the PR description.
- Ensure all CI checks pass.
- A maintainer will review your PR as soon as possible.

---

## Best Practices
- **Update Documentation**: If your change affects how the tool is used, please update `README.md` or `COMMANDS.md`.
- **Stay Updated**: Before starting work, ensure your fork is synced with the main repository's `master` branch.

---

## Admin & Releases
*Note: Tagging and release management are currently handled by the core maintainers to ensure stable versions are delivered.*

Thank you for helping us make MateCommit even better! Happy coding.
