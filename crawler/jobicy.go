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

type JobicyResponse struct {
	Jobs []struct {
		ID             int    `json:"id"`
		URL            string `json:"url"`
		JobTitle       string `json:"jobTitle"`
		CompanyName    string `json:"companyName"`
		JobType        []string `json:"jobType"`
		JobGeo         string `json:"jobGeo"`
		JobDescription string `json:"jobDescription"`
	} `json:"jobs"`
}

func FetchJobicyJobs() ([]models.Job, error) {
	var jobs []models.Job
	policy := bluemonday.StripTagsPolicy()

	for page := 1; page <= 5; page++ {
		url := fmt.Sprintf("https://jobicy.com/api/v2/remote-jobs?count=50&industry=dev&page=%d", page)
		
		resp, err := http.Get(url)
		if err != nil {
			return jobs, err
		}
		
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return jobs, err
		}

		var data JobicyResponse
		if err := json.Unmarshal(body, &data); err != nil {
			continue // Skip errors and go to next page
		}

		for _, j := range data.Jobs {
			cleanDesc := policy.Sanitize(j.JobDescription)
			
			job := models.Job{
				ID:          fmt.Sprintf("jobicy-%d", j.ID),
				Title:       j.JobTitle,
				Company:     j.CompanyName,
				Location:    j.JobGeo,
				Type:        strings.Join(j.JobType, ", "),
				Description: cleanDesc,
				URL:         j.URL,
				Source:      "Jobicy",
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
