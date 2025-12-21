package regex

import "regexp"

var (
	// Commit and Release patterns
	ConventionalCommit = regexp.MustCompile(`^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\(([^)]+)\))?(!)?:\s*(.+)`)
	BreakingChange     = regexp.MustCompile(`BREAKING[ -]CHANGE:\s*(.+)`)
	SemVer             = regexp.MustCompile(`v?(\d+)\.(\d+)\.(\d+)`)
	GitHubPR           = regexp.MustCompile(`\(#(\d+)\)`)

	// Issue and Ticket patterns
	JiraTicket             = regexp.MustCompile(`([A-Za-z]+-\d+)`)
	NumberedList           = regexp.MustCompile(`^\d+\.\s*`)
	MarkdownCheckbox       = regexp.MustCompile(`^\s*[\-*+]\s+\[([ xX])]\s+(.+)`)
	MarkdownCheckboxUpdate = regexp.MustCompile(`^(\s*[\-*+]\s+)\[([ xX])](\s+.+)`)

	// GitHub linkage patterns
	GitHubClosedLink = regexp.MustCompile(`(?i)(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?)\s+#(\d+)`)

	// Branch patterns for issue detection
	BranchIssueSharp  = regexp.MustCompile(`#(\d+)`)
	BranchIssueName   = regexp.MustCompile(`issue[/-](\d+)`)
	BranchIssueStart  = regexp.MustCompile(`^(\d+)-`)
	BranchIssueFolder = regexp.MustCompile(`/(\d+)-`)
	BranchIssueMid    = regexp.MustCompile(`-(\d+)-`)

	// Service specific patterns
	FixKeywords      = regexp.MustCompile(`(?i)(fix|bug|resolve|close)`)
	FeatKeywords     = regexp.MustCompile(`(?i)(feat|feature|add|implement)`)
	RefactorKeywords = regexp.MustCompile(`(?i)(refactor|restructure|reorganize)`)

	// Git and Repo patterns
	SSHRepo   = regexp.MustCompile(`git@([^:]+):([^/]+)/(.+)\.git$`)
	HTTPSRepo = regexp.MustCompile(`https://([^/]+)/([^/]+)/(.+?)(?:\.git)?$`)

	// Jira Acceptance Criteria cleanup
	AcceptanceCriteriaEN = regexp.MustCompile(`(?i)Acceptance criteria:.*(\n.*)*`)
	AcceptanceCriteriaES = regexp.MustCompile(`(?i)Criterio de aceptacion:.*(\n.*)*`)

	// AI and JSON parsing
	MarkdownJSONBlock = regexp.MustCompile("(?s)```(?:json)?\n?(.*?)```")
	JSONString        = regexp.MustCompile(`"(?:\\.|[^"\\])*"`)
	QuotedString      = regexp.MustCompile(`"(.*)"`)

	// Dependency management
	GoModRequireBlock  = regexp.MustCompile(`^\s+(\S+)\s+v?(\S+)(\s+//\s*indirect)?`)
	GoModRequireSingle = regexp.MustCompile(`^require\s+(\S+)\s+v?(\S+)(\s+//\s*indirect)?`)
)
