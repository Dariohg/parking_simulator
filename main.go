package main

import (
	"fmt"
	"parkingSimulator/src/models"
	"parkingSimulator/src/ui"
	"sync"
)

func main() {
	var wg sync.WaitGroup

	// Create the parking lot (model)
	parkingLot := models.NewParkingLot(20)
	subject := parkingLot.GetSubject()

	// Create the view
	view := ui.NewParkingLotView(subject)

	// Simulate vehicles in the background
	wg.Add(1)
	go func() {
		defer wg.Done()
		parkingLot.SimulateVehicles(&wg)
	}()

	// Stop simulation when the window is closed
	view.Window.SetOnClosed(func() {
		fmt.Println("Closing the simulator...")
		parkingLot.Stop()
		view.Close()
	})

	// Run the GUI
	view.Run()

	// Wait for all goroutines to finish
	wg.Wait()
	fmt.Println("Simulation finished.")
}
