# MateCommit CLI Reference

**Technical Manual & Usage Guide**

This document serves as the authoritative reference for the MateCommit CLI. It details execution flows, configuration parameters, and internal behaviors to help you integrate the tool seamlessly into your development pipeline.

---

## 1. Commit Intelligence

### `suggest` / `s`
Analyzes staged changes (`git diff --cached`) and prompts the configured AI model to generate conventional commit messages.

**Command:**
```bash
matecommit suggest [flags]
```

**Technical Details:**
*   **Context Window**: The tool sends the diff summary, file names, and (optionally) linked issue context to the LLM. Large diffs are automatically truncated to fit within the model's token limit while preserving critical logic changes.
*   **Precedence**: Flags override `config.yaml` settings. For example, passing `--lang en` overrides a global `es` configuration.

**Flags:**
| Flag | Short | Type | Description |
| :--- | :--- | :--- | :--- |
| `--count` | `-n` | `int` | Quantity of suggestions to generate (1-10). |
| `--lang` | `-l` | `string` | Output language (ISO code, e.g., `en`, `es`, `pt`). |
| `--issue` | `-i` | `int` | Fetches issue title/description from the VCS provider to enrich AI context. |
| `--no-emoji` | `-ne` | `bool` | Strips emojis from the suggestion output for strict convention adherence. |

**Advanced Example:**
```bash
# Generate 5 suggestions, forcing English, using Issue #42 for context
matecommit suggest -n 5 -l en -i 42 --no-emoji
```

---

## 2. Pull Request Management

### `summarize-pr` / `spr`
Generates a structured Summary, Test Plan, and Breaking Changes warning for an existing Pull Request.

**Command:**
```bash
matecommit spr --pr-number <id>
```

**Workflow:**
1.  **Fetch**: Retries PR metadata (commits, diffs, linked issues) via GitHub API.
2.  **Analyze**: Uses Gemini to synthesize the changes into a cohesive narrative.
3.  **Update**: Patches the PR body directly on GitHub.

**Requirements:**
*   `GITHUB_TOKEN` must be set in your environment or config.
*   Token scopes: `repo` (for private repos) or `public_repo` (for public).

---

## 3. Issue Lifecycle

### `issue generate` / `g`
Creates a GitHub Issue using AI to format vague inputs into professional reports.

**Command:**
```bash
matecommit issue generate [source-flags] [options]
```

**Source Flags (Mutually Exclusive):**
*   `--from-diff` / `-d`: Uses current staged changes as the basis for the issue (ideal for "I fixed this, now I need a ticket").
*   `--from-pr` / `-p`: Uses a PR's title/body to create a tracking issue.
*   `--description` / `-m`: Uses a raw text string as input.

**Options:**
*   `--template` / `-t`: Target specific template keys (e.g., `bug_report`). Matches filename in `.github/ISSUE_TEMPLATE/`.
*   `--checkout` / `-c`: Automates branch creation (`git checkout -b issue/123-title`) after generation.
*   `--dry-run`: Prints the generated Markdown to stdout without calling the GitHub API.

**Scenario:**
*You just hacked together a fix but didn't open a ticket.*
```bash
git add .
matecommit issue generate --from-diff --template bug_report --assign-me --checkout
```
*Result: Creates issue, assigns you, and switches branch.*

---

## 4. Release Automation

### `release` / `r`
Standardizes the release process following [Semantic Versioning](https://semver.org/).

**Subcommands:**

#### `preview` / `p`
Dry-run of the release. Calculates the next version (e.g., `v1.0.0` -> `v1.1.0`) based on commit history (Conventional Commits analysis) and generates the draft changelog.

#### `create` / `c`
Executes the release locally.
1.  Updates `CHANGELOG.md` (prepends new entry).
2.  Creates a git tag.
3.  (Optional) Pushes changes.

**Flags:**
*   `--auto`: Non-interactive mode (for CI/CD scripts).
*   `--changelog`: Forces the commit of the updated changelog file.
*   `--publish`: Triggers `git push origin <tag>` immediately.

#### `publish` / `pub`
Synchronizes a local tag with GitHub Releases. Creates the release entry on GitHub with the AI-generated notes.

---

## 5. System & Config

### `config`
**File Location**: `~/.config/matecommit/config.yaml` (Linux) or standard OS config paths.

*   `init`: Interactive wizard.
*   `doctor`: Connectivity check (Gemini API, GitHub API, Git binary path).

### `stats`
Displays cost estimation based on token usage.
*   **Note**: Costs are estimates based on standard Gemini pricing. Actual billing may vary.

### `update`
Self-updater using GitHub Releases. Replaces the current binary with the latest stable version.

---

## Environment Variables

MateCommit respects the following environment variables, which override config file values:

*   `GEMINI_API_KEY`: Google AI Studio Key.
*   `GITHUB_TOKEN`: GitHub Personal Access Token.
*   `MATECOMMIT_LANG`: Default language override.