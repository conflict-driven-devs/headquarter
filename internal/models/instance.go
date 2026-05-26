package models

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
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

	// try to decrypt AGENT_TOKEN if encrypted
	keyB64 := os.Getenv("HQ_SECRET_KEY")
	if keyB64 != "" {
		if enc, ok := environment["AGENT_TOKEN"]; ok && strings.HasPrefix(enc, "ENC::") {
			raw, derr := decryptString(keyB64, strings.TrimPrefix(enc, "ENC::"))
			if derr == nil {
				environment["AGENT_TOKEN"] = raw
			}
		}
	}

	i.Environment = environment
	return nil
}

func (i *Instance) syncEnvironmentRaw() error {
	if i.Environment == nil {
		i.Environment = map[string]string{}
	}

	// if HQ_SECRET_KEY is set, encrypt AGENT_TOKEN in the environment before marshalling
	keyB64 := os.Getenv("HQ_SECRET_KEY")
	if keyB64 != "" {
		if v, ok := i.Environment["AGENT_TOKEN"]; ok {
			// if already encrypted, leave as-is
			if !strings.HasPrefix(v, "ENC::") {
				enc, err := encryptString(keyB64, v)
				if err != nil {
					return err
				}
				i.Environment["AGENT_TOKEN"] = "ENC::" + enc
			}
		}
	}

	raw, err := json.Marshal(i.Environment)
	if err != nil {
		return err
	}

	i.EnvironmentRaw = string(raw)
	return nil
}

func decodeKey(keyB64 string) ([]byte, error) {
	k, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil {
		return nil, err
	}
	if len(k) != 32 {
		return nil, errors.New("HQ_SECRET_KEY must be base64 of 32 bytes")
	}
	return k, nil
}

func encryptString(keyB64, plaintext string) (string, error) {
	key, err := decodeKey(keyB64)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	out := append(nonce, ct...)
	return base64.StdEncoding.EncodeToString(out), nil
}

func decryptString(keyB64, dataB64 string) (string, error) {
	key, err := decodeKey(keyB64)
	if err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(dataB64)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(raw) < ns {
		return "", errors.New("ciphertext too short")
	}
	nonce := raw[:ns]
	ct := raw[ns:]
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(pt), nil
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