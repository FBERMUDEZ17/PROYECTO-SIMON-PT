// TAREA: unit tests de "sensor_processing" (validación de entrada) —
// vehicle_id, lat/lon, fuel_level y temperatura fuera de rango.
package sensors

import "testing"

func validInput() ReadingInput {
	return ReadingInput{
		VehicleID:    "DEV-1234-XC54",
		Lat:          -34.6037,
		Lon:          -58.3816,
		FuelLevel:    55.5,
		TemperatureC: 25,
	}
}

func TestValidate_AcceptsValidInput(t *testing.T) {
	if err := validInput().Validate(); err != nil {
		t.Fatalf("input válido rechazado: %v", err)
	}
}

func TestValidate_RejectsMissingVehicleID(t *testing.T) {
	in := validInput()
	in.VehicleID = ""
	if err := in.Validate(); err != ErrMissingVehicleID {
		t.Fatalf("got %v, want ErrMissingVehicleID", err)
	}
}

func TestValidate_RejectsInvalidVehicleIDFormat(t *testing.T) {
	cases := []string{"dev with spaces", "dev/1234", "dev--", "..bad..", "dev_underscore"}
	for _, id := range cases {
		in := validInput()
		in.VehicleID = id
		if err := in.Validate(); err != ErrInvalidVehicleID {
			t.Errorf("vehicle_id=%q: got %v, want ErrInvalidVehicleID", id, err)
		}
	}
}

func TestValidate_RejectsOutOfRangeLatLon(t *testing.T) {
	cases := []struct {
		name string
		in   ReadingInput
		want error
	}{
		{"lat > 90", withLat(validInput(), 91), ErrInvalidLatitude},
		{"lat < -90", withLat(validInput(), -91), ErrInvalidLatitude},
		{"lon > 180", withLon(validInput(), 181), ErrInvalidLongitude},
		{"lon < -180", withLon(validInput(), -181), ErrInvalidLongitude},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.in.Validate(); err != tc.want {
				t.Errorf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func withLat(in ReadingInput, lat float64) ReadingInput { in.Lat = lat; return in }
func withLon(in ReadingInput, lon float64) ReadingInput { in.Lon = lon; return in }

func TestValidate_RejectsOutOfRangeFuelLevel(t *testing.T) {
	for _, fuel := range []float64{-1, 100.1, 200} {
		in := validInput()
		in.FuelLevel = fuel
		if err := in.Validate(); err != ErrInvalidFuelLevel {
			t.Errorf("fuel_level=%v: got %v, want ErrInvalidFuelLevel", fuel, err)
		}
	}
}

func TestValidate_RejectsOutOfRangeTemperature(t *testing.T) {
	for _, temp := range []float64{-51, 151} {
		in := validInput()
		in.TemperatureC = temp
		if err := in.Validate(); err != ErrInvalidTemp {
			t.Errorf("temperature=%v: got %v, want ErrInvalidTemp", temp, err)
		}
	}
}
