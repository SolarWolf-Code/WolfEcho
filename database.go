package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	_ "github.com/mattn/go-sqlite3"
)

type DbOperation struct {
    query string
    args  []interface{}
}

var (
	WriteQueue = make(chan DbOperation, 100) // adjust the size as needed
	Db *sql.DB
)

func execStatement(query string, args ...interface{}) error {
	stmt, err := Db.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(args...)
	return err
}

// tableExists checks if a table with the given name exists in the database
func tableExists(tableName string) bool {
	query := fmt.Sprintf("SELECT 1 FROM %s LIMIT 1", tableName)
	_, err := Db.Query(query)
	return err == nil
}

func createTable(tableName string, args... string) {
	exists := tableExists(tableName)
	if !exists {
		log.Infof("'%s' not found in db, creating it with the following columns: %s", tableName, strings.Join(args, ", "))
		columns := strings.Join(args, ", ")
		query := fmt.Sprintf("CREATE TABLE %s (%s)", tableName, columns)
	
		_, err := Db.Exec(query)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func deleteOldReminders() {
	WriteQueue <- DbOperation{
		query: "DELETE FROM reminders WHERE time < ?",
		args: []interface{}{time.Now().Unix()},
	}
	log.Info("Added deleteOldReminders to writeQueue")
}

func InitializeDatabase() {
	log.Info("Initializing database...")
	// creates db if it doesn't exist

	var err error
    Db, err = sql.Open("sqlite3", "./wolfecho.db")
	if err != nil {
		log.Fatal(err)
	}

	// start goroutine from write queue
	go func() {
		for op := range WriteQueue {
			err := execStatement(op.query, op.args...)
			if err != nil {
				log.Errorf("error writing to db: %s", err)
			}
		}
	}()

	// create reminder table if it doesn't exist
	createTable("reminders", "messageid string PRIMARY KEY", "authorid string", "channelid string", "time INTEGER", "message TEXT")

	// delete old reminders
	deleteOldReminders()
}