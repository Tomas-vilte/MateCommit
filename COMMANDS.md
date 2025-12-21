# MateCommit CLI Reference 🧉

I wrote this guide to explain not just *what* each command does, but how they actually work behind the scenes. The design is modular, meaning I can keep adding new AI models and platforms without breaking your existing workflow.

---

## 1. The Suggestion Engine

### `suggest` / `s`
This is the command I use the most. It analyzes what you have in stage and asks the AI to give you commit message options that actually make sense.

**Usage:**
```bash
matecommit suggest [flags]
```

**How the magic works:**
1.  **Diff Analysis**: I run `git diff --cached` to see exactly what you changed.
2.  **Context Construction**: I build a prompt for your provider (like Gemini) using the diff summary and file names.
3.  **Smart Truncation**: If your diff is humongous, I don't just throw an error at you. I use an algorithm that prioritizes the most critical logic changes to stay within the model's token limits while maintaining quality.
4.  **Context Boost**: If you use the `--issue` flag, I'll fetch the issue title and description so the AI understands the "why" behind your code.

**Available Flags:**

`--count` / `-n` (int)
> How many suggestions you want to see at once. (Default: 3, Max: 10)

`--lang` / `-l` (string)
> Override the language for just this commit (e.g., if you're working on an English repo but your global config is set to Spanish).

`--issue` / `-i` (int)
> Pulls in the full context of a specific issue to make the suggestions much smarter.

`--no-emoji` / `-ne` (bool)
> Strips all emojis for when you need a strictly technical and sober commit history.

**Pro Tip**: Run `matecommit suggest -n 5 -l en` to get 5 English suggestions instantly, regardless of your default settings.

---

## 2. PR & Issue Management

### `summarize-pr` / `spr`
I use this when I'm finishing up a PR and can't be bothered to write the whole summary, test plan, and check for breaking changes manually.

**The workflow is simple:**
1.  **Metadata**: It pulls commits and comments directly from your VCS API (GitHub, for now).
2.  **Synthesis**: The LLM reads the entire history of the PR and builds a cohesive summary.
3.  **Direct Patching**: It updates the PR description on the platform for you.

### `issue generate` / `g`
I hate having to leave the terminal and open a browser just to create a ticket. This command turns your rough CLI input into a professional issue.

**Where it gets the info:**
- **From Diff**: Uses your current staged changes as the basis for describing the task or bug.
- **Auto-Checkout**: If you use `--checkout`, I'll automatically create a new branch named after the issue so you can start working immediately.

---

## 3. Release Automation

### `release` / `r`
I built this to take the stress out of managing Semantic Versioning (SemVer) manually.

1.  **Analysis**: I review your commit history (based on Conventional Commits) and suggest if the next step is Patch, Minor, or Major.
2.  **Changelog**: I update your `CHANGELOG.md` automatically with the new entries.
3.  **Tagging**: I create the git tag locally.
4.  **Publishing**: I sync everything with your VCS and create a full Release with AI-generated notes.

---

## 4. Configuration & System

### `config`
All your settings live in `~/.config/matecommit/config.yaml`.
*   **Precedence**: Command flags > Environment variables > Config file.
*   **Doctor**: If something feels off, run `matecommit config doctor`. It checks connectivity, token permissions, and API responses.

### `stats`
Since AI APIs aren't always free (or have limits), I added token tracking. You can see your usage estimates so you don't get a surprise at the end of the month.

---

## Common Troubleshooting

**"The suggestions aren't very good"**
*   *Tip*: Make sure you only stage related changes. If you throw 5 different features into one stage, the AI will get confused by the context.

**"API Error"**
*   *Tip*: Run the `doctor` command. Your `GEMINI_API_KEY` or `GITHUB_TOKEN` likely expired or lacks the necessary scopes.

---

## Current Support

*   **AI Models**: Google Gemini (Default).
*   **VCS**: GitHub.
*   **Issues**: Jira and GitHub Issues.