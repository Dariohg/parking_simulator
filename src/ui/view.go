package ui

import (
	"fmt"
	"image/color"
	"parkingSimulator/src/models"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

type ParkingLotView struct {
	Window         fyne.Window
	mainContainer  *fyne.Container
	parkingSpots   []*canvas.Rectangle
	cars           map[int]*canvas.Image
	queueContainer *fyne.Container
	mutex          sync.Mutex
	updateChannel  chan models.Position
	CloseView      chan struct{}
	carImage       fyne.Resource
	carEntering    fyne.Resource
	carExiting     fyne.Resource
	carQueue       fyne.Resource
	lastQueueY     float32
	parkingLayout  *fyne.Container
	animationLayer *fyne.Container
	queueRoad      *canvas.Rectangle
	// Nuevo canal para sincronizar animaciones
	animationComplete chan struct{}
}

func NewParkingLotView(subject models.Subject) *ParkingLotView {
	myApp := app.New()
	window := myApp.NewWindow("Parking Lot Simulator")
	window.Resize(fyne.NewSize(1200, 700))
	window.SetFixedSize(true)

	view := &ParkingLotView{
		Window:            window,
		mainContainer:     container.NewWithoutLayout(),
		parkingSpots:      make([]*canvas.Rectangle, 20),
		cars:              make(map[int]*canvas.Image),
		updateChannel:     make(chan models.Position, 100),
		CloseView:         make(chan struct{}),
		queueContainer:    container.NewWithoutLayout(),
		lastQueueY:        600,
		animationLayer:    container.NewWithoutLayout(),
		animationComplete: make(chan struct{}, 1),
	}

	view.loadCarImages()
	view.createParkingLayout()
	view.createQueueRoad()
	view.setupMainLayout()

	subject.Register(view)
	go view.processUpdates()

	return view
}

func (v *ParkingLotView) loadCarImages() {
	//v.carImage = theme.FyneLogo() // Placeholder for parked cars
	v.carImage = loadImage("assets/car_entering.png")
	v.carEntering = loadImage("assets/car_entering.png")
	v.carExiting = loadImage("assets/car_exiting.png")
	v.carQueue = loadImage("assets/car_queue.png")
}

func loadImage(path string) fyne.Resource {
	res, err := fyne.LoadResourceFromPath(path)
	if err != nil {
		// Usar un placeholder si no se puede cargar
		return theme.FyneLogo()
	}
	return res
}

func (v *ParkingLotView) createParkingLayout() {
	v.parkingLayout = container.NewWithoutLayout()
	spotColor := color.RGBA{R: 200, G: 200, B: 200, A: 255}

	// Crear la puerta única
	gate := canvas.NewRectangle(color.RGBA{R: 100, G: 100, B: 100, A: 255})
	gate.Resize(fyne.NewSize(60, 20))
	gate.Move(fyne.NewPos(200, 350)) // Posición central para la puerta
	v.parkingLayout.Add(gate)

	for i := 0; i < 20; i++ {
		spot := canvas.NewRectangle(spotColor)
		x := float32(300 + (i%5)*150)
		y := float32(50 + (i/5)*120)
		spot.Resize(fyne.NewSize(120, 80))
		spot.Move(fyne.NewPos(x, y))
		spot.StrokeWidth = 2
		spot.StrokeColor = color.RGBA{R: 100, G: 100, B: 100, A: 255}
		v.parkingSpots[i] = spot
		v.parkingLayout.Add(spot)
	}
}

func (v *ParkingLotView) createQueueRoad() {
	v.queueRoad = canvas.NewRectangle(color.RGBA{R: 120, G: 120, B: 120, A: 255})
	v.queueRoad.Move(fyne.NewPos(20, 100))
	v.queueRoad.Resize(fyne.NewSize(150, 600))
	v.queueContainer.Add(v.queueRoad)
}

func (v *ParkingLotView) addToQueue(carID int) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	if _, exists := v.cars[carID]; exists {
		return
	}

	car := canvas.NewImageFromResource(v.carQueue)
	car.Resize(fyne.NewSize(40, 60))
	// Modificamos la posición inicial para que esté más cerca de la puerta
	startY := float32(350)             // Alineado con la puerta
	car.Move(fyne.NewPos(120, startY)) // Más cerca de la puerta (x=120 en lugar de 40)
	v.cars[carID] = car

	v.queueContainer.Add(car)
	v.queueContainer.Refresh()
	v.mainContainer.Refresh()

	// Ya no necesitamos animar la entrada a la cola
	v.lastQueueY = startY
}

func (v *ParkingLotView) setupMainLayout() {
	v.mainContainer.Add(v.queueRoad)
	v.mainContainer.Add(v.parkingLayout)
	v.mainContainer.Add(v.queueContainer)
	v.mainContainer.Add(v.animationLayer)
	v.Window.SetContent(v.mainContainer)
}

func (v *ParkingLotView) Update(pos models.Position) {
	select {
	case v.updateChannel <- pos:
	default:
		// Si el canal está lleno, descartar la actualización
		fmt.Printf("Warning: Update channel full, skipping update for car %d\n", pos.ID)
	}
}

func (v *ParkingLotView) processUpdates() {
	updateTicker := time.NewTicker(16 * time.Millisecond) // 60 FPS
	defer updateTicker.Stop()

	var pendingUpdates []models.Position

	for {
		select {
		case <-v.CloseView:
			return
		case pos := <-v.updateChannel:
			pendingUpdates = append(pendingUpdates, pos)
		case <-updateTicker.C:
			if len(pendingUpdates) > 0 {
				update := pendingUpdates[0]
				pendingUpdates = pendingUpdates[1:]
				v.handleUpdate(update)
			}
		}
	}
}

func (v *ParkingLotView) handleUpdate(pos models.Position) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	switch pos.Status {
	case "waiting in queue":
		go v.addToQueue(pos.ID)
	case "parked":
		// Esperar a que termine cualquier animación previa
		select {
		case <-v.animationComplete:
		default:
		}
		go v.animateToParking(pos.ID, pos.Spot)
	case "left":
		// Esperar a que termine cualquier animación previa
		select {
		case <-v.animationComplete:
		default:
		}
		go v.animateExit(pos.ID, pos.Spot)
	}
}

func (v *ParkingLotView) animateToParking(carID int, spotIndex int) {
	v.mutex.Lock()

	car, exists := v.cars[carID]
	if !exists {
		car = canvas.NewImageFromResource(v.carEntering)
		car.Resize(fyne.NewSize(120, 80))
		v.cars[carID] = car
	}

	// El auto aparece directamente en la puerta
	gateX := float32(200)
	gateY := float32(350)
	targetX := float32(300 + (spotIndex%5)*150)
	targetY := float32(50 + (spotIndex/5)*120)

	v.queueContainer.Remove(car)
	v.animationLayer.Add(car)
	car.Move(fyne.NewPos(gateX, gateY)) // Posicionamos directamente en la puerta
	v.mutex.Unlock()

	// Ahora solo animamos desde la puerta hasta el espacio de estacionamiento
	go func() {
		v.animateMove(car, fyne.NewPos(gateX, gateY), fyne.NewPos(targetX, targetY))

		v.mutex.Lock()
		v.parkingLayout.Add(car)
		v.animationLayer.Remove(car)
		v.mutex.Unlock()

		v.Window.Canvas().Refresh(v.mainContainer)
	}()
}

func (v *ParkingLotView) animateExit(carID int, spotIndex int) {
	v.mutex.Lock()

	car, exists := v.cars[carID]
	if !exists {
		v.mutex.Unlock()
		return
	}

	v.parkingSpots[spotIndex].FillColor = color.RGBA{R: 200, G: 200, B: 200, A: 255}
	v.parkingSpots[spotIndex].Refresh()

	car.Resource = v.carExiting
	startPos := fyne.NewPos(
		float32(300+(spotIndex%5)*150),
		float32(50+(spotIndex/5)*120),
	)
	gatePos := fyne.NewPos(200, 350)

	v.parkingLayout.Remove(car)
	v.animationLayer.Add(car)
	v.mutex.Unlock()

	go func() {
		// Una sola animación directa hasta la puerta
		v.animateMove(car, startPos, gatePos)

		v.mutex.Lock()
		v.animationLayer.Remove(car)
		delete(v.cars, carID)
		v.mutex.Unlock()

		v.Window.Canvas().Refresh(v.mainContainer)
	}()
}

func (v *ParkingLotView) animateMove(obj *canvas.Image, from, to fyne.Position) {
	const (
		fps      = 60
		delay    = time.Second / fps
		duration = 500 * time.Millisecond // Reducido de 500ms a 300ms para animaciones más rápidas
	)
	steps := int(duration / delay)
	deltaX := (to.X - from.X) / float32(steps)
	deltaY := (to.Y - from.Y) / float32(steps)

	for i := 0; i <= steps; i++ {
		start := time.Now()

		v.mutex.Lock()
		obj.Move(fyne.NewPos(
			from.X+deltaX*float32(i),
			from.Y+deltaY*float32(i),
		))
		obj.Refresh()
		v.mutex.Unlock()

		elapsed := time.Since(start)
		if elapsed < delay {
			time.Sleep(delay - elapsed)
		}
	}

	v.Window.Canvas().Refresh(v.mainContainer)
	v.animationComplete <- struct{}{}
}

func (v *ParkingLotView) animateQueueEntry(car *canvas.Image, fromY, toY float32) {
	const (
		fps      = 60
		delay    = time.Second / fps
		duration = 500 * time.Millisecond
	)
	steps := int(duration / delay)
	deltaY := (toY - fromY) / float32(steps)

	for i := 0; i <= steps; i++ {
		start := time.Now()

		v.mutex.Lock()
		car.Move(fyne.NewPos(40, fromY+deltaY*float32(i)))
		car.Refresh()
		v.queueContainer.Refresh()
		v.mutex.Unlock()

		elapsed := time.Since(start)
		if elapsed < delay {
			time.Sleep(delay - elapsed)
		}
	}

	v.Window.Canvas().Refresh(v.mainContainer)
}

func (v *ParkingLotView) Run() {
	v.Window.ShowAndRun()
}

func (v *ParkingLotView) Close() {
	close(v.CloseView)
}
