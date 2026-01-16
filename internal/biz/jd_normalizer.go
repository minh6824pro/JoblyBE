package biz

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// =============================================================================
// JD NORMALIZER - ONTOLOGY-BASED TEXT NORMALIZATION FOR JOB DESCRIPTIONS
// =============================================================================
// This module provides ontology-based normalization for job descriptions
// in the TECH/IT domain. It standardizes:
// - Skills/Technologies
// - Job Titles
// - Experience Levels
// - Requirements
//
// Usage: Call NormalizeJobPosting() before embedding to ensure consistent
// vector representations across different JD writing styles.
//
// The ontology is loaded from configs/ontology.json file.
// =============================================================================

// OntologyConfig represents the JSON structure of ontology file
type OntologyConfig struct {
	Skills              map[string]map[string]SkillEntry `json:"skills"`
	JobTitles           map[string]map[string][]string   `json:"jobTitles"`
	Levels              map[string]LevelEntry            `json:"levels"`
	SeniorityKeywords   map[string][]string              `json:"seniorityKeywords"`
	RequirementKeywords map[string][]string              `json:"requirementKeywords"`
}

// SkillEntry represents a skill in the ontology
type SkillEntry struct {
	Aliases []string `json:"aliases"`
	Related []string `json:"related"`
}

// LevelEntry represents a level in the ontology
type LevelEntry struct {
	Aliases        []string `json:"aliases"`
	YearsRange     string   `json:"yearsRange"`
	Responsibility string   `json:"responsibility"`
}

// JDNormalizer handles job description normalization using ontology
type JDNormalizer struct {
	skillOntology       *SkillOntology
	titleOntology       *JobTitleOntology
	levelOntology       *LevelOntology
	requirementOntology *RequirementOntology
	configPath          string
	mu                  sync.RWMutex
}

// NewJDNormalizer creates a new JD normalizer with ontology loaded from default config path
func NewJDNormalizer() *JDNormalizer {
	return NewJDNormalizerWithConfig("")
}

// NewJDNormalizerWithConfig creates a new JD normalizer with ontology loaded from config file
func NewJDNormalizerWithConfig(configPath string) *JDNormalizer {
	n := &JDNormalizer{
		configPath: configPath,
	}

	// Try to load from config file
	config, err := n.loadOntologyConfig()
	if err != nil {
		// Log warning but continue with empty ontologies
		fmt.Printf("Warning: Could not load ontology config: %v. Using empty ontology.\n", err)
		n.skillOntology = NewSkillOntology()
		n.titleOntology = NewJobTitleOntology()
		n.levelOntology = NewLevelOntology()
		n.requirementOntology = NewRequirementOntology()
	} else {
		// Initialize from config
		n.skillOntology = NewSkillOntologyFromConfig(config)
		n.titleOntology = NewJobTitleOntologyFromConfig(config)
		n.levelOntology = NewLevelOntologyFromConfig(config)
		n.requirementOntology = NewRequirementOntologyFromConfig(config)
	}

	return n
}

// loadOntologyConfig loads ontology configuration from JSON file
func (n *JDNormalizer) loadOntologyConfig() (*OntologyConfig, error) {
	var configPath string

	if n.configPath != "" {
		configPath = n.configPath
	} else {
		// Try common paths
		possiblePaths := []string{
			"configs/ontology.json",
			"../configs/ontology.json",
			"../../configs/ontology.json",
			"./ontology.json",
		}

		// Also try relative to executable
		if execPath, err := os.Executable(); err == nil {
			execDir := filepath.Dir(execPath)
			possiblePaths = append(possiblePaths,
				filepath.Join(execDir, "configs", "ontology.json"),
				filepath.Join(execDir, "..", "configs", "ontology.json"),
			)
		}

		// Try working directory
		if wd, err := os.Getwd(); err == nil {
			possiblePaths = append(possiblePaths,
				filepath.Join(wd, "configs", "ontology.json"),
			)
		}

		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				configPath = path
				break
			}
		}

		if configPath == "" {
			return nil, fmt.Errorf("ontology.json not found in any of the expected paths")
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read ontology config: %w", err)
	}

	var config OntologyConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse ontology config: %w", err)
	}

	fmt.Printf("Loaded ontology config from: %s\n", configPath)
	return &config, nil
}

// ReloadOntology reloads the ontology from config file
func (n *JDNormalizer) ReloadOntology() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	config, err := n.loadOntologyConfig()
	if err != nil {
		return err
	}

	n.skillOntology = NewSkillOntologyFromConfig(config)
	n.titleOntology = NewJobTitleOntologyFromConfig(config)
	n.levelOntology = NewLevelOntologyFromConfig(config)
	n.requirementOntology = NewRequirementOntologyFromConfig(config)

	return nil
}

// =============================================================================
// NORMALIZED RESULT STRUCTURES
// =============================================================================

// NormalizedJobData contains normalized job posting data ready for embedding
type NormalizedJobData struct {
	// Original values
	OriginalTitle        string
	OriginalSkills       []string
	OriginalRequirements string
	OriginalLevel        string

	// Normalized values
	NormalizedTitle        string
	NormalizedSkills       []string
	NormalizedRequirements string
	NormalizedLevel        string

	// Enriched data from ontology
	SkillCategories     map[string][]string // Category -> Skills
	RelatedSkills       []string            // Additional related skills detected
	TitleCategory       string              // e.g., "Backend", "Frontend", "DevOps"
	TitleSeniority      string              // Extracted seniority from title
	RequirementTypes    map[string][]string // Categorized requirements
	LevelYearsRange     string              // e.g., "3-5 years"
	LevelResponsibility string              // e.g., "Individual Contributor"
}

// =============================================================================
// MAIN NORMALIZATION FUNCTIONS
// =============================================================================

// NormalizeJobPosting normalizes all relevant fields of a job posting
func (n *JDNormalizer) NormalizeJobPosting(job *JobPosting) *NormalizedJobData {
	n.mu.RLock()
	defer n.mu.RUnlock()

	result := &NormalizedJobData{
		OriginalTitle:        job.Title,
		OriginalSkills:       job.JobTech,
		OriginalRequirements: job.Requirements,
		OriginalLevel:        string(job.Level),
		SkillCategories:      make(map[string][]string),
		RequirementTypes:     make(map[string][]string),
	}

	// Normalize each field
	result.NormalizedTitle, result.TitleCategory, result.TitleSeniority = n.NormalizeTitle(job.Title)
	result.NormalizedSkills, result.SkillCategories, result.RelatedSkills = n.NormalizeSkills(job.JobTech)
	result.NormalizedRequirements, result.RequirementTypes = n.NormalizeRequirements(job.Requirements)
	result.NormalizedLevel, result.LevelYearsRange, result.LevelResponsibility = n.NormalizeLevel(string(job.Level))

	return result
}

// NormalizeTitle normalizes a job title and extracts category/seniority
func (n *JDNormalizer) NormalizeTitle(title string) (normalized string, category string, seniority string) {
	return n.titleOntology.Normalize(title)
}

// NormalizeSkills normalizes a list of skills and categorizes them
func (n *JDNormalizer) NormalizeSkills(skills []string) (normalized []string, categories map[string][]string, related []string) {
	return n.skillOntology.NormalizeSkills(skills)
}

// NormalizeRequirements normalizes requirements text and categorizes them
func (n *JDNormalizer) NormalizeRequirements(requirements string) (normalized string, categorized map[string][]string) {
	return n.requirementOntology.Normalize(requirements)
}

// NormalizeLevel normalizes experience level
func (n *JDNormalizer) NormalizeLevel(level string) (normalized string, yearsRange string, responsibility string) {
	return n.levelOntology.Normalize(level)
}

// =============================================================================
// TEXT PREPARATION FOR EMBEDDING
// =============================================================================

// PrepareTextForEmbedding prepares normalized data as text suitable for embedding
func (n *JDNormalizer) PrepareTextForEmbedding(data *NormalizedJobData) map[string]string {
	texts := make(map[string]string)

	// Title text with enriched context
	texts["title"] = n.prepareTitleText(data)

	// Skills text with categories
	texts["skills"] = n.prepareSkillsText(data)

	// Requirements text with structure
	texts["requirements"] = n.prepareRequirementsText(data)

	// Level text with context
	texts["level"] = n.prepareLevelText(data)

	// Full combined text
	texts["full"] = n.prepareFullText(data)

	return texts
}

func (n *JDNormalizer) prepareTitleText(data *NormalizedJobData) string {
	var parts []string
	parts = append(parts, "Job Title: "+data.NormalizedTitle)
	if data.TitleCategory != "" {
		parts = append(parts, "Category: "+data.TitleCategory)
	}
	if data.TitleSeniority != "" {
		parts = append(parts, "Seniority: "+data.TitleSeniority)
	}
	return strings.Join(parts, ". ")
}

func (n *JDNormalizer) prepareSkillsText(data *NormalizedJobData) string {
	var parts []string

	// Main skills
	if len(data.NormalizedSkills) > 0 {
		parts = append(parts, "Technical Skills: "+strings.Join(data.NormalizedSkills, ", "))
	}

	// Categorized skills
	for category, skills := range data.SkillCategories {
		if len(skills) > 0 {
			parts = append(parts, category+": "+strings.Join(skills, ", "))
		}
	}

	// Related skills for enrichment
	if len(data.RelatedSkills) > 0 {
		parts = append(parts, "Related Technologies: "+strings.Join(data.RelatedSkills, ", "))
	}

	return strings.Join(parts, ". ")
}

func (n *JDNormalizer) prepareRequirementsText(data *NormalizedJobData) string {
	var parts []string

	// Original normalized requirements
	if data.NormalizedRequirements != "" {
		parts = append(parts, data.NormalizedRequirements)
	}

	// Categorized requirements
	for category, reqs := range data.RequirementTypes {
		if len(reqs) > 0 {
			parts = append(parts, category+": "+strings.Join(reqs, "; "))
		}
	}

	return strings.Join(parts, "\n")
}

func (n *JDNormalizer) prepareLevelText(data *NormalizedJobData) string {
	var parts []string
	parts = append(parts, "Experience Level: "+data.NormalizedLevel)
	if data.LevelYearsRange != "" {
		parts = append(parts, "Experience: "+data.LevelYearsRange)
	}
	if data.LevelResponsibility != "" {
		parts = append(parts, "Responsibility: "+data.LevelResponsibility)
	}
	return strings.Join(parts, ". ")
}

func (n *JDNormalizer) prepareFullText(data *NormalizedJobData) string {
	return strings.Join([]string{
		n.prepareTitleText(data),
		n.prepareLevelText(data),
		n.prepareSkillsText(data),
		n.prepareRequirementsText(data),
	}, "\n\n")
}

// =============================================================================
// SKILL ONTOLOGY
// =============================================================================

// SkillOntology defines the tech/IT skill taxonomy
type SkillOntology struct {
	// Canonical name -> Aliases
	skillAliases map[string][]string
	// Skill -> Category
	skillCategories map[string]string
	// Skill -> Related Skills
	relatedSkills map[string][]string
	// Compiled regex for skill detection
	skillPatterns map[string]*regexp.Regexp
}

// NewSkillOntology creates a new empty skill ontology
func NewSkillOntology() *SkillOntology {
	return &SkillOntology{
		skillAliases:    make(map[string][]string),
		skillCategories: make(map[string]string),
		relatedSkills:   make(map[string][]string),
		skillPatterns:   make(map[string]*regexp.Regexp),
	}
}

// NewSkillOntologyFromConfig creates a skill ontology from config
func NewSkillOntologyFromConfig(config *OntologyConfig) *SkillOntology {
	o := NewSkillOntology()
	o.loadFromConfig(config)
	return o
}

func (o *SkillOntology) loadFromConfig(config *OntologyConfig) {
	for category, skills := range config.Skills {
		for skillName, skillEntry := range skills {
			o.addSkill(skillName, category, skillEntry.Aliases, skillEntry.Related)
		}
	}
	o.compilePatterns()
}

func (o *SkillOntology) addSkill(canonical string, category string, aliases []string, related []string) {
	o.skillAliases[canonical] = aliases
	o.skillCategories[canonical] = category
	o.relatedSkills[canonical] = related
}

func (o *SkillOntology) compilePatterns() {
	for canonical, aliases := range o.skillAliases {
		allNames := append([]string{canonical}, aliases...)
		// Escape special regex characters and create pattern
		escapedNames := make([]string, len(allNames))
		for i, name := range allNames {
			escapedNames[i] = regexp.QuoteMeta(name)
		}
		pattern := `(?i)\b(` + strings.Join(escapedNames, "|") + `)\b`
		o.skillPatterns[canonical] = regexp.MustCompile(pattern)
	}
}

// NormalizeSkills normalizes a list of skills
func (o *SkillOntology) NormalizeSkills(skills []string) (normalized []string, categories map[string][]string, related []string) {
	categories = make(map[string][]string)
	normalizedSet := make(map[string]bool)
	relatedSet := make(map[string]bool)

	for _, skill := range skills {
		// Try to find canonical form
		canonical := o.findCanonical(skill)
		if canonical != "" {
			if !normalizedSet[canonical] {
				normalizedSet[canonical] = true
				normalized = append(normalized, canonical)

				// Add to category
				if cat, ok := o.skillCategories[canonical]; ok {
					categories[cat] = append(categories[cat], canonical)
				}

				// Collect related skills
				if rel, ok := o.relatedSkills[canonical]; ok {
					for _, r := range rel {
						if !normalizedSet[r] {
							relatedSet[r] = true
						}
					}
				}
			}
		} else {
			// Keep original if not found
			if !normalizedSet[skill] {
				normalizedSet[skill] = true
				normalized = append(normalized, skill)
			}
		}
	}

	// Convert related set to slice
	for r := range relatedSet {
		related = append(related, r)
	}

	return normalized, categories, related
}

func (o *SkillOntology) findCanonical(skill string) string {
	skill = strings.TrimSpace(skill)
	skillLower := strings.ToLower(skill)

	// Check if it's already canonical
	for canonical := range o.skillAliases {
		if strings.EqualFold(canonical, skill) {
			return canonical
		}
	}

	// Check aliases
	for canonical, aliases := range o.skillAliases {
		for _, alias := range aliases {
			if strings.EqualFold(alias, skill) {
				return canonical
			}
		}
	}

	// Try pattern matching
	for canonical, pattern := range o.skillPatterns {
		if pattern.MatchString(skillLower) {
			return canonical
		}
	}

	return ""
}

// DetectSkillsInText extracts skills mentioned in text
func (o *SkillOntology) DetectSkillsInText(text string) []string {
	detected := make(map[string]bool)
	for canonical, pattern := range o.skillPatterns {
		if pattern.MatchString(text) {
			detected[canonical] = true
		}
	}

	result := make([]string, 0, len(detected))
	for skill := range detected {
		result = append(result, skill)
	}
	sort.Strings(result)
	return result
}

// =============================================================================
// JOB TITLE ONTOLOGY
// =============================================================================

// JobTitleOntology defines job title taxonomy for Tech/IT
type JobTitleOntology struct {
	// Canonical title -> Aliases
	titleAliases map[string][]string
	// Title -> Category
	titleCategories map[string]string
	// Seniority keywords
	seniorityKeywords map[string][]string
	// Title patterns
	titlePatterns map[string]*regexp.Regexp
}

// NewJobTitleOntology creates a new empty job title ontology
func NewJobTitleOntology() *JobTitleOntology {
	return &JobTitleOntology{
		titleAliases:      make(map[string][]string),
		titleCategories:   make(map[string]string),
		seniorityKeywords: make(map[string][]string),
		titlePatterns:     make(map[string]*regexp.Regexp),
	}
}

// NewJobTitleOntologyFromConfig creates a job title ontology from config
func NewJobTitleOntologyFromConfig(config *OntologyConfig) *JobTitleOntology {
	o := NewJobTitleOntology()
	o.loadFromConfig(config)
	return o
}

func (o *JobTitleOntology) loadFromConfig(config *OntologyConfig) {
	for category, titles := range config.JobTitles {
		for titleName, aliases := range titles {
			o.addTitle(titleName, category, aliases)
		}
	}
	o.seniorityKeywords = config.SeniorityKeywords
	o.compilePatterns()
}

func (o *JobTitleOntology) addTitle(canonical string, category string, aliases []string) {
	o.titleAliases[canonical] = aliases
	o.titleCategories[canonical] = category
}

func (o *JobTitleOntology) compilePatterns() {
	for canonical, aliases := range o.titleAliases {
		allNames := append([]string{canonical}, aliases...)
		escapedNames := make([]string, len(allNames))
		for i, name := range allNames {
			escapedNames[i] = regexp.QuoteMeta(name)
		}
		pattern := `(?i)(` + strings.Join(escapedNames, "|") + `)`
		o.titlePatterns[canonical] = regexp.MustCompile(pattern)
	}
}

// Normalize normalizes a job title
func (o *JobTitleOntology) Normalize(title string) (normalized string, category string, seniority string) {
	title = strings.TrimSpace(title)
	titleLower := strings.ToLower(title)

	// Extract seniority
	seniority = o.extractSeniority(titleLower)

	// Find canonical title
	normalized = title // Default to original
	for canonical, pattern := range o.titlePatterns {
		if pattern.MatchString(titleLower) {
			normalized = canonical
			category = o.titleCategories[canonical]
			break
		}
	}

	// Add seniority prefix if not already in title
	if seniority != "" && !strings.Contains(strings.ToLower(normalized), strings.ToLower(seniority)) {
		normalized = seniority + " " + normalized
	}

	return normalized, category, seniority
}

func (o *JobTitleOntology) extractSeniority(title string) string {
	for level, keywords := range o.seniorityKeywords {
		for _, keyword := range keywords {
			if strings.Contains(title, keyword) {
				return level
			}
		}
	}
	return ""
}

// =============================================================================
// LEVEL ONTOLOGY
// =============================================================================

// LevelOntology defines experience level taxonomy
type LevelOntology struct {
	levelAliases        map[string][]string
	levelYears          map[string]string
	levelResponsibility map[string]string
}

// NewLevelOntology creates a new empty level ontology
func NewLevelOntology() *LevelOntology {
	return &LevelOntology{
		levelAliases:        make(map[string][]string),
		levelYears:          make(map[string]string),
		levelResponsibility: make(map[string]string),
	}
}

// NewLevelOntologyFromConfig creates a level ontology from config
func NewLevelOntologyFromConfig(config *OntologyConfig) *LevelOntology {
	o := NewLevelOntology()
	o.loadFromConfig(config)
	return o
}

func (o *LevelOntology) loadFromConfig(config *OntologyConfig) {
	for levelName, levelEntry := range config.Levels {
		o.levelAliases[levelName] = levelEntry.Aliases
		o.levelYears[levelName] = levelEntry.YearsRange
		o.levelResponsibility[levelName] = levelEntry.Responsibility
	}
}

// Normalize normalizes a level string
func (o *LevelOntology) Normalize(level string) (normalized string, yearsRange string, responsibility string) {
	level = strings.TrimSpace(level)
	levelLower := strings.ToLower(level)

	for canonical, aliases := range o.levelAliases {
		for _, alias := range aliases {
			if strings.EqualFold(alias, level) || strings.Contains(levelLower, strings.ToLower(alias)) {
				return canonical, o.levelYears[canonical], o.levelResponsibility[canonical]
			}
		}
	}

	// Default mapping for common level values
	switch strings.ToUpper(level) {
	case "ENTRY":
		return "Entry Level", o.levelYears["Entry Level"], o.levelResponsibility["Entry Level"]
	case "JUNIOR":
		return "Junior", o.levelYears["Junior"], o.levelResponsibility["Junior"]
	case "MID":
		return "Mid Level", o.levelYears["Mid Level"], o.levelResponsibility["Mid Level"]
	case "SENIOR":
		return "Senior", o.levelYears["Senior"], o.levelResponsibility["Senior"]
	case "LEAD":
		return "Lead", o.levelYears["Lead"], o.levelResponsibility["Lead"]
	}

	return level, "", ""
}

// =============================================================================
// REQUIREMENT ONTOLOGY
// =============================================================================

// RequirementOntology defines requirement categories
type RequirementOntology struct {
	categoryPatterns    map[string]*regexp.Regexp
	requirementKeywords map[string][]string
}

// NewRequirementOntology creates a new empty requirement ontology
func NewRequirementOntology() *RequirementOntology {
	return &RequirementOntology{
		categoryPatterns:    make(map[string]*regexp.Regexp),
		requirementKeywords: make(map[string][]string),
	}
}

// NewRequirementOntologyFromConfig creates a requirement ontology from config
func NewRequirementOntologyFromConfig(config *OntologyConfig) *RequirementOntology {
	o := NewRequirementOntology()
	o.loadFromConfig(config)
	return o
}

func (o *RequirementOntology) loadFromConfig(config *OntologyConfig) {
	o.requirementKeywords = config.RequirementKeywords

	// Compile patterns
	for category, keywords := range o.requirementKeywords {
		escapedKeywords := make([]string, len(keywords))
		for i, kw := range keywords {
			escapedKeywords[i] = regexp.QuoteMeta(kw)
		}
		pattern := `(?i)(` + strings.Join(escapedKeywords, "|") + `)`
		o.categoryPatterns[category] = regexp.MustCompile(pattern)
	}
}

// Normalize normalizes requirements text and categorizes them
func (o *RequirementOntology) Normalize(requirements string) (normalized string, categorized map[string][]string) {
	categorized = make(map[string][]string)

	if requirements == "" {
		return "", categorized
	}

	// Split by common delimiters
	lines := splitRequirements(requirements)
	var normalizedLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Normalize the line
		normalizedLine := o.normalizeLine(line)
		normalizedLines = append(normalizedLines, normalizedLine)

		// Categorize
		for category, pattern := range o.categoryPatterns {
			if pattern.MatchString(line) {
				categorized[category] = append(categorized[category], normalizedLine)
				break
			}
		}
	}

	normalized = strings.Join(normalizedLines, "\n")
	return normalized, categorized
}

func (o *RequirementOntology) normalizeLine(line string) string {
	// Remove common prefixes
	prefixes := []string{"- ", "• ", "* ", "+ ", "· ", "– "}
	for _, prefix := range prefixes {
		line = strings.TrimPrefix(line, prefix)
	}

	// Remove numbering
	numbering := regexp.MustCompile(`^\d+[\.\)]\s*`)
	line = numbering.ReplaceAllString(line, "")

	// Normalize whitespace
	line = strings.Join(strings.Fields(line), " ")

	return line
}

func splitRequirements(text string) []string {
	// Split by newlines, bullets, or numbered lists
	splitter := regexp.MustCompile(`[\n\r]+|(?:\s*[-•*+·–]\s+)|(?:\s*\d+[\.\)]\s+)`)
	parts := splitter.Split(text, -1)

	var result []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// =============================================================================
// GLOBAL NORMALIZER INSTANCE
// =============================================================================

var (
	defaultNormalizer     *JDNormalizer
	defaultNormalizerOnce sync.Once
)

// GetDefaultNormalizer returns the default JD normalizer instance (singleton)
func GetDefaultNormalizer() *JDNormalizer {
	defaultNormalizerOnce.Do(func() {
		defaultNormalizer = NewJDNormalizer()
	})
	return defaultNormalizer
}

// ResetDefaultNormalizer resets the default normalizer (useful for testing)
func ResetDefaultNormalizer() {
	defaultNormalizerOnce = sync.Once{}
	defaultNormalizer = nil
}

// GetSkillOntology returns the skill ontology for testing/advanced usage
func (n *JDNormalizer) GetSkillOntology() *SkillOntology {
	return n.skillOntology
}

// DetectSkillsInText detects skills mentioned in a text
func (n *JDNormalizer) DetectSkillsInText(text string) []string {
	return n.skillOntology.DetectSkillsInText(text)
}
