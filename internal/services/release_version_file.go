package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	domainErrors "github.com/thomas-vilte/matecommit/internal/errors"
	"github.com/thomas-vilte/matecommit/internal/logger"
	"github.com/thomas-vilte/matecommit/internal/regex"
)

const (
	maxSearchDepth = 5
	maxFilesToScan = 100
)

var ignoreDirs = map[string]bool{
	".git":          true,
	"node_modules":  true,
	"vendor":        true,
	"target":        true,
	"dist":          true,
	"build":         true,
	".next":         true,
	".nuxt":         true,
	"__pycache__":   true,
	".pytest_cache": true,
	"venv":          true,
	".venv":         true,
}

func (s *ReleaseService) UpdateAppVersion(ctx context.Context, version string) error {
	log := logger.FromContext(ctx)

	versionFile, versionPattern, err := s.FindVersionFile(ctx)
	if err != nil {
		if s.config == nil || s.config.VersionFile == "" {
			log.Warn("could not auto-detect version file, skipping version update", "error", err)
			return nil
		}

		log.Warn("could not auto-detect version file, using configured version file", "error", err)
		versionFile = s.config.VersionFile

		if s.config != nil {
			if s.config.VersionPattern != "" {
				versionPattern = s.config.VersionPattern
			}
		}
	}

	log.Debug("updating version",
		"file", versionFile,
		"pattern", versionPattern,
		"new_version", version)

	content, err := os.ReadFile(versionFile)
	if err != nil {
		if os.IsNotExist(err) && (s.config == nil || s.config.VersionFile == "") {
			log.Warn("version file not found, skipping version update", "file", versionFile)
			return nil
		}
		return domainErrors.NewAppError(domainErrors.TypeInternal,
			fmt.Sprintf("failed to read version file: %s", versionFile), err)
	}

	re, err := regexp.Compile(versionPattern)
	if err != nil {
		return domainErrors.NewAppError(domainErrors.TypeInternal,
			fmt.Sprintf("invalid version pattern: %s", versionPattern), err)
	}

	currentContent := string(content)
	if !re.MatchString(currentContent) {
		return domainErrors.NewAppError(domainErrors.TypeInternal,
			fmt.Sprintf("version pattern not found in %s with pattern %s", versionFile, versionPattern), nil)
	}

	match := re.FindString(currentContent)
	if match == "" {
		return domainErrors.NewAppError(domainErrors.TypeInternal,
			"could not find version match", nil)
	}

	cleanVersion := strings.TrimPrefix(version, "v")

	var newMatch string
	ext := filepath.Ext(versionFile)

	switch ext {
	case ".json":
		versionRe := regexp.MustCompile(`"version"\s*:\s*"([^"]+)"`)
		newMatch = versionRe.ReplaceAllString(match, fmt.Sprintf(`"version": "%s"`, cleanVersion))
	case ".toml":
		versionRe := regexp.MustCompile(`version\s*=\s*"([^"]+)"`)
		newMatch = versionRe.ReplaceAllString(match, fmt.Sprintf(`version = "%s"`, cleanVersion))
	case ".xml":
		versionRe := regexp.MustCompile(`<version>([^<]+)</version>`)
		newMatch = versionRe.ReplaceAllString(match, fmt.Sprintf(`<version>%s</version>`, cleanVersion))
	case ".py":
		if strings.Contains(match, `"`) {
			versionRe := regexp.MustCompile(`"([^"]+)"`)
			newMatch = versionRe.ReplaceAllString(match, fmt.Sprintf(`"%s"`, cleanVersion))
		} else if strings.Contains(match, `'`) {
			versionRe := regexp.MustCompile(`'([^']+)'`)
			newMatch = versionRe.ReplaceAllString(match, fmt.Sprintf(`'%s'`, cleanVersion))
		}
	case ".go", ".js", ".ts", ".rs", ".php", ".rb":
		if strings.Contains(match, `"`) {
			valMatch := regex.QuotedString.FindStringIndex(match)
			if valMatch != nil {
				newMatch = match[:valMatch[0]] + fmt.Sprintf(`"%s"`, cleanVersion) + match[valMatch[1]:]
			} else {
				versionRe := regexp.MustCompile(`"([^"]+)"`)
				newMatch = versionRe.ReplaceAllString(match, fmt.Sprintf(`"%s"`, cleanVersion))
			}
		} else if strings.Contains(match, `'`) {
			versionRe := regexp.MustCompile(`'([^']+)'`)
			newMatch = versionRe.ReplaceAllString(match, fmt.Sprintf(`'%s'`, cleanVersion))
		} else {
			versionRe := regexp.MustCompile(`[\d.]+`)
			newMatch = versionRe.ReplaceAllString(match, cleanVersion)
		}
	default:
		if strings.Contains(match, `"`) {
			versionRe := regexp.MustCompile(`"([^"]+)"`)
			newMatch = versionRe.ReplaceAllString(match, fmt.Sprintf(`"%s"`, cleanVersion))
		} else {
			versionRe := regexp.MustCompile(`[\d.]+`)
			newMatch = versionRe.ReplaceAllString(match, cleanVersion)
		}
	}

	if newMatch == "" {
		newMatch = match
	}

	newContent := strings.Replace(currentContent, match, newMatch, 1)

	if err := os.WriteFile(versionFile, []byte(newContent), 0644); err != nil {
		return domainErrors.NewAppError(domainErrors.TypeInternal,
			fmt.Sprintf("failed to write version file: %s", versionFile), err)
	}

	log.Info("version updated successfully",
		"file", versionFile,
		"version", version)

	return nil
}

func (s *ReleaseService) FindVersionFile(ctx context.Context) (string, string, error) {
	log := logger.FromContext(ctx)

	if s.versionFileCache.file != "" && s.versionFileCache.pattern != "" {
		log.Debug("using cached version file",
			"file", s.versionFileCache.file,
			"pattern", s.versionFileCache.pattern,
		)
		return s.versionFileCache.file, s.versionFileCache.pattern, nil
	}

	if s.config != nil && s.config.VersionFile != "" {
		pattern := s.config.VersionPattern
		if pattern == "" {
			detectedPattern, err := s.detectPatternInFile(s.config.VersionFile)
			if err == nil && detectedPattern != "" {
				pattern = detectedPattern
			} else {
				pattern = `Version:\s*".*"`
			}
		}
		log.Debug("using configured version file",
			"file", s.config.VersionFile,
			"pattern", pattern)

		s.versionFileCache.file = s.config.VersionFile
		s.versionFileCache.pattern = pattern

		return s.config.VersionFile, pattern, nil
	}

	projectType := s.detectProjectType()
	log.Debug("detected project type", "type", projectType)

	if files, ok := versionFilesByLanguage[projectType]; ok {
		for _, filePath := range files {
			if strings.Contains(filePath, "*") {
				matches, err := filepath.Glob(filePath)
				if err == nil && len(matches) > 0 {
					filePath = matches[0]
				} else {
					continue
				}
			}

			if _, err := os.Stat(filePath); err == nil {
				pattern, err := s.detectPatternInFileForLanguage(filePath, projectType)
				if err == nil && pattern != "" {
					log.Debug("version file found automatically",
						"file", filePath,
						"pattern", pattern,
						"language", projectType,
					)

					s.versionFileCache.file = filePath
					s.versionFileCache.pattern = pattern
					s.versionFileCache.lang = projectType
					return filePath, pattern, nil
				}
			}
		}
	}

	foundFile, foundPattern, err := s.searchVersionFileRecursive(projectType)
	if err == nil && foundFile != "" {
		log.Debug("version file found recursively",
			"file", foundFile,
			"pattern", foundPattern,
		)

		s.versionFileCache.file = foundFile
		s.versionFileCache.pattern = foundPattern
		s.versionFileCache.lang = projectType

		return foundFile, foundPattern, nil
	}

	if files, ok := versionFilesByLanguage[projectType]; ok {
		return "", "", domainErrors.ErrVersionFileNotFound.
			WithContext("project_type", projectType).
			WithContext("searched_paths", files)
	}

	return "", "", domainErrors.ErrVersionFileNotFound.
		WithContext("project_type", projectType)
}

type versionPatternInfo struct {
	name               string
	detectionPattern   string
	replacementPattern string
}

var consolidatedVersionPatterns = map[string][]versionPatternInfo{
	"go": {
		{
			name:               "const Version",
			detectionPattern:   `const\s+Version\s*=\s*"([^"]+)"`,
			replacementPattern: `const\s+Version\s*=\s*"[^"]*"`,
		},
		{
			name:               "var Version",
			detectionPattern:   `var\s+Version\s*=\s*"([^"]+)"`,
			replacementPattern: `var\s+Version\s*=\s*"[^"]*"`,
		},
		{
			name:               "const Version unquoted",
			detectionPattern:   `const\s+Version\s*=\s*([\d.]+)`,
			replacementPattern: `const\s+Version\s*=\s*[\d.]+`,
		},
		{
			name:               "var Version unquoted",
			detectionPattern:   `var\s+Version\s*=\s*([\d.]+)`,
			replacementPattern: `var\s+Version\s*=\s*[\d.]+`,
		},
		{
			name:               "Version:",
			detectionPattern:   `Version:\s*"([^"]+)"`,
			replacementPattern: `Version:\s*"[^"]*"`,
		},
		{
			name:               "Version =",
			detectionPattern:   `Version\s*=\s*"([^"]+)"`,
			replacementPattern: `Version\s*=\s*"[^"]*"`,
		},
	},
	"python": {
		{
			name:               "__version__ double quotes",
			detectionPattern:   `__version__\s*=\s*"([^"]+)"`,
			replacementPattern: `__version__\s*=\s*"[^"]*"`,
		},
		{
			name:               "__version__ single quotes",
			detectionPattern:   `__version__\s*=\s*'([^']+)'`,
			replacementPattern: `__version__\s*=\s*'[^']*'`,
		},
		{
			name:               "version double quotes",
			detectionPattern:   `version\s*=\s*"([^"]+)"`,
			replacementPattern: `version\s*=\s*"[^"]*"`,
		},
		{
			name:               "version any quotes",
			detectionPattern:   `version\s*=\s*['"]([^'"]+)['"]`,
			replacementPattern: `version\s*=\s*['"][^'"]*['"]`,
		},
		{
			name:               "VERSION",
			detectionPattern:   `VERSION\s*=\s*"([^"]+)"`,
			replacementPattern: `VERSION\s*=\s*"[^"]*"`,
		},
	},
	"js": {
		{
			name:               "version in JSON",
			detectionPattern:   `"version"\s*:\s*"([^"]+)"`,
			replacementPattern: `"version"\s*:\s*"[^"]*"`,
		},
		{
			name:               "version in JSON single quotes",
			detectionPattern:   `'version'\s*:\s*'([^']+)'`,
			replacementPattern: `'version'\s*:\s*'[^']*'`,
		},
		{
			name:               "export const version",
			detectionPattern:   `export\s+const\s+version\s*=\s*"([^"]+)"`,
			replacementPattern: `export\s+const\s+version\s*=\s*"[^"]*"`,
		},
		{
			name:               "export const version single quotes",
			detectionPattern:   `export\s+const\s+version\s*=\s*'([^']+)'`,
			replacementPattern: `export\s+const\s+version\s*=\s*'[^']*'`,
		},
	},
	"rust": {
		{
			name:               "version in TOML",
			detectionPattern:   `version\s*=\s*"([^"]+)"`,
			replacementPattern: `version\s*=\s*"[^"]*"`,
		},
	},
	"java": {
		{
			name:               "version in XML",
			detectionPattern:   `<version>([^<]+)</version>`,
			replacementPattern: `<version>[^<]+</version>`,
		},
		{
			name:               "version in properties",
			detectionPattern:   `version\s*=\s*['"]([^'"]+)['"]`,
			replacementPattern: `version\s*=\s*['"][^'"]*['"]`,
		},
	},
	"csharp": {
		{
			name:               "AssemblyVersion",
			detectionPattern:   `AssemblyVersion\s*\(\s*"([^"]+)"`,
			replacementPattern: `AssemblyVersion\s*\(\s*"[^"]*"`,
		},
		{
			name:               "Version in XML",
			detectionPattern:   `<Version>([^<]+)</Version>`,
			replacementPattern: `<Version>[^<]+</Version>`,
		},
		{
			name:               "version in JSON",
			detectionPattern:   `"version"\s*:\s*"([^"]+)"`,
			replacementPattern: `"version"\s*:\s*"[^"]*"`,
		},
	},
	"php": {
		{
			name:               "version in JSON",
			detectionPattern:   `"version"\s*:\s*"([^"]+)"`,
			replacementPattern: `"version"\s*:\s*"[^"]*"`,
		},
		{
			name:               "const VERSION",
			detectionPattern:   `const\s+VERSION\s*=\s*['"]([^'"]+)['"]`,
			replacementPattern: `const\s+VERSION\s*=\s*['"][^'"]*['"]`,
		},
	},
	"ruby": {
		{
			name:               "VERSION",
			detectionPattern:   `VERSION\s*=\s*['"]([^'"]+)['"]`,
			replacementPattern: `VERSION\s*=\s*['"][^'"]*['"]`,
		},
		{
			name:               ".version",
			detectionPattern:   `\.version\s*=\s*['"]([^'"]+)['"]`,
			replacementPattern: `\.version\s*=\s*['"][^'"]*['"]`,
		},
	},
}

func (s *ReleaseService) detectPatternInFile(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	fileContent := string(content)
	lang := detectLanguageFromFile(filePath)

	if patterns, ok := consolidatedVersionPatterns[lang]; ok {
		for _, patternInfo := range patterns {
			re, err := regexp.Compile(patternInfo.detectionPattern)
			if err != nil {
				continue
			}
			if re.MatchString(fileContent) {
				matches := re.FindStringSubmatch(fileContent)
				if len(matches) > 1 {
					if err := s.validateVersionString(matches[1]); err != nil {
						continue
					}
				}
				return patternInfo.replacementPattern, nil
			}
		}
	}

	for _, patterns := range consolidatedVersionPatterns {
		for _, patternInfo := range patterns {
			re, err := regexp.Compile(patternInfo.detectionPattern)
			if err != nil {
				continue
			}
			if re.MatchString(fileContent) {
				matches := re.FindStringSubmatch(fileContent)
				if len(matches) > 1 {
					if err := s.validateVersionString(matches[1]); err != nil {
						continue
					}
				}
				return patternInfo.replacementPattern, nil
			}
		}
	}

	return "", fmt.Errorf("no version pattern found in file")
}

func (s *ReleaseService) searchVersionFileRecursive(lang string) (string, string, error) {
	searchDirs := map[string][]string{
		"go":     {"internal", "pkg", "version", "cmd"},
		"python": {"src", "lib", "."},
		"js":     {"src", "lib", "."},
		"rust":   {"."},
		"java":   {"src", "."},
		"csharp": {"Properties", "."},
		"php":    {"src", "."},
		"ruby":   {"lib", "."},
	}

	dirs, ok := searchDirs[lang]
	if !ok {
		dirs = []string{"."}
	}

	extensions := map[string][]string{
		"go":     {".go"},
		"python": {".py"},
		"js":     {".js", ".ts"},
		"rust":   {".rs", ".toml"},
		"java":   {".xml", ".gradle", ".properties"},
		"csharp": {".cs", ".csproj", ".props"},
		"php":    {".php", ".json"},
		"ruby":   {".rb", ".gemspec"},
	}

	exts, ok := extensions[lang]
	if !ok {
		exts = []string{""}
	}

	filesScanned := 0

	for _, dir := range dirs {
		var foundPath string
		var foundPattern string

		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if filesScanned >= maxFilesToScan {
				return filepath.SkipDir
			}

			if info.IsDir() {
				if ignoreDirs[info.Name()] {
					return filepath.SkipDir
				}

				depth := len(strings.Split(path, string(os.PathSeparator)))
				if depth > maxSearchDepth {
					return filepath.SkipDir
				}
				return nil
			}

			filesScanned++

			hasValidExt := false
			for _, ext := range exts {
				if ext == "" || strings.HasSuffix(path, ext) {
					hasValidExt = true
					break
				}
			}
			if !hasValidExt {
				return nil
			}

			if strings.Contains(strings.ToLower(path), "version") ||
				strings.Contains(strings.ToLower(info.Name()), "version") ||
				path == "package.json" || path == "Cargo.toml" || path == "setup.py" {
				pattern, err := s.detectPatternInFileForLanguage(path, lang)
				if err == nil && pattern != "" {
					foundPath = path
					foundPattern = pattern
					return fmt.Errorf("found: %s", path)
				}
			}
			return nil
		})

		if err != nil && strings.HasPrefix(err.Error(), "found: ") {
			return foundPath, foundPattern, nil
		}
	}

	return "", "", fmt.Errorf("version file not found")
}

func (s *ReleaseService) detectProjectType() string {
	indicators := map[string][]string{
		"go":     {"go.mod", "Gopkg.toml", "glide.yaml"},
		"python": {"setup.py", "pyproject.toml", "requirements.txt", "Pipfile"},
		"js":     {"package.json", "yarn.lock", "package-lock.json"},
		"rust":   {"Cargo.toml"},
		"java":   {"pom.xml", "build.gradle", "build.gradle.kts"},
		"csharp": {".csproj", ".sln", "project.json"},
		"php":    {"composer.json"},
		"ruby":   {"Gemfile", "Rakefile"},
	}

	for lang, files := range indicators {
		for _, file := range files {
			if _, err := os.Stat(file); err == nil {
				return lang
			}
		}
	}

	if hasFilesWithExtension(".go") {
		return "go"
	}
	if hasFilesWithExtension(".py") {
		return "python"
	}
	if hasFilesWithExtension(".js") || hasFilesWithExtension(".ts") {
		return "js"
	}
	if hasFilesWithExtension(".rs") {
		return "rust"
	}

	return "unknown"
}

func hasFilesWithExtension(ext string) bool {
	entries, err := os.ReadDir(".")
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ext) {
			return true
		}
	}
	return false
}

var versionFilesByLanguage = map[string][]string{
	"go": {
		"internal/version/version.go",
		"pkg/version/version.go",
		"version/version.go",
		"version.go",
		"cmd/main.go",
		"main.go",
		"internal/version.go",
		"pkg/version.go",
	},
	"python": {
		"__version__.py",
		"version.py",
		"src/*/__version__.py",
		"setup.py",
		"pyproject.toml",
	},
	"js": {
		"package.json",
		"src/version.js",
		"src/version.ts",
		"lib/version.js",
	},
	"rust": {
		"Cargo.toml",
	},
	"java": {
		"pom.xml",
		"build.gradle",
		"build.gradle.kts",
		"src/main/resources/version.properties",
	},
	"csharp": {
		"Properties/AssemblyInfo.cs",
		"Directory.Build.props",
		"*.csproj",
	},
	"php": {
		"composer.json",
		"src/Version.php",
	},
	"ruby": {
		"lib/*/version.rb",
		"version.rb",
		"*.gemspec",
	},
}

func (s *ReleaseService) detectPatternInFileForLanguage(filePath, lang string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	fileContent := string(content)
	patterns, ok := consolidatedVersionPatterns[lang]
	if !ok {
		return s.detectPatternInFile(filePath)
	}

	for _, patternInfo := range patterns {
		re, err := regexp.Compile(patternInfo.detectionPattern)
		if err != nil {
			continue
		}
		if re.MatchString(fileContent) {
			matches := re.FindStringSubmatch(fileContent)
			if len(matches) > 1 {
				versionValue := matches[1]
				if err := s.validateVersionString(versionValue); err != nil {
					continue
				}
			}
			return patternInfo.replacementPattern, nil
		}
	}

	return "", fmt.Errorf("no version pattern found in file")
}

func detectLanguageFromFile(filePath string) string {
	ext := filepath.Ext(filePath)
	extToLang := map[string]string{
		".go":      "go",
		".py":      "python",
		".js":      "js",
		".ts":      "js",
		".rs":      "rust",
		".toml":    "rust",
		".xml":     "java",
		".cs":      "csharp",
		".csproj":  "csharp",
		".props":   "csharp",
		".php":     "php",
		".rb":      "ruby",
		".gemspec": "ruby",
		".json":    "js",
	}

	filename := strings.ToLower(filepath.Base(filePath))
	if filename == "package.json" || filename == "package-lock.json" {
		return "js"
	}
	if filename == "composer.json" {
		return "php"
	}
	if filename == "cargo.toml" {
		return "rust"
	}
	if filename == "pom.xml" {
		return "java"
	}
	if filename == "setup.py" || filename == "pyproject.toml" {
		return "python"
	}

	if lang, ok := extToLang[ext]; ok {
		return lang
	}

	return "unknown"
}
