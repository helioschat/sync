package types

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Wallet represents a user's authentication wallet
type Wallet struct {
	UID              uuid.UUID `json:"uid"`
	Salt             string    `json:"salt"`              // Base64 encoded salt
	HashedPassphrase string    `json:"hashed_passphrase"` // Base64 encoded Argon2id hash
	CreatedAt        time.Time `json:"created_at"`
}

// AuthTokens represents JWT tokens
type AuthTokens struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// VersionedData represents data with versioning information
type VersionedData struct {
	ID        uuid.UUID   `json:"id"`
	UserID    uuid.UUID   `json:"user_id"`
	Data      interface{} `json:"data"`
	Version   int64       `json:"version"`
	UpdatedAt time.Time   `json:"updated_at"`
	CreatedAt time.Time   `json:"created_at"`
}

// Thread represents a chat thread with client-encrypted data
// ALL STRING AND INTERFACE{} FIELDS MARKED AS "CLIENT-ENCRYPTED" CONTAIN CLIENT-ENCRYPTED DATA
type Thread struct {
	ID                   uuid.UUID              `json:"id" validate:"required"`
	UserID               uuid.UUID              `json:"user_id" validate:"required"`
	Title                string                 `json:"title" validate:"required"` // CLIENT-ENCRYPTED STRING
	MessageCount         string                 `json:"messageCount"`              // CLIENT-ENCRYPTED STRING (originally int)
	LastMessageDate      string                 `json:"lastMessageDate,omitempty"` // CLIENT-ENCRYPTED STRING (originally *time.Time)
	Pinned               string                 `json:"pinned"`                    // CLIENT-ENCRYPTED STRING (originally bool)
	ProviderInstanceId   string                 `json:"providerInstanceId"`        // CLIENT-ENCRYPTED STRING
	Model                string                 `json:"model"`                     // CLIENT-ENCRYPTED STRING
	BranchedFrom         string                 `json:"branchedFrom,omitempty"`    // CLIENT-ENCRYPTED STRING (originally *uuid.UUID)
	WebSearchEnabled     string                 `json:"webSearchEnabled"`          // CLIENT-ENCRYPTED STRING (originally bool)
	WebSearchContextSize string                 `json:"webSearchContextSize"`      // CLIENT-ENCRYPTED STRING (originally int)
	Settings             map[string]interface{} `json:"settings"`                  // CLIENT-ENCRYPTED JSON VALUES
	Version              int64                  `json:"version"`
	UpdatedAt            string                 `json:"updated_at"` // CLIENT-ENCRYPTED STRING (originally time.Time)
	CreatedAt            string                 `json:"created_at"` // CLIENT-ENCRYPTED STRING (originally time.Time)
}

// Message represents a chat message with client-encrypted data
// ALL FIELDS EXCEPT ID ARE CLIENT-ENCRYPTED STRINGS
type Message struct {
	ID                   string `json:"id" validate:"required"`
	ThreadID             string `json:"threadId" validate:"required"`   // CLIENT-ENCRYPTED STRING (originally uuid.UUID)
	Role                 string `json:"role" validate:"required"`       // CLIENT-ENCRYPTED STRING
	Content              string `json:"content" validate:"required"`    // CLIENT-ENCRYPTED STRING
	AttachmentIds        string `json:"attachmentIds,omitempty"`        // CLIENT-ENCRYPTED STRING (originally []string)
	Reasoning            string `json:"reasoning,omitempty"`            // CLIENT-ENCRYPTED STRING
	ProviderInstanceId   string `json:"providerInstanceId,omitempty"`   // CLIENT-ENCRYPTED STRING
	Model                string `json:"model,omitempty"`                // CLIENT-ENCRYPTED STRING
	Usage                string `json:"usage,omitempty"`                // CLIENT-ENCRYPTED STRING (originally *TokenUsage)
	Metrics              string `json:"metrics,omitempty"`              // CLIENT-ENCRYPTED STRING (originally *StreamMetrics)
	CreatedAt            string `json:"created_at"`                     // CLIENT-ENCRYPTED STRING (originally time.Time)
	UpdatedAt            string `json:"updated_at"`                     // CLIENT-ENCRYPTED STRING (originally time.Time)
	Error                string `json:"error,omitempty"`                // CLIENT-ENCRYPTED STRING (originally *ChatError)
	WebSearchEnabled     string `json:"webSearchEnabled,omitempty"`     // CLIENT-ENCRYPTED STRING (originally *bool)
	WebSearchContextSize string `json:"webSearchContextSize,omitempty"` // CLIENT-ENCRYPTED STRING
}

// ProviderInstances represents user's AI provider configurations
type ProviderInstances struct {
	UserID    uuid.UUID              `json:"user_id" validate:"required"`
	Providers map[string]interface{} `json:"providers" validate:"required"` // CLIENT-ENCRYPTED JSON VALUES
	Version   int64                  `json:"version"`
	UpdatedAt time.Time              `json:"updated_at"`
	CreatedAt time.Time              `json:"created_at"`
}

// DisabledModels represents user's disabled AI models list
type DisabledModels struct {
	UserID    uuid.UUID         `json:"user_id" validate:"required"`
	Models    map[string]string `json:"models" validate:"required"` // CLIENT-ENCRYPTED record mapping provider instance ID to encrypted string
	Version   int64             `json:"version"`
	UpdatedAt time.Time         `json:"updated_at"`
	CreatedAt time.Time         `json:"created_at"`
}

// AdvancedSettings represents user's advanced application settings
type AdvancedSettings struct {
	UserID    uuid.UUID              `json:"user_id" validate:"required"`
	Settings  map[string]interface{} `json:"settings" validate:"required"` // CLIENT-ENCRYPTED JSON VALUES
	Version   int64                  `json:"version"`
	UpdatedAt time.Time              `json:"updated_at"`
	CreatedAt time.Time              `json:"created_at"`
}

// ChangeOperation represents a single change operation for sync
type ChangeOperation struct {
	Resource  string      `json:"resource"`       // e.g., "thread", "message", "provider_instances", etc.
	Operation string      `json:"operation"`      // "add", "update", "delete"
	ID        string      `json:"id"`             // ID of the resource (string to accommodate both UUIDs and message IDs)
	MachineID string      `json:"machine_id"`     // UUIDv7 of the client that made the change
	Data      interface{} `json:"data,omitempty"` // full object for add/update
	Timestamp time.Time   `json:"timestamp"`      // when the change occurred
}

// ChangesSinceResponse represents response data for the changes-since endpoint
// It includes full data on initial sync or operations for incremental updates
type ChangesSinceResponse struct {
	FullThreads       []Thread           `json:"threads,omitempty"`            // full thread list on initial sync
	FullMessages      []Message          `json:"messages,omitempty"`           // full message list on initial sync
	ProviderInstances *ProviderInstances `json:"provider_instances,omitempty"` // full settings on initial sync
	DisabledModels    *DisabledModels    `json:"disabled_models,omitempty"`    // full settings on initial sync
	AdvancedSettings  *AdvancedSettings  `json:"advanced_settings,omitempty"`  // full settings on initial sync
	Operations        []ChangeOperation  `json:"operations,omitempty"`         // incremental operations since last sync
	SyncTimestamp     time.Time          `json:"sync_timestamp"`               // server timestamp for this sync
}

// PaginationParams represents pagination parameters
type PaginationParams struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

// PaginatedThreadsResponse represents a paginated response for threads
type PaginatedThreadsResponse struct {
	Threads []Thread `json:"threads"`
	Total   int      `json:"total"`
	Offset  int      `json:"offset"`
	Limit   int      `json:"limit"`
	HasMore bool     `json:"has_more"`
}

// PaginatedMessagesResponse represents a paginated response for messages
type PaginatedMessagesResponse struct {
	Messages []Message `json:"messages"`
	Total    int       `json:"total"`
	Offset   int       `json:"offset"`
	Limit    int       `json:"limit"`
	HasMore  bool      `json:"has_more"`
}

// APIError represents a standardized API error response
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// APIResponse represents a standardized API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

// ValidateUUIDv7 validates that a UUID is version 7
func ValidateUUIDv7(u uuid.UUID) error {
	if u == uuid.Nil {
		return fmt.Errorf("UUID cannot be nil")
	}

	// Check if it's version 7 by examining the version bits
	// Version 7 has version bits set to 0111 (7) in the most significant 4 bits of the 7th byte
	version := (u[6] & 0xF0) >> 4
	if version != 7 {
		return fmt.Errorf("UUID must be version 7, got version %d", version)
	}

	return nil
}

// SyncRequest represents a generic sync request wrapper for PUT operations
type SyncRequest[T any] struct {
	MachineID string    `json:"machine_id" validate:"required"` // Unique ID for the machine making the request
	UserID    uuid.UUID `json:"user_id" validate:"required"`    // User ID for whom the sync is being performed
	Data      T         `json:"data" validate:"required"`       // The actual data payload
	Version   int64     `json:"version" validate:"required"`    // Version of the data being sent
}

// ThreadUpdateRequest represents a thread update request with machine ID
type ThreadUpdateRequest struct {
	MachineID string    `json:"machine_id" validate:"required"`
	UserID    uuid.UUID `json:"user_id" validate:"required"`
	Data      Thread    `json:"data" validate:"required"`
	Version   int64     `json:"version" validate:"required"`
}

// MessageUpdateRequest represents a message update request with machine ID
type MessageUpdateRequest struct {
	MachineID string    `json:"machine_id" validate:"required"`
	UserID    uuid.UUID `json:"user_id" validate:"required"`
	ThreadID  uuid.UUID `json:"thread_id" validate:"required"`
	Data      Message   `json:"data" validate:"required"`
	Version   int64     `json:"version" validate:"required"`
}

// ProviderInstancesUpdateRequest represents a provider instances update request with machine ID
type ProviderInstancesUpdateRequest struct {
	MachineID string            `json:"machine_id" validate:"required"`
	UserID    uuid.UUID         `json:"user_id" validate:"required"`
	Data      ProviderInstances `json:"data" validate:"required"`
	Version   int64             `json:"version" validate:"required"`
}

// DisabledModelsUpdateRequest represents a disabled models update request with machine ID
type DisabledModelsUpdateRequest struct {
	MachineID string         `json:"machine_id" validate:"required"`
	UserID    uuid.UUID      `json:"user_id" validate:"required"`
	Data      DisabledModels `json:"data" validate:"required"`
	Version   int64          `json:"version" validate:"required"`
}

// AdvancedSettingsUpdateRequest represents an advanced settings update request with machine ID
type AdvancedSettingsUpdateRequest struct {
	MachineID string           `json:"machine_id" validate:"required"`
	UserID    uuid.UUID        `json:"user_id" validate:"required"`
	Data      AdvancedSettings `json:"data" validate:"required"`
	Version   int64            `json:"version" validate:"required"`
}

// Helper function to marshal Wallet to JSON
func WalletToJSON(wallet *Wallet) ([]byte, error) {
	return json.Marshal(wallet)
}

// Helper function to unmarshal JSON to Wallet
func WalletFromJSON(data []byte, wallet *Wallet) error {
	return json.Unmarshal(data, wallet)
}
