package wintec202

import (
	"bytes"
	"io"
	"io/ioutil"
	"time"
)

const (
	bitsPerSecond = 6
	bitsPerMinute = 6
	bitsPerHour = 5
	bitsPerDay = 5
	bitsPerMonth = 4
	bitsPerYear = 6
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

func (b *buffer) loadU16() (uint16, error) {
	if b.position+2 > len(b.data) {
		return 0, io.EOF
	}
	b.position += 2
	return uint16(b.data[b.position-2]) | (uint16(b.data[b.position-1]) << 8), nil
}

func storeU16(b *bytes.Buffer, u16 uint16) error {
	buf := []byte{
		byte(u16 & 0xff),
		byte((u16 >> 8) & 0xff),
	}
	_, err := b.Write(buf)
	return err
}

func (b *buffer) loadU32() (uint32, error) {
	if b.position+4 > len(b.data) {
		return 0, io.EOF
	}
	b.position += 4
	return uint32(b.data[b.position-4]) | (uint32(b.data[b.position-3]) << 8) | (uint32(b.data[b.position-2]) << 16) | (uint32(b.data[b.position-1]) << 24), nil
}

func storeU32(b *bytes.Buffer, u32 uint32) error {
	buf := []byte{
		byte(u32 & 0xff),
		byte((u32 >> 8) & 0xff),
		byte((u32 >> 16) & 0xff),
		byte((u32 >> 24) & 0xff),
	}
	_, err := b.Write(buf)
	return err
}

// loadBits reads n bits from val, returning val shifted down, and the n bits.
func loadBits(val uint32, n uint) (uint32, int) {
	return val >> n, int(val & uint32((1<<n)-1))
}

func storeBits(val uint32, n uint, data uint32) uint32 {
	return (val << n) | data
}

func loadTesDate(b *buffer) (time.Time, error) {
	val, err := b.loadU32()
	if err != nil {
		return time.Time{}, err
	}

	val, sec := loadBits(val, bitsPerSecond)
	val, min := loadBits(val, bitsPerMinute)
	val, hour := loadBits(val, bitsPerHour)
	val, day := loadBits(val, bitsPerDay)
	val, month := loadBits(val, bitsPerMonth)
	_, year := loadBits(val, bitsPerYear)

	return time.Date(year+2000, time.Month(month), day, hour, min, sec, 0, time.UTC), nil
}

func storeTesDate(b *bytes.Buffer, t time.Time) error {
	val := uint32(0)
	val = storeBits(val, bitsPerYear, uint32(t.Year() - 2000))
	val = storeBits(val, bitsPerMonth, uint32(t.Month()))
	val = storeBits(val, bitsPerDay, uint32(t.Day()))
	val = storeBits(val, bitsPerHour, uint32(t.Hour()))
	val = storeBits(val, bitsPerMinute, uint32(t.Minute()))
	val = storeBits(val, bitsPerSecond, uint32(t.Second()))
	return storeU32(b, val)
}

func loadTesPosition(b *buffer) (float64, error) {
	val, err := b.loadU32()
	if err != nil {
		return 0, err
	}
	return float64(int32(val)) / 10000000.0, nil
}

func storeTesPosition(b *bytes.Buffer, f float64) error {
	val := int32(f * 10000000.0)
	return storeU32(b, uint32(val))
}

func loadTesRecord(b *buffer) (GPS, error) {
	flags, err := b.loadU16()
	if err != nil {
		return GPS{}, err
	}
	date, err := loadTesDate(b)
	if err != nil {
		return GPS{}, err
	}
	lat, err := loadTesPosition(b)
	if err != nil {
		return GPS{}, err
	}
	lng, err := loadTesPosition(b)
	if err != nil {
		return GPS{}, err
	}
	alt, err := b.loadU16()
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

func storeTesRecord(b *bytes.Buffer, g GPS) error {
	if g.Marker {
		g.RawFlags = g.RawFlags | 0x2
	} else {
		g.RawFlags = g.RawFlags & ^0x2
	}
	if err := storeU16(b, uint16(g.RawFlags)); err != nil {
		return err
	}
	if err := storeTesDate(b, g.Time); err != nil {
		return err
	}
	if err := storeTesPosition(b, g.Lat); err != nil {
		return err
	}
	if err := storeTesPosition(b, g.Lng); err != nil {
		return err
	}
	if err := storeU16(b, uint16(g.Altitude)); err != nil {
		return err
	}
	return nil
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
		switch gps, err := loadTesRecord(b); err {
		case nil:
			records = append(records, gps)
		case io.EOF:
			return records, nil
		default:
			return nil, err
		}
	}
}

func StoreTES(output io.Writer, data []GPS) error {
	b := bytes.Buffer{}
	for _, g := range data {
		err := storeTesRecord(&b, g)
		if err != nil {
			return err
		}
	}
	_, err := output.Write(b.Bytes())
	return err
}