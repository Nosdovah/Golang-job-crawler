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
	url := "https://remoteok.com/api"

	// Create request with User-Agent to avoid 403 Forbidden
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return jobs, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return jobs, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return jobs, err
	}

	var data []RemoteOKJob
	if err := json.Unmarshal(body, &data); err != nil {
		return jobs, err
	}

	policy := bluemonday.StripTagsPolicy()

	for _, j := range data {
		if j.Legal != "" { // Skip the API disclaimer object
			continue
		}

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
		
		// Ensure tags are also added
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

	return jobs, nil
}
