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

type RemotiveResponse struct {
	Jobs []struct {
		ID             int    `json:"id"`
		URL            string `json:"url"`
		Title          string `json:"title"`
		CompanyName    string `json:"company_name"`
		JobType        string `json:"job_type"`
		CandidateRequiredLocation string `json:"candidate_required_location"`
		Description    string `json:"description"`
	} `json:"jobs"`
}

func FetchRemotiveJobs() ([]models.Job, error) {
	// Fetch massive generic dataset, local engine will perfectly filter Go jobs
	url := "https://remotive.com/api/remote-jobs?category=software-dev&limit=1000"
	
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data RemotiveResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	var jobs []models.Job
	policy := bluemonday.StripTagsPolicy() // strips HTML tags securely

	for _, j := range data.Jobs {
		cleanDesc := policy.Sanitize(j.Description)
		
		job := models.Job{
			ID:          fmt.Sprintf("remotive-%d", j.ID),
			Title:       j.Title,
			Company:     j.CompanyName,
			Location:    j.CandidateRequiredLocation,
			Type:        strings.ReplaceAll(j.JobType, "_", " "),
			Description: cleanDesc,
			URL:         j.URL,
			Source:      "Remotive",
		}
		
		// Run analysis to extract requirements
		if os.Getenv("USE_AI_ANALYSIS") == "true" {
			aiReqs, err := analyzer.ExtractRequirementsWithAI(cleanDesc)
			if err == nil {
				job.Requirements = aiReqs
			} else {
				// Fallback to local if AI fails
				job.Requirements = analyzer.ExtractRequirements(cleanDesc)
			}
		} else {
			job.Requirements = analyzer.ExtractRequirements(cleanDesc)
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}
