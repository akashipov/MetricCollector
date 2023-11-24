package server

import (
	"database/sql"
	"fmt"
	"net/http"
	"syscall"

	"github.com/akashipov/MetricCollector/internal/general"
	_ "github.com/lib/pq"
)

var DB *sql.DB
var OurStorage Storage

func InitDB() error {
	var err error
	if (PsqlInfo != nil) && (*PsqlInfo != "") {
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
		err = general.RetryCode(f, syscall.ECONNREFUSED)
		if err != nil {
			return err
		}
		fmt.Println("Successfully connected to the db")
	}
	OurStorage = NewStorage(nil)
	return nil
}

func TestConnectionPostgres(w http.ResponseWriter, request *http.Request) {
	f := func() error {
		return DB.Ping()
	}
	err := general.RetryCode(f, syscall.ECONNREFUSED)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
