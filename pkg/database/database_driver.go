package dbdriver

import (
	_ "SingleElevator/pkg/elevator"
	_ "SingleElevator/pkg/elevio"
	"database/sql"
	_ "fmt"
	"log"
	_ "log"
	_ "time"
	_"github.com/mattn/go-sqlite3"
)

type state interface{}

type DatabaseDriver interface{
	InitDataBase()
	SaveState(state) bool
	Loadstate() state
}

func DBInit(){
	// Connect to SQLite database, or create one if nonexsistant
	db, err := sql.Open("sqlite3", "./elev.db")
	if err != nil {
		log.Println(err)
		return
	}

	defer db.Close()
	log.Println("SQLite database connection established.")
	
}