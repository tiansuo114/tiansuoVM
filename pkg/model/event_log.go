package model

// EventLog represents a record of an event that occurred within the system.
type EventLog struct {
	ID           uint         `gorm:"primary_key" json:"id"`         // Primary key
	ResourceType ResourceType `gorm:"not null; index:resource_type"` // Resource type (VM or Disk)
	ResourceUID  string       `gorm:"not null; index:resource_uid"`
	EventType    EventType    `gorm:"not null" json:"event_type"`                       // Type of event (e.g., creation, deletion)
	Operation    string       `gorm:"not null" json:"operation"`                        // The operation that was performed
	CreatedAt    int64        `gorm:"autoCreateTime:milli; not null" json:"created_at"` // Event creation timestamp
	Creator      string       `gorm:"not null" json:"creator"`                          // The user who triggered the event
}

type ResourceType string

const (
	ResourceTypeVM    ResourceType = "VM"
	ResourceTypeImage ResourceType = "Image"
)

// EventType defines the type for event types.
type EventType string

// Constants for various event types
const (
	EventTypeCreation EventType = "Creation"
	EventTypeDeletion EventType = "Deletion"
	EventTypeStart    EventType = "Start"
	EventTypeStop     EventType = "Stop"
	EventTypeRestart  EventType = "Restart"
	EventTypeUpdate   EventType = "Update"
	EventTypeError    EventType = "Error"
)

// TableName sets the table name for the EventLog model
func (EventLog) TableName() string {
	return "event_logs"
}
