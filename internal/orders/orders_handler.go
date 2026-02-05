package orders

import (
	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
	statesync "github.com/Mosazghi/elevator-ttk4145/internal/sync"
)

type OrdersContext struct {
	worldView statesync.Worldview
}

type OrdersHandler interface {
	GetNextOrder(id int) (elevator.Behavior, elevio.MotorDirection)
}

func (context OrdersContext) GetNextOrder(id int) (elevator.Behavior, elevio.MotorDirection) {
	return elevator.BIdle, elevio.MDStop
}
func (context OrdersContext) findCost() int { return 0 }
