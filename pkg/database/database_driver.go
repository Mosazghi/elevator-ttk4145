package dbdriver

import (
	"SingleElevator/pkg/elevator"
	"SingleElevator/pkg/elevio"
	"log"
	"fmt"
	"time"
	_ "github.com/mattn/go-sqlite3"
)

type DatabaseDriver interface{
	InitDataBase()
	SaveState(state) bool
	Loadstate() state
}

