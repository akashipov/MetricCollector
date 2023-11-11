package server

import (
	"database/sql"
	"fmt"
	"net/http"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB() {
	var err error
	DB, err = sql.Open("postgres", *PsqlInfo)
	fmt.Println("Initted DB:", DB)
	if err != nil {
		panic(err)
	}
	_, err = DB.Exec(
		"CREATE TABLE IF NOT EXISTS metrics (" +
			"id VARCHAR (50) UNIQUE NOT NULL," +
			"mtype VARCHAR (50) NOT NULL," +
			"value double precision," +
			"delta bigint" +
			")",
	)
	if err != nil {
		panic(err)
	}
	fmt.Println("Initted DB at the end of init method:", DB)
}

func TestConnectionPostgres(w http.ResponseWriter, request *http.Request) {
	err := DB.Ping()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
