package models

import "time"

type MitsubishiTagValue struct {
	TagID string `gorm:"primaryKey;type:uuid" json:"tag_id"`

	Tag MitsubishiTagList `gorm:"foreignKey:TagID;references:ID" json:"tag,omitempty"`

	TagName string `gorm:"type:varchar(255);index" json:"tag_name"`

	Value int `json:"value"`

	UpdatedAt time.Time `json:"updated_at"`
}

func (MitsubishiTagValue) TableName() string {
	return "mitsubishi_tag_values"
}