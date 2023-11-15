package server

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"syscall"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func RetryCode(f func() error) error {
	sleepTime := time.Second
	countOfRepetition := 3
	for i := 0; i >= 0; i++ {
		err := f()
		if err != nil {
			isPsqlError := errors.Is(err, syscall.ECONNREFUSED)
			fmt.Println("isPsqlError:", isPsqlError)
			if isPsqlError && i < countOfRepetition {
				time.Sleep(sleepTime)
				fmt.Println("Repeating... SleepTime:", sleepTime)
				sleepTime += 2 * time.Second
				continue
			}
			return err
		}
		return nil
	}
	return nil
}

func InitDB() error {
	var err error
	DB, err = sql.Open("postgres", *PsqlInfo)
	if err != nil {
		return err
	}
	f := func() error {
		_, err = DB.Exec(
			"CREATE TABLE IF NOT EXISTS metrics (" +
				"id VARCHAR (50) PRIMARY KEY NOT NULL," +
				"mtype VARCHAR (50) NOT NULL," +
				"value double precision," +
				"delta bigint" +
				")",
		)
		return err
	}
	err = RetryCode(f)
	if err != nil {
		return err
	}
	fmt.Println("Successfully connected to the db")
	return nil
}

func TestConnectionPostgres(w http.ResponseWriter, request *http.Request) {
	err := DB.Ping()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
