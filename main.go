package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"

	"golang-role-crawler/analyzer"
	"golang-role-crawler/crawler"
	"golang-role-crawler/models"
)

type CrawlResponse struct {
	TotalJobs int           `json:"total_jobs"`
	Stats     []StatItem    `json:"stats"`
	Matches   []MatchResult `json:"matches"`
	Jobs      []models.Job  `json:"jobs"`
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

func handleCrawl(w http.ResponseWriter, r *http.Request) {
	fmt.Println("API request received: Crawling jobs...")
	
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

	// NEW: Filter out "noise" roles that have no technical requirements extracted.
	// This ensures we only show genuine tech roles with a stack.
	var techJobs []models.Job
	for _, j := range allJobs {
		if len(j.Requirements) > 0 {
			techJobs = append(techJobs, j)
		}
	}
	allJobs = techJobs

	if len(allJobs) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CrawlResponse{TotalJobs: 0})
		return
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

	resp := CrawlResponse{
		TotalJobs: len(allJobs),
		Stats:     stats,
		Matches:   matches,
		Jobs:      allJobs,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	http.HandleFunc("/api/crawl", handleCrawl)

	fmt.Println("Server is running on http://localhost:8080")
	fmt.Println("Press Ctrl+C to stop.")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
