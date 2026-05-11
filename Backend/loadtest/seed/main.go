// Сидер тест-юзеров: insert в users (is_test=true) + dev-login → credentials.json.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Credential struct {
	Email string `json:"email"`
	UserID string `json:"userId"`
	AccessToken string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

func main() {
	baseURL := flag.String("base-url", os.Getenv("BASE_URL"), "Pinz API base URL")
	dbURL := flag.String("db-url", os.Getenv("LOADTEST_DB_URL"), "auth-service Postgres DSN")
	users := flag.Int("users", 100, "number of test users to create")
	concurrency := flag.Int("concurrency", 16, "parallel dev-login requests")
	out := flag.String("out", "credentials.json", "where to write credentials")
	flag.Parse()

	if *baseURL == "" || *dbURL == "" {
		log.Fatal("base-url and db-url are required (or BASE_URL/LOADTEST_DB_URL env)")
	}
	*baseURL = strings.TrimRight(*baseURL, "/")

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, *dbURL)
	if err != nil {
		log.Fatalf("pg connect: %v", err)
	}
	defer conn.Close(ctx)

	emails := make([]string, 0, *users)
	for i := 0; i < *users; i++ {
		emails = append(emails, fmt.Sprintf("loadtest-%d-%s@pinz.local", i, uuid.NewString()[:8]))
	}

	log.Printf("inserting %d test users…", *users)
	for i, email := range emails {
		_, err := conn.Exec(ctx,
			`INSERT INTO users (id, email, username, is_test) VALUES ($1, $2, $3, true)
			 ON CONFLICT (email) DO NOTHING`,
			uuid.New(), email, fmt.Sprintf("lt_%d", i),
		)
		if err != nil {
			log.Fatalf("insert user %d: %v", i, err)
		}
	}

	log.Printf("dev-login for %d users (concurrency=%d)…", *users, *concurrency)
	creds := make([]Credential, 0, *users)
	credCh := make(chan Credential, *users)
	jobCh := make(chan string, *users)
	var wg sync.WaitGroup
	httpc := &http.Client{Timeout: 15 * time.Second}
	for w := 0; w < *concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for email := range jobCh {
				c, err := devLogin(httpc, *baseURL, email)
				if err != nil {
					log.Printf("dev-login %s: %v", email, err)
					continue
				}
				credCh <- c
			}
		}()
	}
	for _, e := range emails {
		jobCh <- e
	}
	close(jobCh)
	go func() { wg.Wait(); close(credCh) }()
	for c := range credCh {
		creds = append(creds, c)
	}

	f, err := os.Create(*out)
	if err != nil {
		log.Fatalf("create out: %v", err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(creds); err != nil {
		log.Fatalf("encode: %v", err)
	}
	log.Printf("wrote %d credentials to %s", len(creds), *out)
}

func devLogin(c *http.Client, baseURL, email string) (Credential, error) {
	body, _ := json.Marshal(map[string]string{"email": email})
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/api/v1/auth/dev-login", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return Credential{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return Credential{}, fmt.Errorf("status %d: %s", resp.StatusCode, string(b))
	}
	var r struct {
		AccessToken string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return Credential{}, err
	}
	return Credential{Email: email, UserID: r.UserID, AccessToken: r.AccessToken, RefreshToken: r.RefreshToken}, nil
}
