package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

const (
	dsn = "postgresql://postgres:example@localhost:5432/postgres?sslmode=disable"
)

func main() {
	ctx := context.Background()

	// create db connection
	db, cancel, err := registerPostgreSQL(ctx, dsn)
	if err != nil {
		panic(err)
	}
	defer cancel()

	// create some configs
	contexts, err := createContexts(db, 5000)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	guard := make(chan struct{}, 100)
	for _, id := range contexts {
		wg.Add(10)
		guard <- struct{}{}
		guard <- struct{}{}
		guard <- struct{}{}
		guard <- struct{}{}
		guard <- struct{}{}
		guard <- struct{}{}
		guard <- struct{}{}
		guard <- struct{}{}
		guard <- struct{}{}
		guard <- struct{}{}
		for i := 10; i > 0; i-- {
			version := i
			go func(id string, version int) {
				defer wg.Done()
				err := setConfigVersion(db, id, version)
				if err != nil {
					log.Printf("set config version: %s", err.Error())
				}
				<-guard
			}(id, version)
		}
	}

	wg.Wait()

	showResults(db)
}

func registerPostgreSQL(ctx context.Context, dsn string) (*sql.DB, func(), error) {
	const pingPostgresTimeout = 10 * time.Second

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("sql open: %w", err)
	}
	closeFunc := func() {
		_ = db.Close()
	}

	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithTimeout(ctx, pingPostgresTimeout)
	defer cancel()
	err = db.PingContext(ctx)
	if err != nil {
		closeFunc()
		return nil, nil, fmt.Errorf("db ping: %w", err)
	}

	return db, closeFunc, nil
}

func createContexts(db *sql.DB, count int) ([]string, error) {
	query := `insert into contexts (id, version) values ($1, 0)`

	var ids []string
	for i := 0; i < count; i++ {
		id := uuid.NewString()
		_, err := db.ExecContext(context.Background(), query, id)
		if err != nil {
			return nil, fmt.Errorf("insert context: %w", err)
		}

		ids = append(ids, id)
	}

	return ids, nil
}

func setConfigVersion(db *sql.DB, contextID string, version int) error {
	query := `
		INSERT INTO contexts (id, version) values ($1, $2)
		ON CONFLICT (id)
		DO UPDATE SET version=excluded.version where contexts.version < excluded.version
	`

	//log.Printf("set version=%d for context=%s", version, contextID)
	_, err := db.ExecContext(context.Background(), query, contextID, version)
	return err
}

func showResults(db *sql.DB) {
	query := `SELECT count(*), version from contexts group by version`
	rows, err := db.QueryContext(context.Background(), query)
	if err != nil {
		log.Printf("select counters query error: %s", err.Error())
		return
	}

	counters := map[int]int{}
	for rows.Next() {
		var total, version int
		err := rows.Scan(&total, &version)
		if err != nil {
			log.Printf("Row scan error: %s", err.Error())
			return
		}

		counters[version] = total
	}

	fmt.Printf("Counters: %v\n", counters)
}
