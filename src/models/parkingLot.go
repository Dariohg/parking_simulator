package models

import (
	"fmt"
	"math/rand"
	"parkingSimulator/src/characters"
	"sync"
	"time"
)

type ParkingLot struct {
	capacity     int
	currentCars  int
	spots        []bool
	waitingQueue []int
	gate         chan struct{}
	observers    []Observer
	stopSignal   chan struct{}
	mutex        sync.RWMutex
}

func NewParkingLot(capacity int) *ParkingLot {
	return &ParkingLot{
		capacity:     capacity,
		spots:        make([]bool, capacity),
		waitingQueue: make([]int, 0),
		gate:         make(chan struct{}, 1),
		observers:    make([]Observer, 0),
		stopSignal:   make(chan struct{}),
	}
}

// Genera intervalos de tiempo siguiendo una distribución de Poisson
func generatePoissonInterval(lambda float64) time.Duration {
	interval := rand.ExpFloat64() / lambda
	return time.Duration(interval * float64(time.Second))
}

func (p *ParkingLot) SimulateVehicles(wg *sync.WaitGroup) {
	defer wg.Done()
	vehicleID := 1

	// Lambda es la tasa media de llegadas (vehículos por segundo)
	lambda := 0.5 // Ajusta este valor según necesites

	// Genera llegadas usando distribución de Poisson
	go func() {
		for {
			select {
			case <-p.stopSignal:
				return
			default:
				// Tiempo entre llegadas sigue una distribución exponencial
				interArrivalTime := generatePoissonInterval(lambda)
				time.Sleep(interArrivalTime)

				vehicle := &characters.Car{ID: vehicleID}
				vehicleID++

				// Lanza una goroutine para cada intento de entrada
				go p.TryToEnter(vehicle)
			}
		}
	}()

	// Esperar señal de detención
	<-p.stopSignal
}

func (p *ParkingLot) findSpot() int {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for i, occupied := range p.spots {
		if !occupied {
			p.spots[i] = true
			p.currentCars++
			return i
		}
	}
	return -1
}

func (p *ParkingLot) TryToEnter(vehicle *characters.Car) {
	// Primero verifica si hay espacio disponible
	p.mutex.RLock()
	isFull := p.currentCars >= p.capacity
	p.mutex.RUnlock()

	if isFull {
		p.mutex.Lock()
		p.waitingQueue = append(p.waitingQueue, vehicle.ID)
		queueLen := len(p.waitingQueue)
		p.mutex.Unlock()

		fmt.Printf("Vehicle %d waiting (parking full). Queue: %d\n", vehicle.ID, queueLen)
		p.NotifyAll(Position{
			ID:            vehicle.ID,
			Status:        "waiting in queue",
			QueuePosition: queueLen,
		})
		return
	}

	// Intenta adquirir la puerta
	select {
	case p.gate <- struct{}{}:
		// Simula el tiempo de entrada
		time.Sleep(500 * time.Millisecond)
		spot := p.findSpot()

		if spot != -1 {
			fmt.Printf("Vehicle %d entering spot %d\n", vehicle.ID, spot)

			// Libera la puerta después de entrar
			<-p.gate

			// Notifica el estacionamiento exitoso
			p.NotifyAll(Position{
				ID:     vehicle.ID,
				Spot:   spot,
				Status: "parked",
			})

			// Simula tiempo de estacionamiento
			go func() {
				parkingTime := time.Duration(rand.Intn(2)+3) * time.Second
				time.Sleep(parkingTime)
				p.Leave(vehicle, spot)
			}()
		}
	case <-p.stopSignal:
		return
	}
}

func (p *ParkingLot) Leave(vehicle *characters.Car, spot int) {
	// Intenta adquirir la puerta para salir
	select {
	case p.gate <- struct{}{}:
		// Simula el tiempo de salida
		time.Sleep(500 * time.Millisecond)

		p.mutex.Lock()
		p.spots[spot] = false
		p.currentCars--

		// Procesa el siguiente vehículo en la cola
		var nextVehicle *characters.Car
		if len(p.waitingQueue) > 0 {
			nextID := p.waitingQueue[0]
			p.waitingQueue = p.waitingQueue[1:]
			nextVehicle = &characters.Car{ID: nextID}
		}
		p.mutex.Unlock()

		fmt.Printf("Vehicle %d leaving spot %d\n", vehicle.ID, spot)

		// Libera la puerta
		<-p.gate

		p.NotifyAll(Position{
			ID:     vehicle.ID,
			Spot:   spot,
			Status: "left",
		})

		// Procesa el siguiente vehículo si existe
		if nextVehicle != nil {
			go p.TryToEnter(nextVehicle)
		}
	case <-p.stopSignal:
		return
	}
}

func (p *ParkingLot) Register(observer Observer) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.observers = append(p.observers, observer)
}

func (p *ParkingLot) NotifyAll(pos Position) {
	p.mutex.RLock()
	observers := make([]Observer, len(p.observers))
	copy(observers, p.observers)
	p.mutex.RUnlock()

	for _, observer := range observers {
		observer.Update(pos)
	}
}

func (p *ParkingLot) Stop() {
	close(p.stopSignal)
}

func (p *ParkingLot) GetSubject() Subject {
	return p
}
