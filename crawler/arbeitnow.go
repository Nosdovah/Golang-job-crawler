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

type ArbeitnowResponse struct {
	Data []struct {
		Slug        string   `json:"slug"`
		Title       string   `json:"title"`
		CompanyName string   `json:"company_name"`
		Location    string   `json:"location"`
		JobTypes    []string `json:"job_types"`
		Remote      bool     `json:"remote"`
		URL         string   `json:"url"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
	} `json:"data"`
}

func FetchArbeitnowJobs() ([]models.Job, error) {
	// Arbeitnow API doesn't have a direct search param in the free tier,
	// so we fetch latest jobs and filter locally.
	var jobs []models.Job
	policy := bluemonday.StripTagsPolicy()

	for page := 1; page <= 30; page++ { // Increased to 30 pages for much deeper search
		url := fmt.Sprintf("https://www.arbeitnow.com/api/job-board-api?page=%d", page)
		
		resp, err := http.Get(url)
		if err != nil {
			return jobs, err
		}
		
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return jobs, err
		}

		var data ArbeitnowResponse
		if err := json.Unmarshal(body, &data); err != nil {
			return jobs, err
		}

		for _, j := range data.Data {
			cleanDesc := policy.Sanitize(j.Description)
			
			jobType := "Full-time"
			if len(j.JobTypes) > 0 {
				jobType = strings.Join(j.JobTypes, ", ")
			}

			location := j.Location
			if j.Remote {
				location += " (Remote)"
			}

			job := models.Job{
				ID:          fmt.Sprintf("arbeitnow-%s", j.Slug),
				Title:       j.Title,
				Company:     j.CompanyName,
				Location:    location,
				Type:        jobType,
				Description: cleanDesc,
				URL:         j.URL,
				Source:      "Arbeitnow",
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
			jobs = append(jobs, job)
		}
	}

	return jobs, nil
}
