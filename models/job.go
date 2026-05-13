package models

// Job represents a single job listing
type Job struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Company     string   `json:"company"`
	Location    string   `json:"location"`
	Type        string   `json:"type"` // e.g., "Full-time", "Contract"
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Source      string   `json:"source"`
	Requirements []string `json:"requirements"` // Extracted keywords
}
