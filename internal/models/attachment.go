package models

// Attachment represents a file attached to an email message
type Attachment struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	MessageID   uint   `gorm:"not null;index" json:"message_id"`
	Filename    string `gorm:"size:255" json:"filename"`
	ContentType string `gorm:"size:100" json:"content_type"`
	FilePath    string `gorm:"size:500" json:"file_path"`
	SizeBytes   int64  `json:"size_bytes"`

	// Relationships
	Message Message `gorm:"foreignKey:MessageID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName returns the table name for Attachment
func (Attachment) TableName() string {
	return "attachments"
}
