package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"jiramo/internal/models"
	"jiramo/internal/utils"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type InstanceHandler struct {
	DB       *gorm.DB
	Validate *validator.Validate
}

func NewInstanceHandler(db *gorm.DB) *InstanceHandler {
	return &InstanceHandler{DB: db, Validate: validator.New()}
}

type InstanceInput struct {
	Name        string            `json:"name" validate:"required,min=3,max=64"`
	Server      string            `json:"server" validate:"required,min=3,max=255"`
	Domain      string            `json:"domain" validate:"omitempty,min=3,max=255"`
	BaseRepoURL string            `json:"base_repo_url" validate:"required,min=3,max=255"`
	BaseRepoRef string            `json:"base_repo_ref" validate:"omitempty,min=1,max=128"`
	ComposePath string            `json:"compose_path" validate:"omitempty,min=1,max=255"`
	Environment map[string]string `json:"environment"`
	Status      string            `json:"status" validate:"omitempty,oneof=draft ready paused error"`
}

type UpdateInstanceInput struct {
	Name        *string           `json:"name" validate:"omitempty,min=3,max=64"`
	Server      *string           `json:"server" validate:"omitempty,min=3,max=255"`
	Domain      *string           `json:"domain" validate:"omitempty,min=3,max=255"`
	BaseRepoURL *string           `json:"base_repo_url" validate:"omitempty,min=3,max=255"`
	BaseRepoRef *string           `json:"base_repo_ref" validate:"omitempty,min=1,max=128"`
	ComposePath *string           `json:"compose_path" validate:"omitempty,min=1,max=255"`
	Environment map[string]string `json:"environment"`
	Status      *string           `json:"status" validate:"omitempty,oneof=draft ready paused error"`
}

type ApplyInstanceResponse struct {
	Command           string            `json:"command"`
	EnvironmentFile   string            `json:"environment_file"`
	Environment       map[string]string `json:"environment"`
	ComposePath       string            `json:"compose_path"`
	BaseRepoURL       string            `json:"base_repo_url"`
	BaseRepoRef       string            `json:"base_repo_ref"`
	LastAppliedAt     *time.Time        `json:"last_applied_at"`
	Status            string            `json:"status"`
}

func (h *InstanceHandler) ListInstances(w http.ResponseWriter, r *http.Request) {
	var instances []models.Instance
	if err := h.DB.Order("created_at desc").Find(&instances).Error; err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to retrieve instances")
		return
	}

	utils.WriteJSON(w, http.StatusOK, instances)
}

func (h *InstanceHandler) CreateInstance(w http.ResponseWriter, r *http.Request) {
	var input InstanceInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if err := h.Validate.Struct(input); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	if input.BaseRepoRef == "" {
		input.BaseRepoRef = "main"
	}
	if input.ComposePath == "" {
		input.ComposePath = "."
	}
	if input.Environment == nil {
		input.Environment = map[string]string{}
	}
	if input.Status == "" {
		input.Status = "draft"
	}

	instance := models.Instance{
		Name:           input.Name,
		Server:         input.Server,
		Domain:         input.Domain,
		BaseRepoURL:    input.BaseRepoURL,
		BaseRepoRef:    input.BaseRepoRef,
		ComposePath:     input.ComposePath,
		Environment:    input.Environment,
		Status:         input.Status,
	}

	if err := h.DB.Create(&instance).Error; err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Error during creation")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, instance)
}

func (h *InstanceHandler) GetInstance(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid id")
		return
	}

	var instance models.Instance
	if err := h.DB.First(&instance, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.WriteError(w, http.StatusNotFound, "Instance not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "Failed to retrieve instance")
		return
	}

	utils.WriteJSON(w, http.StatusOK, instance)
}

func (h *InstanceHandler) UpdateInstance(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid id")
		return
	}

	var input UpdateInstanceInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if err := h.Validate.Struct(input); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	updates := map[string]any{}
	if input.Name != nil {
		updates["name"] = *input.Name
	}
	if input.Server != nil {
		updates["server"] = *input.Server
	}
	if input.Domain != nil {
		updates["domain"] = *input.Domain
	}
	if input.BaseRepoURL != nil {
		updates["base_repo_url"] = *input.BaseRepoURL
	}
	if input.BaseRepoRef != nil {
		updates["base_repo_ref"] = *input.BaseRepoRef
	}
	if input.ComposePath != nil {
		updates["compose_path"] = *input.ComposePath
	}
	if input.Environment != nil {
		raw, err := json.Marshal(input.Environment)
		if err != nil {
			utils.WriteError(w, http.StatusBadRequest, "Invalid environment payload")
			return
		}
		updates["environment_raw"] = string(raw)
	}
	if input.Status != nil {
		updates["status"] = *input.Status
	}

	if len(updates) == 0 {
		utils.WriteJSON(w, http.StatusOK, map[string]string{"message": "no changes applied"})
		return
	}

	if err := h.DB.Model(&models.Instance{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Error during update")
		return
	}

	var instance models.Instance
	if err := h.DB.First(&instance, "id = ?", id).Error; err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to retrieve updated instance")
		return
	}

	utils.WriteJSON(w, http.StatusOK, instance)
}

func (h *InstanceHandler) DeleteInstance(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid id")
		return
	}

	if err := h.DB.Delete(&models.Instance{}, "id = ?", id).Error; err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Deletion failed")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{"message": "Deleted"})
}

func (h *InstanceHandler) DispatchInstance(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid id")
		return
	}

	var instance models.Instance
	if err := h.DB.First(&instance, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.WriteError(w, http.StatusNotFound, "Instance not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "Failed to retrieve instance")
		return
	}

	agentURL := instance.Server
	if agentURL == "" {
		utils.WriteError(w, http.StatusBadRequest, "instance server (agent URL) is not set")
		return
	}

	token := instance.Environment["AGENT_TOKEN"]
	if token == "" {
		utils.WriteError(w, http.StatusBadRequest, "agent token not configured for instance")
		return
	}

	payload := map[string]any{
		"base_repo_url": instance.BaseRepoURL,
		"base_repo_ref": instance.BaseRepoRef,
		"compose_path":  instance.ComposePath,
		"environment":   instance.Environment,
	}
	body, _ := json.Marshal(payload)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", agentURL+"/apply", bytes.NewReader(body))
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "failed to build request to agent")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		utils.WriteError(w, http.StatusBadGateway, "failed to contact agent: "+err.Error())
		return
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)

	// forward agent response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(data)
}

func (h *InstanceHandler) ApplyInstance(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid id")
		return
	}

	var instance models.Instance
	if err := h.DB.First(&instance, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.WriteError(w, http.StatusNotFound, "Instance not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "Failed to retrieve instance")
		return
	}

	now := time.Now().UTC()
	updates := map[string]any{
		"last_applied_at": now,
		"status":          "ready",
	}

	if err := h.DB.Model(&models.Instance{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to apply instance configuration")
		return
	}

	instance.LastAppliedAt = &now
	instance.Status = "ready"

	utils.WriteJSON(w, http.StatusOK, ApplyInstanceResponse{
		Command:         "docker compose up --build -d",
		EnvironmentFile:  instance.RenderEnvironmentFile(),
		Environment:      instance.Environment,
		ComposePath:      instance.ComposePath,
		BaseRepoURL:      instance.BaseRepoURL,
		BaseRepoRef:      instance.BaseRepoRef,
		LastAppliedAt:    instance.LastAppliedAt,
		Status:           instance.Status,
	})
}