package dto

type SearchJobRequest struct {
	Keywords []string `form:"keywords" json:"keywords"`
	Page     int      `form:"page" json:"page"`
}
