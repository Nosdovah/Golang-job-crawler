package analyzer

import (
	"regexp"
	"strings"
	"golang-role-crawler/models"
)

// List of common technical keywords we want to track for Golang roles
var TechKeywords = []string{
	"golang", "grpc", "protobuf", "rest api", "restful", "api", "microservices",
	"docker", "kubernetes", "k8s", "aws", "gcp", "azure", "ci/cd",
	"postgresql", "mysql", "mongodb", "redis", "kafka", "rabbitmq",
	"terraform", "linux", "elasticsearch", "graphql", "sql", "nosql",
	"tdd", "bdd", "agile", "scrum", "python", "java", "c++", "rust",
	"gin", "echo", "fiber", "gorm", "sqlx", "chi", "swag", "promethus", "grafana",
}

// ExtractRequirements parses a text description and finds matching keywords
func ExtractRequirements(desc string) []string {
	descLower := strings.ToLower(desc)
	found := make(map[string]bool)
	var reqs []string

	// Special check for "Go" as a programming language to avoid the English word "go"
	// We look for "Go developer", "Go engineer", "Go programming", or "using Go"
	goPatterns := []string{
		`\bgo developer\b`, `\bgo engineer\b`, `\bgo backend\b`, 
		`\bprogramming in go\b`, `\bexperience with go\b`, `\bgo programming\b`,
	}
	for _, p := range goPatterns {
		if match, _ := regexp.MatchString(p, descLower); match {
			if !found["go"] {
				found["go"] = true
				reqs = append(reqs, "go")
			}
			break
		}
	}

	for _, kw := range TechKeywords {
		// Special handling for keywords like c++ which have regex meta characters
		safeKw := regexp.QuoteMeta(kw)
		
		var pattern string
		// If keyword contains non-word characters like ++ or /, word boundary \b might fail at the end.
		if regexp.MustCompile(`^\w+$`).MatchString(kw) {
			pattern = `\b` + safeKw + `\b`
		} else {
			pattern = `(?:^|\s)` + safeKw + `(?:$|\s|[.,;:!?)])`
		}

		match, _ := regexp.MatchString(pattern, descLower)
		if match && !found[kw] {
			found[kw] = true
			reqs = append(reqs, kw)
		}
	}
	return reqs
}

// JaccardSimilarity calculates the similarity between two slices of strings
// Returns a value between 0.0 (no overlap) and 1.0 (exact match)
func JaccardSimilarity(reqs1, reqs2 []string) float64 {
	if len(reqs1) == 0 && len(reqs2) == 0 {
		return 0.0
	}

	set1 := make(map[string]bool)
	for _, r := range reqs1 {
		set1[r] = true
	}

	intersection := 0
	unionSet := make(map[string]bool)

	for _, r := range reqs1 {
		unionSet[r] = true
	}

	for _, r := range reqs2 {
		if set1[r] {
			intersection++
		}
		unionSet[r] = true
	}

	return float64(intersection) / float64(len(unionSet))
}

// AnalyzeJobPool takes a list of jobs, extracts requirements for each,
// and returns overall requirement statistics.
func AnalyzeJobPool(jobs []models.Job) map[string]int {
	stats := make(map[string]int)
	for _, j := range jobs {
		for _, req := range j.Requirements {
			stats[req]++
		}
	}
	return stats
}
