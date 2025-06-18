package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/helioschat/sync/internal/middleware"
	"github.com/helioschat/sync/internal/services"
	"github.com/helioschat/sync/internal/types"
)

type SyncHandler struct {
	syncService *services.SyncService
	authService *services.AuthService
}

func NewSyncHandler(syncService *services.SyncService, authService *services.AuthService) *SyncHandler {
	return &SyncHandler{
		syncService: syncService,
		authService: authService,
	}
}

// Thread handlers
func (h *SyncHandler) GetThreads(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusUnauthorized,
				Message: "User not authenticated",
			},
		})
		return
	}

	// Parse pagination parameters
	const maxLimit = 28 // Hard-coded maximum limit
	const defaultLimit = 10

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	limit := defaultLimit
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
			if limit > maxLimit {
				limit = maxLimit
			}
		}
	}

	// Parse optional since parameter
	var since *time.Time
	if sinceStr := c.Query("since"); sinceStr != "" {
		if sinceTime, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since = &sinceTime
		}
	}

	// Use paginated method
	result, err := h.syncService.GetThreadsPaginated(userID, offset, limit, since)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusInternalServerError,
				Message: "Failed to get threads",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    result,
	})
}

func (h *SyncHandler) UpsertThread(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusUnauthorized,
				Message: "User not authenticated",
			},
		})
		return
	}

	// Validate and parse thread ID from URL parameter
	threadID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusBadRequest,
				Message: "Invalid thread ID format - must be a valid UUID",
				Details: err.Error(),
			},
		})
		return
	}

	var req types.ThreadUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusBadRequest,
				Message: "Invalid request format",
				Details: err.Error(),
			},
		})
		return
	}

	// Validate that the user ID in the request matches the authenticated user
	if req.UserID != userID {
		c.JSON(http.StatusForbidden, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusForbidden,
				Message: "User ID in request does not match authenticated user",
			},
		})
		return
	}

	// Validate machine ID is a valid UUIDv7
	machineID, err := uuid.Parse(req.MachineID)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusBadRequest,
				Message: "Invalid machine ID format - must be a valid UUID",
				Details: err.Error(),
			},
		})
		return
	}

	if err := types.ValidateUUIDv7(machineID); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusBadRequest,
				Message: "Machine ID must be a valid UUIDv7",
				Details: err.Error(),
			},
		})
		return
	}

	thread := req.Data

	// Validate that the thread ID in the body matches the URL parameter
	if thread.ID != uuid.Nil && thread.ID != threadID {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusBadRequest,
				Message: "Thread ID in request body does not match URL parameter",
			},
		})
		return
	}

	// Set the IDs and version from the request
	thread.ID = threadID
	thread.UserID = req.UserID
	thread.Version = req.Version

	// Try to upsert the thread
	created, err := h.syncService.UpsertThread(&thread, req.MachineID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusInternalServerError,
				Message: "Failed to save thread",
				Details: err.Error(),
			},
		})
		return
	}

	statusCode := http.StatusOK
	if created {
		statusCode = http.StatusCreated
	}

	c.JSON(statusCode, types.APIResponse{
		Success: true,
		Data:    thread,
	})
}

func (h *SyncHandler) DeleteThread(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusUnauthorized,
				Message: "User not authenticated",
			},
		})
		return
	}

	threadID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusBadRequest,
				Message: "Invalid thread ID",
				Details: err.Error(),
			},
		})
		return
	}

	if err := h.syncService.DeleteThread(userID, threadID); err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusInternalServerError,
				Message: "Failed to delete thread",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    gin.H{"message": "Thread deleted successfully"},
	})
}

// Message handlers
func (h *SyncHandler) GetMessages(c *gin.Context) {
	// Parse required thread_id parameter
	threadIDStr := c.Query("thread_id")
	if threadIDStr == "" {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusBadRequest,
				Message: "thread_id parameter is required",
			},
		})
		return
	}

	// Parse pagination parameters
	const maxLimit = 50 // Hard-coded maximum limit for messages
	const defaultLimit = 20

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	limit := defaultLimit
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			if parsedLimit > maxLimit {
				limit = maxLimit
			} else {
				limit = parsedLimit
			}
		}
	}

	// Parse optional since parameter
	var since *time.Time
	if sinceStr := c.Query("since"); sinceStr != "" {
		if sinceTime, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since = &sinceTime
		}
	}

	// Use paginated method
	result, err := h.syncService.GetMessagesPaginated(threadIDStr, offset, limit, since)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusInternalServerError,
				Message: "Failed to get messages",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    result,
	})
}

func (h *SyncHandler) CreateMessage(c *gin.Context) {
	// Get threadID from URL parameter or request body
	threadIDStr := c.Query("thread_id")
	if threadIDStr == "" {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusBadRequest,
				Message: "thread_id parameter is required",
			},
		})
		return
	}

	var message types.Message
	if err := c.ShouldBindJSON(&message); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusBadRequest,
				Message: "Invalid request format",
				Details: err.Error(),
			},
		})
		return
	}

	// Since the Message struct no longer has UserID, we don't set it
	// The service will handle ID generation if needed

	if err := h.syncService.CreateMessage(threadIDStr, &message); err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusInternalServerError,
				Message: "Failed to create message",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, types.APIResponse{
		Success: true,
		Data:    message,
	})
}

func (h *SyncHandler) UpdateMessage(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusUnauthorized,
				Message: "User not authenticated",
			},
		})
		return
	}

	messageID := c.Param("id") // Now expecting string ID

	var req types.MessageUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusBadRequest,
				Message: "Invalid request format",
				Details: err.Error(),
			},
		})
		return
	}

	// Validate that the user ID in the request matches the authenticated user
	if req.UserID != userID {
		c.JSON(http.StatusForbidden, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusForbidden,
				Message: "User ID in request does not match authenticated user",
			},
		})
		return
	}

	// Validate machine ID is a valid UUIDv7
	machineID, err := uuid.Parse(req.MachineID)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusBadRequest,
				Message: "Invalid machine ID format - must be a valid UUID",
				Details: err.Error(),
			},
		})
		return
	}

	if err := types.ValidateUUIDv7(machineID); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusBadRequest,
				Message: "Machine ID must be a valid UUIDv7",
				Details: err.Error(),
			},
		})
		return
	}

	message := req.Data
	message.ID = messageID
	// Note: UserID and Version are no longer part of Message struct

	threadIDStr := req.ThreadID.String() // Convert UUID to string for service call

	if err := h.syncService.UpdateMessage(threadIDStr, &message, req.MachineID); err != nil {
		c.JSON(http.StatusConflict, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusConflict,
				Message: "Failed to update message",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    message,
	})
}

func (h *SyncHandler) DeleteMessage(c *gin.Context) {
	// Parse required thread_id parameter
	threadIDStr := c.Query("thread_id")
	if threadIDStr == "" {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusBadRequest,
				Message: "thread_id parameter is required",
			},
		})
		return
	}

	messageID := c.Param("id") // Now expecting string ID

	if err := h.syncService.DeleteMessage(threadIDStr, messageID); err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusInternalServerError,
				Message: "Failed to delete message",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    gin.H{"message": "Message deleted successfully"},
	})
}

// User settings handlers
func (h *SyncHandler) GetProviderInstances(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusUnauthorized,
				Message: "User not authenticated",
			},
		})
		return
	}

	providers, err := h.syncService.GetProviderInstances(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusNotFound,
				Message: "Provider instances not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    providers,
	})
}

func (h *SyncHandler) UpdateProviderInstances(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusUnauthorized,
				Message: "User not authenticated",
			},
		})
		return
	}

	var req types.ProviderInstancesUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusBadRequest,
				Message: "Invalid request format",
				Details: err.Error(),
			},
		})
		return
	}

	// Validate that the user ID in the request matches the authenticated user
	if req.UserID != userID {
		c.JSON(http.StatusForbidden, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusForbidden,
				Message: "User ID in request does not match authenticated user",
			},
		})
		return
	}

	// Validate machine ID is a valid UUIDv7
	machineID, err := uuid.Parse(req.MachineID)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusBadRequest,
				Message: "Invalid machine ID format - must be a valid UUID",
				Details: err.Error(),
			},
		})
		return
	}

	if err := types.ValidateUUIDv7(machineID); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusBadRequest,
				Message: "Machine ID must be a valid UUIDv7",
				Details: err.Error(),
			},
		})
		return
	}

	providers := req.Data
	providers.UserID = req.UserID
	providers.Version = req.Version

	if err := h.syncService.UpdateProviderInstances(&providers, req.MachineID); err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusInternalServerError,
				Message: "Failed to update provider instances",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    providers,
	})
}

func (h *SyncHandler) GetDisabledModels(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusUnauthorized,
				Message: "User not authenticated",
			},
		})
		return
	}

	models, err := h.syncService.GetDisabledModels(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusNotFound,
				Message: "Disabled models not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    models,
	})
}

func (h *SyncHandler) UpdateDisabledModels(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusUnauthorized,
				Message: "User not authenticated",
			},
		})
		return
	}

	var req types.DisabledModelsUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusBadRequest,
				Message: "Invalid request format",
				Details: err.Error(),
			},
		})
		return
	}

	// Validate that the user ID in the request matches the authenticated user
	if req.UserID != userID {
		c.JSON(http.StatusForbidden, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusForbidden,
				Message: "User ID in request does not match authenticated user",
			},
		})
		return
	}

	// Validate machine ID is a valid UUIDv7
	machineID, err := uuid.Parse(req.MachineID)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusBadRequest,
				Message: "Invalid machine ID format - must be a valid UUID",
				Details: err.Error(),
			},
		})
		return
	}

	if err := types.ValidateUUIDv7(machineID); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusBadRequest,
				Message: "Machine ID must be a valid UUIDv7",
				Details: err.Error(),
			},
		})
		return
	}

	models := req.Data
	models.UserID = req.UserID
	models.Version = req.Version

	if err := h.syncService.UpdateDisabledModels(&models, req.MachineID); err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusInternalServerError,
				Message: "Failed to update disabled models",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    models,
	})
}

func (h *SyncHandler) GetAdvancedSettings(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusUnauthorized,
				Message: "User not authenticated",
			},
		})
		return
	}

	settings, err := h.syncService.GetAdvancedSettings(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusNotFound,
				Message: "Advanced settings not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    settings,
	})
}

func (h *SyncHandler) UpdateAdvancedSettings(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusUnauthorized,
				Message: "User not authenticated",
			},
		})
		return
	}

	var req types.AdvancedSettingsUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusBadRequest,
				Message: "Invalid request format",
				Details: err.Error(),
			},
		})
		return
	}

	// Validate that the user ID in the request matches the authenticated user
	if req.UserID != userID {
		c.JSON(http.StatusForbidden, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusForbidden,
				Message: "User ID in request does not match authenticated user",
			},
		})
		return
	}

	// Validate machine ID is a valid UUIDv7
	machineID, err := uuid.Parse(req.MachineID)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusBadRequest,
				Message: "Invalid machine ID format - must be a valid UUID",
				Details: err.Error(),
			},
		})
		return
	}

	if err := types.ValidateUUIDv7(machineID); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusBadRequest,
				Message: "Machine ID must be a valid UUIDv7",
				Details: err.Error(),
			},
		})
		return
	}

	settings := req.Data
	settings.UserID = req.UserID
	settings.Version = req.Version

	if err := h.syncService.UpdateAdvancedSettings(&settings, req.MachineID); err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusInternalServerError,
				Message: "Failed to update advanced settings",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    settings,
	})
}

func (h *SyncHandler) GetChangesSince(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusUnauthorized,
				Message: "User not authenticated",
			},
		})
		return
	}

	timestampStr := c.Param("timestamp")
	timestampInt, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusBadRequest,
				Message: "Invalid timestamp format",
				Details: err.Error(),
			},
		})
		return
	}

	timestamp := time.UnixMilli(timestampInt)

	response, err := h.syncService.GetChangesSince(userID, timestamp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    http.StatusInternalServerError,
				Message: "Failed to get changes",
				Details: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    response,
	})
}
