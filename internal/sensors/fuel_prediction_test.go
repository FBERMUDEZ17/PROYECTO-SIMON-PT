// TAREA: unit tests de "fuel_prediction" (pedidos explícitamente) —
// datos insuficientes, timestamps iguales, consumo decreciente, nivel
// estable/recarga, alerta <1h, regresión con ruido.
package sensors

import (
	"math"
	"testing"
	"time"
)

func mustPoints(base time.Time, offsetsHours []float64, fuelLevels []float64) []FuelPoint {
	points := make([]FuelPoint, len(offsetsHours))
	for i := range offsetsHours {
		points[i] = FuelPoint{
			RecordedAt: base.Add(time.Duration(offsetsHours[i] * float64(time.Hour))),
			FuelLevel:  fuelLevels[i],
		}
	}
	return points
}

func TestPredictAutonomy_InsufficientData(t *testing.T) {
	now := time.Now()

	cases := [][]FuelPoint{
		nil,
		{},
		{{RecordedAt: now, FuelLevel: 50}},
	}
	for _, points := range cases {
		result := PredictAutonomy(points)
		if result.Ok {
			t.Errorf("con %d punto(s) se esperaba Ok=false, got %+v", len(points), result)
		}
	}
}

func TestPredictAutonomy_SameTimestamp(t *testing.T) {
	now := time.Now()
	points := []FuelPoint{
		{RecordedAt: now, FuelLevel: 50},
		{RecordedAt: now, FuelLevel: 40}, // mismo instante: no hay pendiente temporal
	}

	result := PredictAutonomy(points)
	if result.Ok {
		t.Errorf("con timestamps idénticos se esperaba Ok=false, got %+v", result)
	}
}

func TestPredictAutonomy_DecreasingFuel(t *testing.T) {
	base := time.Now()
	// 50% en t=0h, 40% en t=1h -> consumo de 10%/h
	points := mustPoints(base, []float64{0, 1}, []float64{50, 40})

	result := PredictAutonomy(points)
	if !result.Ok {
		t.Fatalf("se esperaba una predicción válida")
	}
	// autonomía = nivel_actual(40) / tasa(10 %/h) = 4h
	if math.Abs(result.AutonomyHours-4) > 1e-6 {
		t.Errorf("AutonomyHours = %v, want 4", result.AutonomyHours)
	}
	if result.LowFuelAlert() {
		t.Errorf("4h de autonomía no debería disparar la alerta de <1h")
	}
}

func TestPredictAutonomy_StableOrIncreasingFuel_NoAlert(t *testing.T) {
	base := time.Now()

	stable := mustPoints(base, []float64{0, 1, 2}, []float64{50, 50, 50})
	result := PredictAutonomy(stable)
	if !result.Ok {
		t.Fatalf("se esperaba una predicción válida para nivel estable")
	}
	if !math.IsInf(result.AutonomyHours, 1) {
		t.Errorf("nivel estable: AutonomyHours = %v, want +Inf", result.AutonomyHours)
	}
	if result.LowFuelAlert() {
		t.Errorf("nivel estable no debería disparar alerta")
	}

	increasing := mustPoints(base, []float64{0, 1}, []float64{30, 60}) // recarga
	result = PredictAutonomy(increasing)
	if !result.Ok || !math.IsInf(result.AutonomyHours, 1) {
		t.Errorf("nivel subiendo (recarga): got %+v", result)
	}
}

func TestPredictAutonomy_LowFuelTriggersAlert(t *testing.T) {
	base := time.Now()
	// 5% en t=0h, 2% en t=0.5h -> consumo de 6%/h -> autonomía = 2/6 = 0.333h (<1h)
	points := mustPoints(base, []float64{0, 0.5}, []float64{5, 2})

	result := PredictAutonomy(points)
	if !result.Ok {
		t.Fatalf("se esperaba una predicción válida")
	}
	if result.AutonomyHours >= LowFuelThresholdHours {
		t.Fatalf("AutonomyHours = %v, se esperaba < %v", result.AutonomyHours, LowFuelThresholdHours)
	}
	if !result.LowFuelAlert() {
		t.Errorf("se esperaba que LowFuelAlert() fuera true")
	}
}

func TestPredictAutonomy_UsesRegressionOverNoisyReadings(t *testing.T) {
	base := time.Now()
	// Tendencia general decreciente (~10%/h) con ruido de sensor punto a punto.
	points := mustPoints(base,
		[]float64{0, 1, 2, 3},
		[]float64{80, 71, 61, 50},
	)

	result := PredictAutonomy(points)
	if !result.Ok {
		t.Fatalf("se esperaba una predicción válida")
	}
	// tasa esperada ~ -10 %/h; autonomía ~ 50/10 = 5h, con tolerancia por el ruido.
	if result.AutonomyHours < 4 || result.AutonomyHours > 6 {
		t.Errorf("AutonomyHours = %v, se esperaba ~5h", result.AutonomyHours)
	}
}
