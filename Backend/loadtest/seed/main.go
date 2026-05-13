// Сидер тест-юзеров и trips: insert в users (is_test=true) + dev-login + опционально
// создание N DRAFT trips через REST. Trips остаются в DRAFT_GROUPING_REVIEW (без ML
// pipeline их нельзя finalize), но они появляются в `GET /api/v1/trips` и нагружают
// list/get-ручки в нагрузочных сценариях.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
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
	tripsPerUser := flag.Int("trips-per-user", 0, "number of DRAFT trips to create per user (0 = skip)")
	concurrency := flag.Int("concurrency", 16, "parallel HTTP requests")
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
	httpc := &http.Client{Timeout: 15 * time.Second}
	creds := runWorkers(*concurrency, emails, func(email string) (Credential, bool) {
		c, err := devLogin(httpc, *baseURL, email)
		if err != nil {
			log.Printf("dev-login %s: %v", email, err)
			return Credential{}, false
		}
		return c, true
	})

	f, err := os.Create(*out)
	if err != nil {
		log.Fatalf("create out: %v", err)
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(creds); err != nil {
		f.Close()
		log.Fatalf("encode: %v", err)
	}
	f.Close()
	log.Printf("wrote %d credentials to %s", len(creds), *out)

	if *tripsPerUser > 0 {
		total := len(creds) * *tripsPerUser
		log.Printf("creating ~%d DRAFT trips (per user=%d, concurrency=%d)…", total, *tripsPerUser, *concurrency)
		var ok atomic.Int64
		jobs := make([]Credential, 0, total)
		for _, c := range creds {
			for i := 0; i < *tripsPerUser; i++ {
				jobs = append(jobs, c)
			}
		}
		runWorkers(*concurrency, jobs, func(c Credential) (struct{}, bool) {
			if err := createDraftTrip(httpc, *baseURL, c.AccessToken); err != nil {
				return struct{}{}, false
			}
			ok.Add(1)
			return struct{}{}, true
		})
		log.Printf("trips created: %d / %d", ok.Load(), total)
	}
}

// runWorkers — типизированный worker-pool: shards input через jobs[T]→outputs[R].
func runWorkers[T any, R any](concurrency int, jobs []T, fn func(T) (R, bool)) []R {
	jobCh := make(chan T, len(jobs))
	resCh := make(chan R, len(jobs))
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobCh {
				if r, ok := fn(j); ok {
					resCh <- r
				}
			}
		}()
	}
	for _, j := range jobs {
		jobCh <- j
	}
	close(jobCh)
	go func() { wg.Wait(); close(resCh) }()
	out := make([]R, 0, len(jobs))
	for r := range resCh {
		out = append(out, r)
	}
	return out
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

var categories = []string{"vacation", "business", "holidays", "active", "education", "custom"}
var seasons = []string{"winter", "spring", "summer", "autumn"}

func createDraftTrip(c *http.Client, baseURL, accessToken string) error {
	body := map[string]string{
		"name": fmt.Sprintf("loadtest-trip-%s", uuid.NewString()[:8]),
		"category": categories[rand.Intn(len(categories))],
		"season": seasons[rand.Intn(len(seasons))],
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/api/v1/trips/creation/start", strings.NewReader(string(b)))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}
