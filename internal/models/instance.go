package models

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Instance struct {
	ID             uuid.UUID         `gorm:"type:uuid;primaryKey" json:"id"`
	Name           string            `json:"name" gorm:"uniqueIndex;not null"`
	Server         string            `json:"server" gorm:"not null"`
	Domain         string            `json:"domain"`
	BaseRepoURL    string            `json:"base_repo_url" gorm:"not null"`
	BaseRepoRef    string            `json:"base_repo_ref" gorm:"not null;default:main"`
	ComposePath    string            `json:"compose_path" gorm:"not null;default:."`
	Environment    map[string]string  `json:"environment" gorm:"-"`
	EnvironmentRaw string            `json:"-" gorm:"type:text;not null;default:'{}'"`
	Status         string            `json:"status" gorm:"not null;default:draft"`
	LastAppliedAt  *time.Time        `json:"last_applied_at"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

func (i *Instance) BeforeCreate(tx *gorm.DB) (err error) {
	i.ID = uuid.New()
	return i.syncEnvironmentRaw()
}

func (i *Instance) BeforeSave(tx *gorm.DB) (err error) {
	return i.syncEnvironmentRaw()
}

func (i *Instance) AfterFind(tx *gorm.DB) (err error) {
	if i.EnvironmentRaw == "" {
		i.Environment = map[string]string{}
		return nil
	}

	var environment map[string]string
	if err := json.Unmarshal([]byte(i.EnvironmentRaw), &environment); err != nil {
		return err
	}

	if environment == nil {
		environment = map[string]string{}
	}

	i.Environment = environment
	return nil
}

func (i *Instance) syncEnvironmentRaw() error {
	if i.Environment == nil {
		i.Environment = map[string]string{}
	}

	raw, err := json.Marshal(i.Environment)
	if err != nil {
		return err
	}

	i.EnvironmentRaw = string(raw)
	return nil
}

func (i *Instance) RenderEnvironmentFile() string {
	if i.Environment == nil {
		return ""
	}

	keys := make([]string, 0, len(i.Environment))
	for key := range i.Environment {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var builder strings.Builder
	for _, key := range keys {
		builder.WriteString(key)
		builder.WriteString("=")
		builder.WriteString(i.Environment[key])
		builder.WriteString("\n")
	}

	return builder.String()
}