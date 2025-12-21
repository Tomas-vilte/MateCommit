# Changelog

All notable changes to this project will be documented in this file.

## [v1.4.0]

[v1.4.0]: https://github.com/Tomas-vilte/MateCommit/compare/v1.3.0...v1.4.0

In this release, we focused on transforming your interaction with the tool by drastically improving the user experience with real-time visual feedback and optimizing release process automation. Additionally, we've enhanced the AI for deeper and more contextualized analysis.

### Highlights

- **Renovated User Experience:** Implemented spinners, colors, and change previews (diff) for real-time visual feedback. Added a `doctor` command to validate your Gemini API key and improved commit previews, allowing message editing before confirmation.
- **Comprehensive Release Automation:** Simplified and automated release note generation for `CHANGELOG.md`, version updates, and automatic changelog commits. You can now also edit existing releases, streamlining your workflow.
- **Contextualized AI:** Enhanced the AI's ability to understand the context of your commits and Pull Requests (PRs). It now automatically detects issues, breaking changes, and test plans, enriching PR summaries and release notes with more relevant information.
- **Multi-language Dependency Analysis:** Added the ability to analyze dependency changes in your projects, even in multi-language environments, providing a more complete and detailed view of each release.
- **CLI Improvements:** Implemented autocompletion for commands and flags, making your terminal experience smoother and more efficient.
- **Fixes and Stability:** Ensured the PR summarizer correctly uses the JSON format template, guaranteeing consistency in summary generation.

## [v1.3.0] - 2025-12-09

[v1.3.0]: https://github.com/Tomas-vilte/MateCommit/compare/v1.2.0...v1.3.0

In this release, we focused on further simplifying and automating your workflows, from release management to Pull Request interaction. We also improved the configuration experience and general application stability.

### Highlights

- **Simplified Release Management:** You can now generate and publish new versions more smoothly with dedicated commands and a prompt assistant using a natural first-person tone.
- **Renewed CLI Configuration & Assistance:** Introduced `config init` to guide you through initial setup, an `edit` command to easily adjust parameters, and a `help` command. Optimized VCS configuration guides and added a 'performance' tag for AI.
- **Enhanced PR Interaction:** Automatically detect repository information for PR commands, validate and normalize tags, and handle large diffs better with a fallback system.
- **Improved Localization:** Added internationalized messages for GitHub token permission errors and large PR diff processing, providing clearer feedback in both English and Spanish.
- **AI Model Updates:** Migrated to Gemini v1.5/2.0 models for more accurate and efficient responses.
- **Stability Fixes:** Resolved AI service errors in specific scenarios, improved `git add` precision, and corrected spelling in Spanish messages.

## [v1.2.0] - 2025-02-18

[v1.2.0]: https://github.com/Tomas-vilte/MateCommit/compare/v1.1.1...v1.2.0

In this version of MateCommit, we're excited to introduce a time-saving feature: Pull Request Summarization. We've also strengthened the application by improving error handling and adaptability.

### Highlights

- **Pull Request Summary:** Added the `summarize-pr` command to get concise PR summaries directly from the terminal. Implemented a robust GitHub client for smoother repository interaction.
- **Improved Stability & Adaptability:** Significantly optimized error handling and expanded internationalization for a more adaptable and reliable experience.

## [v1.1.1] - 2025-02-06

[v1.1.1]: https://github.com/Tomas-vilte/MateCommit/compare/v1.1.0...v1.1.1

Focused on strengthening application robustness with key error handling improvements for more stable operations and precise feedback.

### Highlights

- **Enhanced Error Handling:** Reinforced error management when staging files, providing clearer information if issues arise.

## [v1.1.0] - 2025-02-05

[v1.1.0]: https://github.com/Tomas-vilte/MateCommit/compare/v1.0.0...v1.1.0

Expanded CLI capabilities, allowing users to configure AI settings directly from the terminal for greater personalization and control.

### Highlights

- **AI Configuration via CLI:** New functionality to configure AI settings directly from the command line, offering an integrated experience for advanced users.

## [v1.0.0] - 2025-01-15

[v1.0.0]: https://github.com/Tomas-vilte/MateCommit/releases/tag/v1.0.0

The first version of MateCommit focuses on boosting your workflow with AI integration. Generating descriptive and professional commit messages is now easier than ever.

### Highlights

- **AI-Powered Commit Suggestions:** Integration with AI models (like Gemini) to offer intelligent and relevant commit message suggestions.
- **Improved CLI Welcome:** Added a greeting message to make the user experience friendlier from the start.
