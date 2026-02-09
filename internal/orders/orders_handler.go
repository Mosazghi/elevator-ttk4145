package orders

// import (
// 	"fmt"
// 	"time"
//
// 	"github.com/Mosazghi/elevator-ttk4145/internal/elevator"
// 	elevio "github.com/Mosazghi/elevator-ttk4145/internal/hw"
// 	statesync "github.com/Mosazghi/elevator-ttk4145/internal/sync"
// )
//
// type TempOrderHandler struct {
// 	Worldview *statesync.Worldview
// }
//
// type OrdersHandler interface {
// 	GetNextOrder(id int)
// }
//
// func (h *TempOrderHandler) GetNextOrder(id int, order chan<- elevator.Action) {
// 	dir := false
// 	originalOrderDir := 0
// 	for {
// 		view := h.Worldview
// 		hallCalls := view.GetAllHallCalls()
// 		elev := view.GetLocalElevator()
//
// 		for floor := range hallCalls {
// 			call := &hallCalls[floor]
//
// 			for i := range 2 {
// 				if call[i].State == statesync.HSNone {
// 					continue
// 				}
//
// 				if floor == elev.CurrentFloor {
// 					h.Worldview.SetHallCall(floor, statesync.HallCallDir(originalOrderDir), statesync.HSNone)
// 					order <- elevator.Action{elevator.BIdle, elevio.MDStop}
// 					dir = false
// 				}
//
// 				if call[i].State == statesync.HSProcessing {
// 					fmt.Println("target: ", floor, " floor: ", elev.CurrentFloor)
// 					fmt.Println("currently processing")
// 					continue
// 				}
//
// 				if dir {
// 					continue
// 				}
//
// 				fmt.Println("came here")
// 				dir = true
//
// 				// Decide direction
// 				if floor > elev.CurrentFloor {
// 					originalOrderDir = statesync.HDUp
// 					h.Worldview.SetHallCall(floor, statesync.HDUp, statesync.HSProcessing)
// 					order <- elevator.Action{elevator.BMoving, elevio.MDUp}
// 				}
//
// 				if floor < elev.CurrentFloor {
// 					originalOrderDir = statesync.HDDown
// 					h.Worldview.SetHallCall(floor, statesync.HDDown, statesync.HSProcessing)
// 					order <- elevator.Action{elevator.BMoving, elevio.MDDown}
// 				}
//
// 				if floor == elev.CurrentFloor {
// 					h.Worldview.SetHallCall(floor, statesync.HallCallDir(originalOrderDir), statesync.HSNone)
// 					order <- elevator.Action{elevator.BIdle, elevio.MDStop}
// 					dir = false
// 				}
//
// 			}
// 		}
//
// 		time.Sleep(20 * time.Millisecond) // tune; prevents 100% CPU
// 	}
// }
//
// // func (orderHandler *TempOrderHandler) GetNextOrder(id int, order chan elevator.Action) {
// // 	view := orderHandler.Worldview
// // 	hallCalls := view.GetAllHallCalls()
// // 	elev := view.GetLocalElevator()
// // 	fmt.Println("current location at start: ", elev.CurrentFloor)
// //
// // 	for floor := range hallCalls {
// // 		fmt.Println("currently on order!")
// // 		call := &hallCalls[floor]
// //
// // 		for i := range 2 {
// // 			if call[i].State == statesync.HSProcessing {
// // 				if floor == elev.CurrentFloor {
// // 					call[i].State = statesync.HSNone
// // 					call[i].By = -1
// // 				}
// // 				continue
// // 			}
// //
// // 			if call[i].State == statesync.HSNone {
// // 				continue
// // 			}
// //
// // 			if floor > elev.CurrentFloor {
// // 				order <- elevator.Action{elevator.BMoving, elevio.MDUp}
// // 			} else {
// // 				order <- elevator.Action{elevator.BMoving, elevio.MDDown}
// // 			}
// //
// // 			call[i].State = statesync.HSProcessing
// // 			call[i].By = id
// // 		}
// // 	}
// // }
