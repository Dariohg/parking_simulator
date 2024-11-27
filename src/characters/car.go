package characters

import (
	"fmt"
	"time"
)

type Car struct {
	ID int
}

// Simulates the time the vehicle is parked before leaving
func (c *Car) Park(duration int) {
	fmt.Printf("Vehicle %d parked for %d seconds\n", c.ID, duration)
	time.Sleep(time.Duration(duration) * time.Second)
	fmt.Printf("Vehicle %d is leaving\n", c.ID)
}
