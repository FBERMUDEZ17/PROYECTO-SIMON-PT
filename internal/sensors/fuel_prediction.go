package sensors

import (
	"math"
	"sort"
	"time"
)

// TAREA: "cálculo predictivo de combustible: alerta si el nivel baja a
// <1 hora de autonomía" — regresión lineal simple sobre lecturas
// recientes para estimar la tasa de consumo (unit tests en
// fuel_prediction_test.go).

// LowFuelThresholdHours es el umbral de autonomía por debajo del cual se
// dispara la alerta de combustible bajo.
const LowFuelThresholdHours = 1.0

// FuelPoint es un punto (tiempo, nivel de combustible %) usado para estimar
// la tasa de consumo.
type FuelPoint struct {
	RecordedAt time.Time
	FuelLevel  float64
}

// PredictionResult es el resultado de estimar la autonomía de combustible.
type PredictionResult struct {
	AutonomyHours float64 // +Inf si el nivel no está decreciendo
	RateFuelPerHr float64 // pendiente estimada, %/hora (negativa = consumiendo)
	Ok            bool    // false si no hay datos suficientes para predecir
}

// LowFuelAlert indica si el resultado amerita disparar la alerta.
func (p PredictionResult) LowFuelAlert() bool {
	return p.Ok && p.AutonomyHours < LowFuelThresholdHours
}

// TAREA: núcleo del cálculo predictivo — mínimos cuadrados sobre
// (tiempo, %combustible) para estimar la pendiente de consumo y proyectar
// la autonomía restante desde el nivel más reciente.
//
// PredictAutonomy estima cuántas horas de autonomía de combustible quedan,
// ajustando una regresión lineal simple (mínimos cuadrados) sobre las
// lecturas recientes de nivel de combustible en función del tiempo, y
// proyectando desde el nivel más reciente con la tasa de consumo estimada.
//
// Requiere al menos 2 puntos con timestamps distintos; en cualquier otro
// caso Ok es false y no se puede predecir todavía.
func PredictAutonomy(points []FuelPoint) PredictionResult {
	if len(points) < 2 {
		return PredictionResult{Ok: false}
	}

	sorted := make([]FuelPoint, len(points))
	copy(sorted, points)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].RecordedAt.Before(sorted[j].RecordedAt) })

	t0 := sorted[0].RecordedAt
	var n, sumX, sumY, sumXY, sumXX float64
	for _, p := range sorted {
		x := p.RecordedAt.Sub(t0).Hours()
		y := p.FuelLevel
		n++
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}

	denom := n*sumXX - sumX*sumX
	if denom == 0 {
		// Todos los timestamps son iguales: no hay variación temporal
		// para estimar una tasa.
		return PredictionResult{Ok: false}
	}
	slope := (n*sumXY - sumX*sumY) / denom // %/hora

	if slope >= 0 {
		// El nivel no está bajando (estable o recargando): sin alerta.
		return PredictionResult{AutonomyHours: math.Inf(1), RateFuelPerHr: slope, Ok: true}
	}

	latestFuel := sorted[len(sorted)-1].FuelLevel
	autonomy := latestFuel / -slope
	if autonomy < 0 {
		autonomy = 0
	}

	return PredictionResult{AutonomyHours: autonomy, RateFuelPerHr: slope, Ok: true}
}
