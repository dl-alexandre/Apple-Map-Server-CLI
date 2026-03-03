package commands

import (
	"errors"
	"testing"
)

func TestParseCoordinate(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantLat   float64
		wantLng   float64
		wantError error
	}{
		{
			name:    "valid coordinates",
			input:   "37.7749,-122.4194",
			wantLat: 37.7749,
			wantLng: -122.4194,
		},
		{
			name:    "valid with whitespace",
			input:   "  37.7749 , -122.4194  ",
			wantLat: 37.7749,
			wantLng: -122.4194,
		},
		{
			name:      "empty string",
			input:     "",
			wantError: ErrInvalidCoordinateFormat,
		},
		{
			name:      "single value",
			input:     "37.7749",
			wantError: ErrInvalidCoordinateFormat,
		},
		{
			name:      "three values",
			input:     "37.7749,-122.4194,100",
			wantError: ErrInvalidCoordinateFormat,
		},
		{
			name:      "non-numeric latitude",
			input:     "abc,-122.4194",
			wantError: ErrInvalidCoordinateFormat,
		},
		{
			name:      "non-numeric longitude",
			input:     "37.7749,abc",
			wantError: ErrInvalidCoordinateFormat,
		},
		{
			name:    "latitude at limit",
			input:   "90.0,0.0",
			wantLat: 90.0,
			wantLng: 0.0,
		},
		{
			name:      "latitude above max",
			input:     "90.1,0.0",
			wantError: ErrInvalidLatitude,
		},
		{
			name:      "latitude below min",
			input:     "-90.1,0.0",
			wantError: ErrInvalidLatitude,
		},
		{
			name:    "longitude at limit",
			input:   "0.0,180.0",
			wantLat: 0.0,
			wantLng: 180.0,
		},
		{
			name:      "longitude above max",
			input:     "0.0,180.1",
			wantError: ErrInvalidLongitude,
		},
		{
			name:      "longitude below min",
			input:     "0.0,-180.1",
			wantError: ErrInvalidLongitude,
		},
		{
			name:    "equator",
			input:   "0.0,0.0",
			wantLat: 0.0,
			wantLng: 0.0,
		},
		{
			name:    "negative coordinates",
			input:   "-33.8688,151.2093",
			wantLat: -33.8688,
			wantLng: 151.2093,
		},
		{
			name:    "high precision",
			input:   "37.77490123456789,-122.41939876543210",
			wantLat: 37.77490123456789,
			wantLng: -122.41939876543210,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lat, lng, err := parseCoordinate(tt.input)

			if tt.wantError != nil {
				if err == nil {
					t.Errorf("parseCoordinate(%q) expected error %v, got nil", tt.input, tt.wantError)
					return
				}
				if !errors.Is(err, tt.wantError) {
					t.Errorf("parseCoordinate(%q) error = %v, want error wrapping %v", tt.input, err, tt.wantError)
				}
				return
			}

			if err != nil {
				t.Errorf("parseCoordinate(%q) unexpected error = %v", tt.input, err)
				return
			}

			if lat != tt.wantLat || lng != tt.wantLng {
				t.Errorf("parseCoordinate(%q) = (%v, %v), want (%v, %v)", tt.input, lat, lng, tt.wantLat, tt.wantLng)
			}
		})
	}
}

func TestParseBoundingBox(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantNorth float64
		wantEast  float64
		wantSouth float64
		wantWest  float64
		wantError error
	}{
		{
			name:      "valid bounding box",
			input:     "37.8,-122.4,37.7,-122.5",
			wantNorth: 37.8,
			wantEast:  -122.4,
			wantSouth: 37.7,
			wantWest:  -122.5,
		},
		{
			name:      "valid with whitespace",
			input:     "  37.8 , -122.4 , 37.7 , -122.5  ",
			wantNorth: 37.8,
			wantEast:  -122.4,
			wantSouth: 37.7,
			wantWest:  -122.5,
		},
		{
			name:      "empty string",
			input:     "",
			wantError: ErrInvalidBoundingBoxFormat,
		},
		{
			name:      "three values",
			input:     "37.8,-122.4,37.7",
			wantError: ErrInvalidBoundingBoxFormat,
		},
		{
			name:      "five values",
			input:     "37.8,-122.4,37.7,-122.5,extra",
			wantError: ErrInvalidBoundingBoxFormat,
		},
		{
			name:      "non-numeric north",
			input:     "abc,-122.4,37.7,-122.5",
			wantError: ErrInvalidBoundingBoxFormat,
		},
		{
			name:      "non-numeric east",
			input:     "37.8,abc,37.7,-122.5",
			wantError: ErrInvalidBoundingBoxFormat,
		},
		{
			name:      "non-numeric south",
			input:     "37.8,-122.4,abc,-122.5",
			wantError: ErrInvalidBoundingBoxFormat,
		},
		{
			name:      "non-numeric west",
			input:     "37.8,-122.4,37.7,abc",
			wantError: ErrInvalidBoundingBoxFormat,
		},
		{
			name:      "north equals south",
			input:     "37.8,-122.4,37.8,-122.5",
			wantError: ErrInvalidBoundingBox,
		},
		{
			name:      "north less than south",
			input:     "37.7,-122.4,37.8,-122.5",
			wantError: ErrInvalidBoundingBox,
		},
		{
			name:      "east equals west",
			input:     "37.8,-122.4,37.7,-122.4",
			wantError: ErrInvalidBoundingBox,
		},
		{
			name:      "east less than west",
			input:     "37.8,-122.5,37.7,-122.4",
			wantError: ErrInvalidBoundingBox,
		},
		{
			name:      "latitude out of range north",
			input:     "90.1,-122.4,37.7,-122.5",
			wantError: ErrInvalidLatitude,
		},
		{
			name:      "latitude out of range south",
			input:     "37.8,-122.4,-90.1,-122.5",
			wantError: ErrInvalidLatitude,
		},
		{
			name:      "longitude out of range east",
			input:     "37.8,180.1,37.7,-122.5",
			wantError: ErrInvalidLongitude,
		},
		{
			name:      "longitude out of range west",
			input:     "37.8,-122.4,37.7,-180.1",
			wantError: ErrInvalidLongitude,
		},
		{
			name:      "world bounding box",
			input:     "90.0,180.0,-90.0,-180.0",
			wantNorth: 90.0,
			wantEast:  180.0,
			wantSouth: -90.0,
			wantWest:  -180.0,
		},
		{
			name:      "small bounding box",
			input:     "0.001,0.001,-0.001,-0.001",
			wantNorth: 0.001,
			wantEast:  0.001,
			wantSouth: -0.001,
			wantWest:  -0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			north, east, south, west, err := parseBoundingBox(tt.input)

			if tt.wantError != nil {
				if err == nil {
					t.Errorf("parseBoundingBox(%q) expected error %v, got nil", tt.input, tt.wantError)
					return
				}
				if !errors.Is(err, tt.wantError) {
					t.Errorf("parseBoundingBox(%q) error = %v, want error wrapping %v", tt.input, err, tt.wantError)
				}
				return
			}

			if err != nil {
				t.Errorf("parseBoundingBox(%q) unexpected error = %v", tt.input, err)
				return
			}

			if north != tt.wantNorth || east != tt.wantEast || south != tt.wantSouth || west != tt.wantWest {
				t.Errorf("parseBoundingBox(%q) = (%v, %v, %v, %v), want (%v, %v, %v, %v)",
					tt.input, north, east, south, west, tt.wantNorth, tt.wantEast, tt.wantSouth, tt.wantWest)
			}
		})
	}
}
