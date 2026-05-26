package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type applyRequest struct {
	BaseRepoURL string            `json:"base_repo_url"`
	BaseRepoRef string            `json:"base_repo_ref"`
	ComposePath string            `json:"compose_path"`
	Environment map[string]string `json:"environment"`
}

type applyResponse struct {
	Success bool   `json:"success"`
	Log     string `json:"log"`
}

func writeEnvFile(dir string, env map[string]string) error {
	fpath := filepath.Join(dir, ".env")
	f, err := os.Create(fpath)
	if err != nil {
		return err
	}
	defer f.Close()

	for k, v := range env {
		if _, err := fmt.Fprintf(f, "%s=%s\n", k, v); err != nil {
			return err
		}
	}
	return nil
}

func applyHandler(token string, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != fmt.Sprintf("Bearer %s", token) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var req applyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}

		tmp, err := os.MkdirTemp("", "hq-agent-")
		if err != nil {
			http.Error(w, "cannot create tmp dir", http.StatusInternalServerError)
			return
		}
		defer os.RemoveAll(tmp)

		// clone repo
		cloneDir := filepath.Join(tmp, "repo")
		cloneCmd := exec.Command("git", "clone", "--depth", "1", "--branch", req.BaseRepoRef, req.BaseRepoURL, cloneDir)
		out, err := cloneCmd.CombinedOutput()
		if err != nil {
			// try a plain git clone without branch
			cloneCmd = exec.Command("git", "clone", req.BaseRepoURL, cloneDir)
			out2, err2 := cloneCmd.CombinedOutput()
			out = append(out, out2...)
			if err2 != nil {
				resp := applyResponse{Success: false, Log: string(out)}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(resp)
				return
			}
		}

		// write env file
		workDir := cloneDir
		if req.ComposePath != "" && req.ComposePath != "." {
			workDir = filepath.Join(cloneDir, req.ComposePath)
		}

		if err := writeEnvFile(workDir, req.Environment); err != nil {
			http.Error(w, "cannot write env", http.StatusInternalServerError)
			return
		}

		// run docker compose up
		cmd := exec.Command("docker", "compose", "up", "--build", "-d")
		cmd.Dir = workDir
		log.Printf("running: docker compose up in %s", workDir)
		out, err = cmd.CombinedOutput()
		res := applyResponse{Success: err == nil, Log: string(out)}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	}
}

func main() {
	var addr string
	var token string
	flag.StringVar(&addr, "listen", ":8085", "agent listen address")
	flag.StringVar(&token, "token", "", "agent auth token (required)")
	flag.Parse()

	if token == "" {
		log.Fatal("agent token is required")
	}

	http.HandleFunc("/apply", applyHandler(token, 10*time.Minute))

	id := uuid.New().String()
	log.Printf("Headquarter agent starting (%s) on %s", id, addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("listen failed: %v", err)
	}
}
