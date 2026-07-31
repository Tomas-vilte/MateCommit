# MateCommit CLI Reference 🧉

I wrote this guide to explain not just *what* each command does, but how they actually work behind the scenes. The design is modular, meaning I can keep adding new AI models and platforms without breaking your existing workflow.

Every command also has its own `--help`, so if this guide ever falls behind the code, that's the source of truth.

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
2.  **Context Construction**: I build a prompt for your provider (Gemini, for now) using the diff summary and file names.
3.  **Smart Truncation**: If your diff is humongous, I don't just throw an error at you. I use an algorithm that prioritizes the most critical logic changes to stay within the model's token limits while maintaining quality.
4.  **Context Boost**: If you use the `--issue` flag, I'll fetch the issue title and description so the AI understands the "why" behind your code.

**Available Flags:**

`--count` / `-n` (int)
> How many suggestions you want to see at once. (Default: 3, Max: 10)

`--lang` / `-l` (string)
> Override the language for just this commit (e.g., if you're working on an English repo but your global config is set to Spanish).

`--issue` (int)
> Pulls in the full context of a specific issue to make the suggestions much smarter.

`--no-emoji` / `--ne`
> Strips all emojis for when you need a strictly technical and sober commit history.

`--interactive` / `-i`
> Instead of committing everything you have staged, this lets you pick exactly which changed files go into the AI summary. Useful when you staged more than one logical change and don't want to split it into separate `git add` calls.

`--dry-run` / `-d`
> Shows you the file list, the diff stats, and an estimated token cost — without calling the AI or touching your repo. Good for a sanity check before you burn a request.

**Pro Tip**: Run `matecommit suggest -n 5 -l en` to get 5 English suggestions instantly, regardless of your default settings.

---

## 2. PR & Issue Management

### `summarize-pr` / `spr`
I use this when I'm finishing up a PR and can't be bothered to write the whole summary, test plan, and check for breaking changes manually.

**Usage:**
```bash
matecommit summarize-pr --pr-number <id>
```

**The workflow is simple:**
1.  **Metadata**: It pulls commits, comments, and the diff directly from your VCS API (GitHub, for now).
2.  **Synthesis**: The LLM reads the entire history of the PR and builds a cohesive summary, test plan, and breaking-change callout.
3.  **Direct Patching**: It updates the PR title, body, and labels on the platform for you.

**Available Flags:**

`--pr-number` / `-n` (int)
> The number of the PR you want summarized. Required.

`--hint` / `-H` (string)
> Anything extra you want the AI to keep in mind — context that isn't obvious from the diff alone.

### `issue` / `i`
Everything related to creating and managing issues lives under this one. I hate having to leave the terminal and open a browser just to file a ticket.

#### `issue generate` / `g`
Turns your rough CLI input into a properly written issue, with labels inferred automatically from your repo's actual label set.

**Where it gets the info (pick one source):**

`--from-diff` / `-d`
> Uses your current staged changes as the basis for describing the task or bug.

`--from-pr` / `-p` / `--pr` (int)
> Generates the issue from an existing Pull Request instead — useful for opening a tracking issue after the fact.

`--description` / `-m` (string)
> Just tell it what you want in plain language and let the AI flesh it out.

**Other flags:**

`--hint` / `-h` (string)
> Extra guidance for the AI, on top of whichever source you picked above.

`--template` / `-t` (string)
> Force a specific issue template instead of letting the AI infer the type of issue.

`--auto-template`
> Let the AI pick the best-fitting template on its own, if you haven't specified one.

`--no-labels`
> Skip label inference entirely.

`--assign-me` / `-a`
> Assign the created issue to yourself.

`--checkout` / `-c`
> Automatically create and check out a new branch named after the issue, so you can start working immediately.

`--dry-run`
> Preview the generated issue without actually creating it.

#### `issue link` / `l`
Links an existing PR to an existing issue (adds the "Closes #X" reference GitHub understands).

```bash
matecommit issue link --pr <pr-number> --issue <issue-number>
```

#### `issue template` / `t`
Manages the issue templates matecommit uses to structure generated issues.

- `issue template init` — Drops the default set of templates (bug report, feature request, tech debt, security, etc.) into `.github/ISSUE_TEMPLATE/`. Pass `--force` to overwrite ones you already have.
- `issue template list` / `ls` / `l` — Shows which templates are currently available in the repo.

#### `issue from-plan`
If you (or an AI assistant) already wrote out an implementation plan as a markdown file, this breaks it down into individual issues instead of making you copy-paste each section by hand.

```bash
matecommit issue from-plan --file PLAN.md
```

`--file` / `-f` (string)
> Path to the plan file. Required.

`--labels` / `-l` (string, repeatable)
> Extra labels to add to every issue created from the plan.

`--assign-me` / `-a`
> Assign yourself to the created issues.

`--dry-run` / `-d`
> Preview what would be created without actually opening anything.

---

## 3. Release Automation

### `release` / `r`
I built this to take the stress out of managing Semantic Versioning (SemVer) manually. It's actually six commands, because "create a release" means different things depending on where you are in the process.

Most of them (`preview`, `generate`, `create`, `publish`) expect you to be on `main` or `master` first — releases shouldn't come from a random feature branch. `git checkout main` before you run one if it complains.

- **`release preview` / `p`** — Shows what the next release would look like (version bump, changelog entries) without creating anything. Good for a sanity check before committing to a version number.
- **`release generate` / `g`** — Generates the release notes and writes them to a file (`RELEASE_NOTES.md` by default, override with `--output` / `-o`) instead of publishing anything.
- **`release create` / `c`** — The full pipeline: analyzes commits since the last tag, updates `CHANGELOG.md`, bumps the version file, creates the git tag, and (optionally) publishes.
  - `--auto` / `-y` — Skip the confirmation prompts.
  - `--version` / `-v` — Override the auto-detected version (e.g. `v1.2.3`).
  - `--publish` — Also publish the release to GitHub once it's created.
  - `--draft` — Publish as a draft (only makes sense together with `--publish`).
  - `--changelog` — Update `CHANGELOG.md` and commit that change automatically.
  - `--build-binaries` / `-b` — Cross-compile and upload binaries as release assets.
  - `--main-path` — Where your `main` package lives, if binary building needs it.
- **`release push`** — Pushes an existing tag to the remote. Auto-detects the version if you don't pass `--version` / `-v`. If your remote has a ruleset blocking direct pushes, this is the command most likely to hit it (see the note below).
- **`release publish` / `pub`** — Publishes an already-tagged release to GitHub. Same `--version`, `--draft` (`-d` here), `--build-binaries` (`-b`), and `--main-path` flags as `create`.
- **`release edit` / `e`** — Opens an existing release's notes in your editor for manual tweaks. Pass `--ai` / `-a` to have the AI regenerate/improve them first, and `--editor` / `-e` to override which editor it opens (defaults to `$EDITOR`, then falls back to nano/vim).

**About GitHub Rulesets**: if your main/master branch (or your tag names) are protected by a GitHub ruleset, a direct push will get rejected — and I'll tell you exactly why, with GitHub's own error message included. When the push is for the changelog commit specifically, I'll try pushing it as a branch and opening a PR for you automatically instead; you just merge it and re-run the command to finish the release.

---

## 4. Configuration & System

### `config` / `c`
Your settings live in `.matecommit/config.json` if you're inside a git repo (local config), or `~/.config/matecommit/config.json` otherwise (global). Local always wins over global when both exist — matecommit doesn't merge them field by field.

- **`config show`** — Prints the resolved configuration (local or global, whichever applies), with the API key masked.
- **`config init`** — The setup wizard. Run it with no flags and it'll ask you quick vs. full; or skip straight to one with `--quick` / `-q` or `--full`. Add `--local` / `-l` to scope it to the current repo instead of your global config, or `--global` / `-g` to force global even inside a repo.
- **`config set <key> <value>`** — Set a single value without going through the wizard, e.g. `matecommit config set lang es`. Supported keys: `lang`/`language`, `emoji`/`use_emoji`, `count`/`suggestions_count`, `active-ai`, `model`, `active-vcs`, `git.name`, `git.email`. Add `--local` / `-l` or `--global` / `-g` to be explicit about which file gets written.
- **`config edit`** — Opens the config file directly in your editor, for when the wizard is more trouble than it's worth.

### `doctor` / `dr`
Runs a full health check: internet connectivity, that git is installed and you're inside a repo, your git identity (name/email), whether your active AI provider is configured, your GitHub token and its scopes, and whether it can find an editor to use. If something feels off, this is always the first thing to run — it tells you exactly what's missing instead of making you guess from a stack trace.

```bash
matecommit doctor
```

### `stats` / `cost`
AI APIs aren't free, so I added usage tracking. Every call gets logged locally with its token count and cost.

- `matecommit stats` — Today's usage.
- `--monthly` / `-m` — This month's usage instead, broken down by day.
- `--breakdown` / `-b` — Usage grouped by command (how much of your spend is `suggest` vs `summarize-pr` vs everything else).
- `--forecast` / `-f` — A projection of what you'll spend by the end of the month at your current pace.

### `cache`
Responses from the AI are cached locally so re-running the same request (or retrying after a failure) doesn't burn another API call. `matecommit cache clean` wipes that cache if you want a clean slate — useful if you suspect a stale cached response is the reason a suggestion looks off.

### `completion`
Generates a shell completion script.

```bash
matecommit completion bash   # print the bash script
matecommit completion zsh    # print the zsh script
matecommit completion install  # detect your shell and wire it up automatically
```

`completion install` looks at your `$SHELL`, appends the right `source` line to your `.bashrc` or `.zshrc`, and tells you to restart your shell (or just `source` the file yourself). Only bash and zsh are supported right now.

### `update`
Updates matecommit to the latest release. It figures out how you installed it (`go install`, Homebrew, or a raw binary download) and updates it the same way, so it doesn't fight with your package manager.

```bash
matecommit update
```

matecommit also checks for new versions in the background and nudges you about it before most commands. If that gets annoying, set `MATECOMMIT_DISABLE_UPDATE_CHECK=1` and it'll leave you alone.

---

## Common Troubleshooting

**"The suggestions aren't very good"**
*   *Tip*: Make sure you only stage related changes. If you throw 5 different features into one stage, the AI will get confused by the context.

**"API Error" / something's not authenticating**
*   *Tip*: Run `matecommit doctor`. Your Gemini API key or GitHub token likely expired, is missing, or lacks the necessary scopes — `doctor` will tell you which.

**"My push got rejected out of nowhere"**
*   *Tip*: If you (or your org) have a GitHub ruleset protecting the branch or tag, that's expected — matecommit will surface GitHub's actual rejection reason instead of a generic git error. Push through a PR, or ask whoever manages the ruleset to adjust it.

---

## Current Support

*   **AI Models**: Google Gemini (Default).
*   **VCS**: GitHub.
*   **Issues**: Jira and GitHub Issues.
