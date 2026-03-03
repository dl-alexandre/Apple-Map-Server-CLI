package commands

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	ErrInvalidCoordinateFormat  = errors.New("invalid coordinate format, expected lat,lng")
	ErrInvalidBoundingBoxFormat = errors.New("invalid bounding box format, expected north,east,south,west")
	ErrInvalidLatitude          = errors.New("latitude must be between -90 and 90")
	ErrInvalidLongitude         = errors.New("longitude must be between -180 and 180")
	ErrInvalidBoundingBox       = errors.New("invalid bounding box: north must be > south and east must be > west")
)

func parseCoordinate(s string) (lat, lng float64, err error) {
	parts := strings.Split(strings.TrimSpace(s), ",")
	if len(parts) != 2 {
		return 0, 0, ErrInvalidCoordinateFormat
	}

	latStr := strings.TrimSpace(parts[0])
	lngStr := strings.TrimSpace(parts[1])

	lat, err = strconv.ParseFloat(latStr, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: invalid latitude %q", ErrInvalidCoordinateFormat, latStr)
	}

	lng, err = strconv.ParseFloat(lngStr, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: invalid longitude %q", ErrInvalidCoordinateFormat, lngStr)
	}

	if lat < -90 || lat > 90 {
		return 0, 0, ErrInvalidLatitude
	}

	if lng < -180 || lng > 180 {
		return 0, 0, ErrInvalidLongitude
	}

	return lat, lng, nil
}

func parseBoundingBox(s string) (north, east, south, west float64, err error) {
	parts := strings.Split(strings.TrimSpace(s), ",")
	if len(parts) != 4 {
		return 0, 0, 0, 0, ErrInvalidBoundingBoxFormat
	}

	northStr := strings.TrimSpace(parts[0])
	eastStr := strings.TrimSpace(parts[1])
	southStr := strings.TrimSpace(parts[2])
	westStr := strings.TrimSpace(parts[3])

	north, err = strconv.ParseFloat(northStr, 64)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("%w: invalid north %q", ErrInvalidBoundingBoxFormat, northStr)
	}

	east, err = strconv.ParseFloat(eastStr, 64)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("%w: invalid east %q", ErrInvalidBoundingBoxFormat, eastStr)
	}

	south, err = strconv.ParseFloat(southStr, 64)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("%w: invalid south %q", ErrInvalidBoundingBoxFormat, southStr)
	}

	west, err = strconv.ParseFloat(westStr, 64)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("%w: invalid west %q", ErrInvalidBoundingBoxFormat, westStr)
	}

	if north <= south {
		return 0, 0, 0, 0, ErrInvalidBoundingBox
	}

	if east <= west {
		return 0, 0, 0, 0, ErrInvalidBoundingBox
	}

	if north < -90 || north > 90 || south < -90 || south > 90 {
		return 0, 0, 0, 0, ErrInvalidLatitude
	}

	if east < -180 || east > 180 || west < -180 || west > 180 {
		return 0, 0, 0, 0, ErrInvalidLongitude
	}

	return north, east, south, west, nil
}
