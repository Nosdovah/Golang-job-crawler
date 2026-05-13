package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"

	"golang-role-crawler/analyzer"
	"golang-role-crawler/crawler"
	"golang-role-crawler/models"
)

func isMostlyEnglish(title, desc string) bool {
	combinedLower := strings.ToLower(title + " " + desc)
	
	// Immediate rejection for standard German job title gender markers e.g. (m/w/d), (f/m/x), (w/m/d)
	if match, _ := regexp.MatchString(`\([a-z]/[a-z]/[a-z]\)`, combinedLower); match {
		return false
	}

	// Common German stop words
	germanStopWords := []string{" und ", " der ", " die ", " das ", " mit ", " für ", " auf ", " sind ", " wir ", " eine ", " oder ", " werden ", " zu ", " gesucht "}
	germanCount := 0
	for _, w := range germanStopWords {
		if strings.Contains(combinedLower, w) {
			germanCount++
		}
	}
	return germanCount < 2
}

func isTechRole(title string) bool {
	titleLower := strings.ToLower(title)
	
	// Must contain at least one strictly engineering/tech keyword in the title
	whitelist := []string{
		"engineer", "developer", "programmer", "architect", "backend", "frontend", "fullstack", 
		"full-stack", "data scientist", "sre", "devops", "systems", "platform", "software",
		"technical lead", "tech lead", "cto", "data engineer", "machine learning",
	}
	
	for _, w := range whitelist {
		if strings.Contains(titleLower, w) {
			return true
		}
	}
	
	return false
}

type CrawlResponse struct {
	TotalJobs int           `json:"total_jobs"`
	Stats     []StatItem    `json:"stats"`
	Matches   []MatchResult `json:"matches"`
	Jobs      []models.Job  `json:"jobs"`
	Summary   string        `json:"summary"`
}

type StatItem struct {
	Requirement string `json:"requirement"`
	Count       int    `json:"count"`
	Percentage  int    `json:"percentage"`
}

type MatchResult struct {
	Job1        models.Job `json:"job1"`
	Job2        models.Job `json:"job2"`
	Score       int        `json:"score"`
	SharedStack []string   `json:"shared_stack"`
}

func runCrawlLogic() CrawlResponse {
	fmt.Println("Crawling jobs...")
	// Force disable AI analysis because local Ollama models often fail to extract JSON 
	// properly and return 0 requirements for thousands of jobs.
	os.Setenv("USE_AI_ANALYSIS", "false")

	
	var allJobs []models.Job

	// Fetch from sources
	remotiveJobs, _ := crawler.FetchRemotiveJobs()
	allJobs = append(allJobs, remotiveJobs...)

	arbeitnowJobs, _ := crawler.FetchArbeitnowJobs()
	allJobs = append(allJobs, arbeitnowJobs...)

	// 3. Fetch from Jobicy
	fmt.Println("Fetching jobs from Jobicy (Aggregator API)...")
	jobicyJobs, _ := crawler.FetchJobicyJobs()
	allJobs = append(allJobs, jobicyJobs...)

	// 4. Fetch from RemoteOK
	fmt.Println("Fetching jobs from RemoteOK (Aggregator API)...")
	remoteokJobs, _ := crawler.FetchRemoteOKJobs()
	allJobs = append(allJobs, remoteokJobs...)

	// Filter out non-tech roles and non-English roles
	// To ensure we only get high-quality backend/tech roles, we require the job to have 
	// at least one CORE backend tech stack element, not just generic terms like "agile" or "sql".
	var techJobs []models.Job
	for _, j := range allJobs {
		hasCoreTech := false
		for _, req := range j.Requirements {
			if req == "golang" || req == "go" || req == "rust" || req == "python" || req == "java" || req == "c++" || req == "node" || req == "kubernetes" || req == "docker" || req == "aws" || req == "gcp" || req == "ruby" || req == "php" || req == "c#" || req == ".net" || req == "swift" || req == "kotlin" || req == "typescript" || req == "react" || req == "vue" || req == "angular" {
				hasCoreTech = true
				break
			}
		}

		if hasCoreTech && len(j.Requirements) > 0 && isMostlyEnglish(j.Title, j.Description) && isTechRole(j.Title) {
			techJobs = append(techJobs, j)
		}
	}
	allJobs = techJobs

	if len(allJobs) == 0 {
		return CrawlResponse{TotalJobs: 0}
	}

	// 1. Requirement Stats
	statsMap := analyzer.AnalyzeJobPool(allJobs)
	var stats []StatItem
	for k, v := range statsMap {
		stats = append(stats, StatItem{
			Requirement: strings.ToUpper(k),
			Count:       v,
			Percentage:  int((float64(v) / float64(len(allJobs))) * 100),
		})
	}
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Count > stats[j].Count
	})
	if len(stats) > 10 {
		stats = stats[:10]
	}

	// 2. Similarity Matches
	var matches []MatchResult
	for i := 0; i < len(allJobs); i++ {
		for j := i + 1; j < len(allJobs); j++ {
			score := analyzer.JaccardSimilarity(allJobs[i].Requirements, allJobs[j].Requirements)
			if score >= 0.5 { // Only jobs with 50%+ overlap
				common := []string{}
				reqSet := make(map[string]bool)
				for _, req := range allJobs[i].Requirements {
					reqSet[req] = true
				}
				for _, req := range allJobs[j].Requirements {
					if reqSet[req] {
						common = append(common, req)
					}
				}
				matches = append(matches, MatchResult{
					Job1:        allJobs[i],
					Job2:        allJobs[j],
					Score:       int(score * 100),
					SharedStack: common,
				})
			}
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})
	if len(matches) > 10 {
		matches = matches[:10] // top 10 matches
	}

	// Generate Overall Market Summary
	var summary string
	if len(stats) >= 3 {
		summary = fmt.Sprintf("Based on the analysis of %d high-quality roles, there is a strong similarity in Cloud-Native infrastructure. The most dominant requirements across these roles are %s (%d%%), %s (%d%%), and %s (%d%%).", 
			len(allJobs),
			stats[0].Requirement, stats[0].Percentage,
			stats[1].Requirement, stats[1].Percentage,
			stats[2].Requirement, stats[2].Percentage)
	} else {
		summary = "Not enough data to generate a market summary."
	}

	return CrawlResponse{
		TotalJobs: len(allJobs),
		Stats:     stats,
		Matches:   matches,
		Jobs:      allJobs,
		Summary:   summary,
	}
}

func handleCrawl(w http.ResponseWriter, r *http.Request) {
	resp := runCrawlLogic()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	buildFlag := flag.Bool("build", false, "Build static JSON data file for GitHub Pages")
	flag.Parse()

	if *buildFlag {
		fmt.Println("Building static data for GitHub Pages...")
		resp := runCrawlLogic()
		file, err := os.Create("static/data.json")
		if err != nil {
			log.Fatalf("Failed to create static/data.json: %v", err)
		}
		defer file.Close()
		json.NewEncoder(file).Encode(resp)
		fmt.Println("✅ Successfully built static/data.json")
		return
	}

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	http.HandleFunc("/api/crawl", handleCrawl)

	fmt.Println("Server is running on http://localhost:8080")
	fmt.Println("Press Ctrl+C to stop.")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
