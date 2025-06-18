package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/helioschat/sync/internal/database"
	"github.com/helioschat/sync/internal/types"
)

type SyncService struct {
	db *database.RedisClient
}

func NewSyncService(db *database.RedisClient) *SyncService {
	return &SyncService{
		db: db,
	}
}

// Thread operations
func (s *SyncService) GetThreads(userID uuid.UUID, since *time.Time) ([]types.Thread, error) {
	pattern := fmt.Sprintf("threads:%s:*", userID.String())
	keys, err := s.db.Keys(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to get thread keys: %w", err)
	}

	var threads []types.Thread
	for _, key := range keys {
		data, err := s.db.Get(key)
		if err != nil {
			continue
		}

		var thread types.Thread
		if err := json.Unmarshal([]byte(data), &thread); err != nil {
			continue
		}

		// Filter by timestamp if provided
		// Since UpdatedAt is encrypted, use Version (milliseconds timestamp) for filtering
		if since != nil {
			threadTimestamp := time.UnixMilli(thread.Version)
			if !threadTimestamp.After(*since) {
				continue
			}
		}

		threads = append(threads, thread)
	}

	return threads, nil
}

// GetThreadsPaginated returns threads with pagination support
func (s *SyncService) GetThreadsPaginated(userID uuid.UUID, offset, limit int, since *time.Time) (*types.PaginatedThreadsResponse, error) {
	pattern := fmt.Sprintf("threads:%s:*", userID.String())
	keys, err := s.db.Keys(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to get thread keys: %w", err)
	}

	var allThreads []types.Thread
	for _, key := range keys {
		data, err := s.db.Get(key)
		if err != nil {
			continue
		}

		var thread types.Thread
		if err := json.Unmarshal([]byte(data), &thread); err != nil {
			continue
		}

		// Filter by timestamp if provided
		// Since UpdatedAt is encrypted, use Version (milliseconds timestamp) for filtering
		if since != nil {
			threadTimestamp := time.UnixMilli(thread.Version)
			if !threadTimestamp.After(*since) {
				continue
			}
		}

		allThreads = append(allThreads, thread)
	}

	total := len(allThreads)

	// Apply pagination
	var paginatedThreads []types.Thread
	if offset < total {
		end := offset + limit
		if end > total {
			end = total
		}
		paginatedThreads = allThreads[offset:end]
	}

	hasMore := offset+limit < total

	return &types.PaginatedThreadsResponse{
		Threads: paginatedThreads,
		Total:   total,
		Offset:  offset,
		Limit:   limit,
		HasMore: hasMore,
	}, nil
}

func (s *SyncService) UpsertThread(thread *types.Thread, machineID string) (bool, error) {
	// Check if thread already exists
	existing, err := s.getThread(thread.UserID, thread.ID)
	isCreating := err != nil // If we can't get the thread, we're creating a new one

	now := time.Now()

	if !isCreating {
		// Updating existing thread - check for version conflicts
		if thread.Version <= existing.Version {
			return false, fmt.Errorf("version conflict: server version %d, client version %d", existing.Version, thread.Version)
		}
	}

	if err := s.saveThread(thread); err != nil {
		return false, err
	}

	// Store the machine ID for this change
	if err := s.storeMachineIDForChange("thread", thread.ID, machineID, now); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to store machine ID for thread change: %v\n", err)
	}

	return isCreating, nil
}

func (s *SyncService) DeleteThread(userID, threadID uuid.UUID) error {
	key := fmt.Sprintf("threads:%s:%s", userID.String(), threadID.String())

	// Simply delete the key from Redis
	if err := s.db.Del(key); err != nil {
		return fmt.Errorf("failed to delete thread: %w", err)
	}

	// Remove from timestamp index
	timestampKey := fmt.Sprintf("timestamps:threads:%s", userID.String())
	if err := s.db.ZRem(timestampKey, threadID.String()); err != nil {
		return fmt.Errorf("failed to remove from timestamp index: %w", err)
	}

	return nil
}

func (s *SyncService) getThread(userID, threadID uuid.UUID) (*types.Thread, error) {
	key := fmt.Sprintf("threads:%s:%s", userID.String(), threadID.String())
	data, err := s.db.Get(key)
	if err != nil {
		return nil, err
	}

	var thread types.Thread
	if err := json.Unmarshal([]byte(data), &thread); err != nil {
		return nil, fmt.Errorf("failed to unmarshal thread: %w", err)
	}

	return &thread, nil
}

func (s *SyncService) saveThread(thread *types.Thread) error {
	key := fmt.Sprintf("threads:%s:%s", thread.UserID.String(), thread.ID.String())

	data, err := json.Marshal(thread)
	if err != nil {
		return fmt.Errorf("failed to marshal thread: %w", err)
	}

	if err := s.db.Set(key, string(data), 0); err != nil {
		return fmt.Errorf("failed to save thread: %w", err)
	}

	// Add to timestamp index for efficient querying
	// Since UpdatedAt is now encrypted, we'll use Version (which is a timestamp in milliseconds)
	timestampKey := fmt.Sprintf("timestamps:threads:%s", thread.UserID.String())
	score := float64(thread.Version)
	if err := s.db.ZAdd(timestampKey, score, thread.ID.String()); err != nil {
		return fmt.Errorf("failed to update timestamp index: %w", err)
	}

	return nil
}

// Message operations
func (s *SyncService) GetMessages(threadID string, since *time.Time) ([]types.Message, error) {
	pattern := fmt.Sprintf("messages:%s:*", threadID)
	keys, err := s.db.Keys(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to get message keys: %w", err)
	}

	var messages []types.Message
	for _, key := range keys {
		data, err := s.db.Get(key)
		if err != nil {
			continue
		}

		var message types.Message
		if err := json.Unmarshal([]byte(data), &message); err != nil {
			continue
		}

		// Since timestamps are now encrypted, we can't filter by time
		// Client will need to handle filtering if needed
		messages = append(messages, message)
	}

	return messages, nil
}

// GetMessagesPaginated returns messages with pagination support
func (s *SyncService) GetMessagesPaginated(threadID string, offset, limit int, since *time.Time) (*types.PaginatedMessagesResponse, error) {
	pattern := fmt.Sprintf("messages:%s:*", threadID)
	keys, err := s.db.Keys(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to get message keys: %w", err)
	}

	var allMessages []types.Message
	for _, key := range keys {
		data, err := s.db.Get(key)
		if err != nil {
			continue
		}

		var message types.Message
		if err := json.Unmarshal([]byte(data), &message); err != nil {
			continue
		}

		// Since timestamps are now encrypted, we can't filter by time
		// Client will need to handle filtering if needed
		allMessages = append(allMessages, message)
	}

	total := len(allMessages)

	// Apply pagination
	var paginatedMessages []types.Message
	if offset < total {
		end := offset + limit
		if end > total {
			end = total
		}
		paginatedMessages = allMessages[offset:end]
	}

	hasMore := offset+limit < total

	return &types.PaginatedMessagesResponse{
		Messages: paginatedMessages,
		Total:    total,
		Offset:   offset,
		Limit:    limit,
		HasMore:  hasMore,
	}, nil
}

func (s *SyncService) CreateMessage(threadID string, message *types.Message) error {
	if message.ID == "" {
		message.ID = uuid.New().String()
	}

	if err := s.saveMessage(threadID, message); err != nil {
		return err
	}

	// Store the change tracking for new message
	now := time.Now()
	if err := s.storeMessageChange("message", message.ID, "create", now, threadID); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to store message change tracking: %v\n", err)
	}

	return nil
}

func (s *SyncService) UpdateMessage(threadID string, message *types.Message, machineID string) error {
	// Since version is now encrypted, we can't do version checking here
	// Version checking would need to be done on the client side

	if err := s.saveMessage(threadID, message); err != nil {
		return err
	}

	// Store the machine ID for this change
	now := time.Now()
	if err := s.storeMachineIDForChange("message", uuid.MustParse(message.ID), machineID, now); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to store machine ID for message change: %v\n", err)
	}

	// Store the change tracking for updated message
	if err := s.storeMessageChange("message", message.ID, "update", now, threadID); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to store message change tracking: %v\n", err)
	}

	return nil
}

func (s *SyncService) DeleteMessage(threadID, messageID string) error {
	key := fmt.Sprintf("messages:%s:%s", threadID, messageID)

	// Store the change tracking for deleted message before actually deleting it
	now := time.Now()
	if err := s.storeMessageChange("message", messageID, "delete", now, threadID); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to store message change tracking: %v\n", err)
	}

	// Simply delete the key from Redis
	if err := s.db.Del(key); err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	return nil
}

func (s *SyncService) saveMessage(threadID string, message *types.Message) error {
	key := fmt.Sprintf("messages:%s:%s", threadID, message.ID)

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if err := s.db.Set(key, string(data), 0); err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	return nil
}

// User settings operations
func (s *SyncService) GetProviderInstances(userID uuid.UUID) (*types.ProviderInstances, error) {
	key := fmt.Sprintf("provider_instances:%s", userID.String())
	data, err := s.db.Get(key)
	if err != nil {
		return nil, err
	}

	var providers types.ProviderInstances
	if err := json.Unmarshal([]byte(data), &providers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal provider instances: %w", err)
	}

	return &providers, nil
}

func (s *SyncService) UpdateProviderInstances(providers *types.ProviderInstances, machineID string) error {
	now := time.Now()
	providers.UpdatedAt = now

	key := fmt.Sprintf("provider_instances:%s", providers.UserID.String())
	data, err := json.Marshal(providers)
	if err != nil {
		return fmt.Errorf("failed to marshal provider instances: %w", err)
	}

	if err := s.db.Set(key, string(data), 0); err != nil {
		return err
	}

	// Store the machine ID for this change
	if err := s.storeMachineIDForChange("provider_instances", providers.UserID, machineID, now); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to store machine ID for provider instances change: %v\n", err)
	}

	return nil
}

func (s *SyncService) GetDisabledModels(userID uuid.UUID) (*types.DisabledModels, error) {
	key := fmt.Sprintf("disabled_models:%s", userID.String())
	data, err := s.db.Get(key)
	if err != nil {
		return nil, err
	}

	var models types.DisabledModels
	if err := json.Unmarshal([]byte(data), &models); err != nil {
		return nil, fmt.Errorf("failed to unmarshal disabled models: %w", err)
	}

	return &models, nil
}

func (s *SyncService) UpdateDisabledModels(models *types.DisabledModels, machineID string) error {
	now := time.Now()
	models.UpdatedAt = now

	key := fmt.Sprintf("disabled_models:%s", models.UserID.String())
	data, err := json.Marshal(models)
	if err != nil {
		return fmt.Errorf("failed to marshal disabled models: %w", err)
	}

	if err := s.db.Set(key, string(data), 0); err != nil {
		return err
	}

	// Store the machine ID for this change
	if err := s.storeMachineIDForChange("disabled_models", models.UserID, machineID, now); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to store machine ID for disabled models change: %v\n", err)
	}

	return nil
}

func (s *SyncService) GetAdvancedSettings(userID uuid.UUID) (*types.AdvancedSettings, error) {
	key := fmt.Sprintf("advanced_settings:%s", userID.String())
	data, err := s.db.Get(key)
	if err != nil {
		return nil, err
	}

	var settings types.AdvancedSettings
	if err := json.Unmarshal([]byte(data), &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal advanced settings: %w", err)
	}

	return &settings, nil
}

func (s *SyncService) UpdateAdvancedSettings(settings *types.AdvancedSettings, machineID string) error {
	now := time.Now()
	settings.UpdatedAt = now

	key := fmt.Sprintf("advanced_settings:%s", settings.UserID.String())
	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal advanced settings: %w", err)
	}

	if err := s.db.Set(key, string(data), 0); err != nil {
		return err
	}

	// Store the machine ID for this change
	if err := s.storeMachineIDForChange("advanced_settings", settings.UserID, machineID, now); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to store machine ID for advanced settings change: %v\n", err)
	}

	return nil
}

// GetChangesSince retrieves changes since the given timestamp
func (s *SyncService) GetChangesSince(userID uuid.UUID, timestamp time.Time) (*types.ChangesSinceResponse, error) {
	now := time.Now()
	response := &types.ChangesSinceResponse{SyncTimestamp: now}

	// Initial full sync if timestamp is zero
	if timestamp.IsZero() {
		fullThreads, _ := s.GetThreads(userID, nil)
		// For messages, we need to get all messages across all threads
		// Since messages are now encrypted, we'll get them by thread pattern
		var fullMessages []types.Message
		pattern := "messages:*"
		keys, err := s.db.Keys(pattern)
		if err == nil {
			for _, key := range keys {
				data, err := s.db.Get(key)
				if err != nil {
					continue
				}
				var message types.Message
				if err := json.Unmarshal([]byte(data), &message); err != nil {
					continue
				}
				fullMessages = append(fullMessages, message)
			}
		}

		pi, _ := s.GetProviderInstances(userID)
		if pi != nil {
			response.ProviderInstances = pi
		}
		dm, _ := s.GetDisabledModels(userID)
		if dm != nil {
			response.DisabledModels = dm
		}
		as, _ := s.GetAdvancedSettings(userID)
		if as != nil {
			response.AdvancedSettings = as
		}
		response.FullThreads = fullThreads
		response.FullMessages = fullMessages
		return response, nil
	}

	// Incremental sync: build operations since timestamp
	var ops []types.ChangeOperation

	// Threads
	threads, _ := s.GetThreads(userID, &timestamp)
	for _, t := range threads {
		// Since UpdatedAt is encrypted, use Version (which is milliseconds timestamp) to create time.Time
		changeTimestamp := time.UnixMilli(t.Version)
		machineID, _ := s.getMachineIDForChange("thread", t.ID, changeTimestamp)
		ops = append(ops, types.ChangeOperation{
			Resource:  "thread",
			Operation: "update",
			ID:        t.ID.String(),
			MachineID: machineID,
			Data:      t,
			Timestamp: changeTimestamp,
		})
	}

	// For messages, since everything is encrypted, we can't easily filter by timestamp
	// We'll need to return all messages and let the client handle filtering
	// This is a limitation of having encrypted timestamps

	// Provider Instances
	if pi, err := s.GetProviderInstances(userID); err == nil && pi != nil && pi.UpdatedAt.After(timestamp) {
		machineID, _ := s.getMachineIDForChange("provider_instances", pi.UserID, pi.UpdatedAt)
		ops = append(ops, types.ChangeOperation{
			Resource:  "provider_instances",
			Operation: "update",
			ID:        pi.UserID.String(),
			MachineID: machineID,
			Data:      pi,
			Timestamp: pi.UpdatedAt,
		})
	}

	// Disabled Models
	if dm, err := s.GetDisabledModels(userID); err == nil && dm != nil && dm.UpdatedAt.After(timestamp) {
		machineID, _ := s.getMachineIDForChange("disabled_models", dm.UserID, dm.UpdatedAt)
		ops = append(ops, types.ChangeOperation{
			Resource:  "disabled_models",
			Operation: "update",
			ID:        dm.UserID.String(),
			MachineID: machineID,
			Data:      dm,
			Timestamp: dm.UpdatedAt,
		})
	}

	// Advanced Settings
	if as, err := s.GetAdvancedSettings(userID); err == nil && as != nil && as.UpdatedAt.After(timestamp) {
		machineID, _ := s.getMachineIDForChange("advanced_settings", as.UserID, as.UpdatedAt)
		ops = append(ops, types.ChangeOperation{
			Resource:  "advanced_settings",
			Operation: "update",
			ID:        as.UserID.String(),
			MachineID: machineID,
			Data:      as,
			Timestamp: as.UpdatedAt,
		})
	}

	// Message changes
	messageChanges, _ := s.getMessageChangesSince(timestamp)
	ops = append(ops, messageChanges...)

	response.Operations = ops
	return response, nil
}

// storeMachineIDForChange stores the machine ID that made a specific change
func (s *SyncService) storeMachineIDForChange(resourceType string, resourceID uuid.UUID, machineID string, timestamp time.Time) error {
	key := fmt.Sprintf("machine_id:%s:%s:%d", resourceType, resourceID.String(), timestamp.UnixMilli())
	return s.db.Set(key, machineID, 0) // Store permanently for now
}

// getMachineIDForChange retrieves the machine ID that made a specific change
func (s *SyncService) getMachineIDForChange(resourceType string, resourceID uuid.UUID, timestamp time.Time) (string, error) {
	key := fmt.Sprintf("machine_id:%s:%s:%d", resourceType, resourceID.String(), timestamp.UnixMilli())
	return s.db.Get(key)
}

// storeMessageChange stores a message change for tracking in the changes-since endpoint
func (s *SyncService) storeMessageChange(resourceType, messageID, operation string, timestamp time.Time, threadID string) error {
	key := fmt.Sprintf("message_changes:%s:%d", messageID, timestamp.UnixMilli())
	changeData := map[string]interface{}{
		"resource":   resourceType,
		"message_id": messageID,
		"thread_id":  threadID,
		"operation":  operation,
		"timestamp":  timestamp.UnixMilli(),
	}

	data, err := json.Marshal(changeData)
	if err != nil {
		return fmt.Errorf("failed to marshal message change: %w", err)
	}

	// Store with TTL of 30 days (2592000 seconds) to prevent infinite growth
	return s.db.Set(key, string(data), 2592000)
}

// getMessageChangesSince retrieves message changes since the given timestamp
func (s *SyncService) getMessageChangesSince(timestamp time.Time) ([]types.ChangeOperation, error) {
	pattern := "message_changes:*"
	keys, err := s.db.Keys(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to get message change keys: %w", err)
	}

	var ops []types.ChangeOperation
	for _, key := range keys {
		data, err := s.db.Get(key)
		if err != nil {
			continue
		}

		var changeData map[string]interface{}
		if err := json.Unmarshal([]byte(data), &changeData); err != nil {
			continue
		}

		// Extract timestamp and check if it's after the requested timestamp
		timestampMs, ok := changeData["timestamp"].(float64)
		if !ok {
			continue
		}

		changeTimestamp := time.UnixMilli(int64(timestampMs))
		if !changeTimestamp.After(timestamp) {
			continue
		}

		// Get the actual message data
		messageID, ok := changeData["message_id"].(string)
		if !ok {
			continue
		}

		threadID, ok := changeData["thread_id"].(string)
		if !ok {
			continue
		}

		operation, ok := changeData["operation"].(string)
		if !ok {
			continue
		}

		var messageData interface{}
		if operation != "delete" {
			// For non-delete operations, include the message data
			messageKey := fmt.Sprintf("messages:%s:%s", threadID, messageID)
			messageDataStr, err := s.db.Get(messageKey)
			if err == nil {
				var message types.Message
				if err := json.Unmarshal([]byte(messageDataStr), &message); err == nil {
					messageData = message
				}
			}
		}

		// Get machine ID if available
		machineID, _ := s.getMachineIDForChange("message", uuid.MustParse(messageID), changeTimestamp)

		ops = append(ops, types.ChangeOperation{
			Resource:  "message",
			Operation: operation,
			ID:        messageID,
			MachineID: machineID,
			Data:      messageData,
			Timestamp: changeTimestamp,
		})
	}

	return ops, nil
}
