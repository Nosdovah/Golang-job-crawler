package crawler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"golang-role-crawler/analyzer"
	"golang-role-crawler/models"

	"github.com/microcosm-cc/bluemonday"
)

type RemoteOKJob struct {
	ID          string   `json:"id"`
	URL         string   `json:"url"`
	Title       string   `json:"position"`
	Company     string   `json:"company"`
	Location    string   `json:"location"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Legal       string   `json:"legal"` // To skip the first dummy object
}

func FetchRemoteOKJobs() ([]models.Job, error) {
	var jobs []models.Job
	
	urls := []string{
		"https://remoteok.com/api",
		"https://remoteok.com/api?tags=engineer",
		"https://remoteok.com/api?tags=developer",
		"https://remoteok.com/api?tags=backend",
	}

	client := &http.Client{}
	policy := bluemonday.StripTagsPolicy()
	seenIDs := make(map[string]bool)

	for _, url := range urls {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

		resp, err := client.Do(req)
		if err != nil {
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		var data []RemoteOKJob
		if err := json.Unmarshal(body, &data); err != nil {
			continue
		}

		for _, j := range data {
			if j.Legal != "" {
				continue
			}
			
			if seenIDs[j.ID] {
				continue
			}
			seenIDs[j.ID] = true

			cleanDesc := policy.Sanitize(j.Description)

			job := models.Job{
				ID:          fmt.Sprintf("remoteok-%s", j.ID),
				Title:       j.Title,
				Company:     j.Company,
				Location:    j.Location,
				Type:        "Remote",
				Description: cleanDesc,
				URL:         j.URL,
				Source:      "RemoteOK",
			}

			if os.Getenv("USE_AI_ANALYSIS") == "true" {
				aiReqs, err := analyzer.ExtractRequirementsWithAI(cleanDesc)
				if err == nil {
					job.Requirements = aiReqs
				} else {
					job.Requirements = analyzer.ExtractRequirements(cleanDesc)
				}
			} else {
				job.Requirements = analyzer.ExtractRequirements(cleanDesc)
			}
			
			for _, tag := range j.Tags {
				if strings.TrimSpace(tag) != "" {
					tagLower := strings.ToLower(tag)
					found := false
					for _, r := range job.Requirements {
						if r == tagLower {
							found = true
							break
						}
					}
					if !found {
						job.Requirements = append(job.Requirements, tagLower)
					}
				}
			}

			jobs = append(jobs, job)
		}
	}

	return jobs, nil
}
