package entities

import "gorm.io/datatypes"

type TrackingType string

const (
	TrackingJobSearch TrackingType = "tracking_job_search"
)

type UserTracking struct {
	ID           uint           `gorm:"primary_key,AUTO_INCREMENT" json:"id"`
	UserID       uint           `json:"user_id"`
	TrackingType TrackingType   `json:"tracking_type"`
	Metadata     datatypes.JSON `json:"metadata"`
}
