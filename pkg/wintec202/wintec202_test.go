package wintec202_test

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/squizzling/wintec202/pkg/wintec202"

	"github.com/stretchr/testify/require"
)

func hexToRaw(s string) io.Reader {
	chars := strings.Split(s, " ")
	b := bytes.Buffer{}
	for _, ch := range chars {
		i, err := strconv.ParseInt(ch, 16, 16)
		if err != nil {
			panic(fmt.Sprintf("strconv.ParseInt: %v", err))
		}
		b.WriteByte(byte(i))
	}
	return bytes.NewReader(b.Bytes())
}

func TestEmpty(t *testing.T) {
	input := bytes.NewBufferString(``)
	output, err := wintec202.LoadTES(input)
	require.NoError(t, err)
	require.Equal(t, 0, len(output))
}

func TestBasicRecord(t *testing.T) {
	output, err := wintec202.LoadTES(hexToRaw("00 00 E3 C2 CC 46 20 01 FD 11 C0 54 B6 CE 25 00"))
	require.NoError(t, err)
	require.Equal(t, 1, len(output))
	expected := []wintec202.GPS{
		{
			Lat:  30.1793568,
			Lng:  -82.6911552,
			Altitude: 37,
			Time: time.Date(2017, 11, 06, 12, 11, 35, 0, time.UTC),
		},
	}
	require.Equal(t, expected, output)
}

func TestMultipleRecords(t *testing.T) {
	data := "00 00 E3 C2 CC 46 20 01 FD 11 C0 54 B6 CE 25 00 "
	data += "00 00 E4 C2 CC 46 00 01 FD 11 C0 54 B6 CE 25 00 "
	data += "00 00 E5 C2 CC 46 E0 00 FD 11 C0 54 B6 CE 25 00"

	output, err := wintec202.LoadTES(hexToRaw(data))
	require.NoError(t, err)
	require.Equal(t, 3, len(output))
	expected := []wintec202.GPS{
		{
			Lat:  30.1793568,
			Lng:  -82.6911552,
			Altitude: 37,
			Time: time.Date(2017, 11, 06, 12, 11, 35, 0, time.UTC),
		},
		{
			Lat:  30.1793536,
			Lng:  -82.6911552,
			Altitude: 37,
			Time: time.Date(2017, 11, 06, 12, 11, 36, 0, time.UTC),
		},
		{
			Lat:  30.1793504,
			Lng:  -82.6911552,
			Altitude: 37,
			Time: time.Date(2017, 11, 06, 12, 11, 37, 0, time.UTC),
		},
	}
	require.Equal(t, expected, output)
}

func TestExtraData(t *testing.T) {
	output, err := wintec202.LoadTES(hexToRaw("00 00 E3 C2 CC 46 20 01 FD 11 C0 54 B6 CE 25 00 FF FF"))
	require.NoError(t, err)
	require.Equal(t, 1, len(output))
	expected := []wintec202.GPS{
		{
			Lat:  30.1793568,
			Lng:  -82.6911552,
			Altitude: 37,
			Time: time.Date(2017, 11, 06, 12, 11, 35, 0, time.UTC),
		},
	}
	require.Equal(t, expected, output)
}

func TestRecordWithMarker(t *testing.T) {
	output, err := wintec202.LoadTES(hexToRaw("02 00 E3 C2 CC 46 20 01 FD 11 C0 54 B6 CE 25 00"))
	require.NoError(t, err)
	require.Equal(t, 1, len(output))
	expected := []wintec202.GPS{
		{
			Lat:      30.1793568,
			Lng:      -82.6911552,
			Altitude: 37,
			Time:     time.Date(2017, 11, 06, 12, 11, 35, 0, time.UTC),
			Marker:   true,
			RawFlags: 2,
		},
	}
	require.Equal(t, expected, output)
}
