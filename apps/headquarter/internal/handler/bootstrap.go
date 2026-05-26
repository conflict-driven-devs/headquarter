package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/conflict-driven-devs/headquarter/internal/models"
	"github.com/conflict-driven-devs/headquarter/internal/utils"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	errBootstrapTokenUsed    = errors.New("bootstrap token used")
	errBootstrapTokenExpired = errors.New("bootstrap token expired")
)

type BootstrapHandler struct {
	DB *gorm.DB
}

func NewBootstrapHandler(db *gorm.DB) *BootstrapHandler {
	return &BootstrapHandler{DB: db}
}

type bootstrapRequest struct {
	Hostname   string `json:"hostname"`
	ServerAddr string `json:"server_addr"`
}

type bootstrapResponse struct {
	AgentToken     string            `json:"agent_token"`
	AgentBinaryURL string            `json:"agent_binary_url"`
	InstanceName   string            `json:"instance_name"`
	BaseRepoURL    string            `json:"base_repo_url"`
	BaseRepoRef    string            `json:"base_repo_ref"`
	ComposePath    string            `json:"compose_path"`
	Environment    map[string]string `json:"environment"`
}

func (h *BootstrapHandler) Register(w http.ResponseWriter, r *http.Request) {
	// Accept either a global HQ_BOOTSTRAP_KEY (legacy/admin flow) or a one-time bootstrap token
	auth := r.Header.Get("Authorization")
	if auth == "" {
		utils.WriteError(w, http.StatusUnauthorized, "missing authorization header")
		return
	}

	// If matches HQ_BOOTSTRAP_KEY -> create new instance (legacy admin flow)
	key := os.Getenv("HQ_BOOTSTRAP_KEY")
	if key != "" && auth == "Bearer "+key {
		var req bootstrapRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			utils.WriteError(w, http.StatusBadRequest, "invalid body")
			return
		}

		// generate agent token
		tok := make([]byte, 32)
		if _, err := rand.Read(tok); err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "token generation failed")
			return
		}
		agentToken := hex.EncodeToString(tok)

		instance := models.Instance{
			Name:        req.Hostname,
			Server:      req.ServerAddr,
			BaseRepoURL: "",
			BaseRepoRef: "main",
			ComposePath: ".",
			Environment: map[string]string{"AGENT_TOKEN": agentToken},
			Status:      "draft",
		}

		if err := h.DB.Create(&instance).Error; err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "failed to create instance")
			return
		}

		resp := bootstrapResponse{
			AgentToken:     agentToken,
			AgentBinaryURL: os.Getenv("HQ_AGENT_BINARY_URL"),
		}

		utils.WriteJSON(w, http.StatusCreated, resp)
		return
	}

	// Otherwise expect a one-time bootstrap token: validate and consume atomically.
	token := ""
	if len(auth) > 7 && auth[:7] == "Bearer " {
		token = auth[7:]
	}
	if token == "" {
		utils.WriteError(w, http.StatusUnauthorized, "invalid bootstrap token")
		return
	}

	sum := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(sum[:])

	var response bootstrapResponse
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		var bt models.BootstrapToken
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&bt, "token_hash = ?", tokenHash).Error; err != nil {
			return err
		}
		if bt.Used {
			return errBootstrapTokenUsed
		}
		if bt.ExpiresAt != nil && bt.ExpiresAt.Before(time.Now()) {
			return errBootstrapTokenExpired
		}

		var instance models.Instance
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&instance, "id = ?", bt.InstanceID).Error; err != nil {
			return err
		}

		tok := make([]byte, 32)
		if _, err := rand.Read(tok); err != nil {
			return err
		}
		agentToken := hex.EncodeToString(tok)
		if instance.Environment == nil {
			instance.Environment = map[string]string{}
		}
		instance.Environment["AGENT_TOKEN"] = agentToken

		if err := tx.Save(&instance).Error; err != nil {
			return err
		}

		bt.Used = true
		if err := tx.Save(&bt).Error; err != nil {
			return err
		}

		// sanitize env for the setup script; never send the agent token back inside environment
		envCopy := make(map[string]string, len(instance.Environment))
		for k, v := range instance.Environment {
			if k == "AGENT_TOKEN" {
				continue
			}
			envCopy[k] = v
		}

		response = bootstrapResponse{
			AgentToken:     agentToken,
			AgentBinaryURL: os.Getenv("HQ_AGENT_BINARY_URL"),
			InstanceName:   instance.Name,
			BaseRepoURL:    instance.BaseRepoURL,
			BaseRepoRef:    instance.BaseRepoRef,
			ComposePath:    instance.ComposePath,
			Environment:    envCopy,
		}
		return nil
	})

	if err != nil {
		switch {
		case errors.Is(err, errBootstrapTokenUsed):
			utils.WriteError(w, http.StatusGone, "token already used")
		case errors.Is(err, errBootstrapTokenExpired):
			utils.WriteError(w, http.StatusGone, "token expired or not found")
		default:
			utils.WriteError(w, http.StatusInternalServerError, "bootstrap failed")
		}
		return
	}

	utils.WriteJSON(w, http.StatusOK, response)
}

// CreateToken generates a one-time bootstrap token for an existing instance.
func (h *BootstrapHandler) CreateToken(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var instance models.Instance
	if err := h.DB.First(&instance, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.WriteError(w, http.StatusNotFound, "instance not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "failed to retrieve instance")
		return
	}

	// generate token
	tok := make([]byte, 32)
	if _, err := rand.Read(tok); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "token generation failed")
		return
	}
	token := hex.EncodeToString(tok)

	// store only hash
	sum := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(sum[:])

	expires := time.Now().Add(24 * time.Hour)
	bt := models.BootstrapToken{
		TokenHash:  tokenHash,
		InstanceID: id,
		Used:       false,
		ExpiresAt:  &expires,
	}

	if err := h.DB.Create(&bt).Error; err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "failed to store bootstrap token")
		return
	}

	publicURL := os.Getenv("HQ_PUBLIC_URL")
	if publicURL == "" {
		publicURL = os.Getenv("HQ_URL")
	}
	if publicURL == "" {
		utils.WriteError(w, http.StatusInternalServerError, "HQ public URL is not configured")
		return
	}
	setupCmd := "wget -qO- \"" + publicURL + "/scripts/setup.sh\" | BOOTSTRAP_TOKEN=" + token + " bash"

	utils.WriteJSON(w, http.StatusCreated, map[string]any{
		"token":         token,
		"expires_at":    expires,
		"setup_command": setupCmd,
	})
}
