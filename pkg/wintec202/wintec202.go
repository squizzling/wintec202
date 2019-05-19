package wintec202

import (
	"io"
	"io/ioutil"
	"time"
)

type GPS struct {
	Lat, Lng float64
	Time     time.Time

	// Altitude is a best guess for this field, and the units are unknown
	Altitude int

	// Marker indicates a marker was captured at this datapoint.
	Marker   bool

	// RawFlags exposes all 16 bits of the record header flags, as they're not all known.
	// Bit 0 might be a "first record" flag.
	// Bit 1 is the marker flag.
	RawFlags int
}

// buffer keeps things strongly typed.  It appears that passing an io.Reader around will cause escape analysis to
// believe the buffer escapes, causing a massive slow down.
type buffer struct {
	data     []byte
	position int
}

func (b *buffer) u16() (uint16, error) {
	if b.position+2 > len(b.data) {
		return 0, io.EOF
	}
	b.position += 2
	return uint16(b.data[b.position-2]) | (uint16(b.data[b.position-1]) << 8), nil
}

func (b *buffer) u32() (uint32, error) {
	if b.position+4 > len(b.data) {
		return 0, io.EOF
	}
	b.position += 4
	return uint32(b.data[b.position-4]) | (uint32(b.data[b.position-3]) << 8) | (uint32(b.data[b.position-2]) << 16) | (uint32(b.data[b.position-1]) << 24), nil
}

// bits reads n bits from val, returning val shifted down, and the n bits.
func bits(val uint32, n uint) (uint32, int) {
	return val >> n, int(val & uint32((1<<n)-1))
}

func tesDate(b *buffer) (time.Time, error) {
	val, err := b.u32()
	if err != nil {
		return time.Time{}, err
	}

	val, sec := bits(val, 6)
	val, min := bits(val, 6)
	val, hour := bits(val, 5)
	val, day := bits(val, 5)
	val, month := bits(val, 4)
	_, year := bits(val, 6)

	return time.Date(year+2000, time.Month(month), day, hour, min, sec, 0, time.UTC), nil
}

func tesPosition(b *buffer) (float64, error) {
	val, err := b.u32()
	if err != nil {
		return 0, err
	}
	return float64(int32(val)) / 10000000.0, nil
}

func tesRecord(b *buffer) (GPS, error) {
	flags, err := b.u16()
	if err != nil {
		return GPS{}, err
	}
	date, err := tesDate(b)
	if err != nil {
		return GPS{}, err
	}
	lat, err := tesPosition(b)
	if err != nil {
		return GPS{}, err
	}
	lng, err := tesPosition(b)
	if err != nil {
		return GPS{}, err
	}
	alt, err := b.u16()
	if err != nil {
		return GPS{}, err
	}

	return GPS{
		Lat:      lat,
		Lng:      lng,
		Altitude: int(alt),
		Time:     date,
		Marker:   (flags & 2) != 0,
		RawFlags: int(flags),
	}, nil
}

// LoadTES will read all records fom a .TES file and return them as a []GPS.  input data will be completely consumed
// unless it fails to read.  Data dangling at the end of the file will be read but ignored.
func LoadTES(input io.Reader) ([]GPS, error) {
	data, err := ioutil.ReadAll(input)
	if err != nil {
		return nil, err
	}

	b := &buffer{
		data:     data,
		position: 0,
	}

	records := make([]GPS, 0, len(data)/16)

	for {
		switch gps, err := tesRecord(b); err {
		case nil:
			records = append(records, gps)
		case io.EOF:
			return records, nil
		default:
			return nil, err
		}
	}
}
