//go:build !plan9 && !js

package arrowlist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/stretchr/testify/require"
)

func TestArrowJSONCompatibility(t *testing.T) {
	for _, tc := range []struct {
		name  string
		dtype arrow.DataType
		input string
	}{
		{"strings", arrow.BinaryTypes.String, `["quote\"",null,"é/雪","<>&","\u0000"]`},
		{"signed limits", arrow.PrimitiveTypes.Int64, `[0,-9223372036854775808,9223372036854775807,null]`},
		{"unsigned limits", arrow.PrimitiveTypes.Uint64, `[0,18446744073709551615,null]`},
		{"booleans", arrow.FixedWidthTypes.Boolean, `[true,false,null]`},
		{"binary", arrow.BinaryTypes.Binary, `["AAEC/w==",null,""]`},
		{"maps", arrow.MapOf(arrow.BinaryTypes.String, arrow.BinaryTypes.String), `[[{"key":"é","value":"雪"},{"key":"empty","value":null}],null,[]]`},
		{"floats", arrow.PrimitiveTypes.Float64, `[0.5,-1.25,null]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			alloc := memory.NewCheckedAllocator(memory.DefaultAllocator)
			defer alloc.AssertSize(t, 0)
			a, _, err := array.FromJSON(alloc, tc.dtype, strings.NewReader(tc.input))
			require.NoError(t, err)
			defer a.Release()
			encoded, err := a.MarshalJSON()
			require.NoError(t, err)
			decode := func(data []byte) any {
				d := json.NewDecoder(bytes.NewReader(data))
				d.UseNumber()
				var v any
				require.NoError(t, d.Decode(&v))
				return v
			}
			require.Equal(t, decode([]byte(tc.input)), decode(encoded))
			restored, _, err := array.FromJSON(alloc, tc.dtype, bytes.NewReader(encoded))
			require.NoError(t, err)
			defer restored.Release()
			require.True(t, array.Equal(a, restored))
		})
	}
	for _, input := range []string{`[18446744073709551616]`, `[-1]`, `["not a number"]`, `[1,]`} {
		t.Run(input, func(t *testing.T) {
			a, _, err := array.FromJSON(memory.DefaultAllocator, arrow.PrimitiveTypes.Uint64, strings.NewReader(input))
			if a != nil {
				a.Release()
			}
			require.Error(t, err)
		})
	}
}

func BenchmarkArrowListing(b *testing.B) {
	for _, rows := range []int{1000, 10000} {
		b.Run(fmt.Sprint(rows), func(b *testing.B) {
			md := arrow.NewMetadata([]string{"NextMarker", "NumberOfRecords"}, []string{"next", fmt.Sprint(rows)})
			schema := coreSchema(&md)
			builder := array.NewRecordBuilder(memory.DefaultAllocator, schema)
			for i := 0; i < rows; i++ {
				builder.Field(0).(*array.StringBuilder).Append(fmt.Sprintf("dir/é-雪-%06d.txt", i))
				builder.Field(1).(*array.TimestampBuilder).Append(arrow.Timestamp(1700000000000000))
				builder.Field(2).(*array.TimestampBuilder).Append(arrow.Timestamp(1700000001000000))
				builder.Field(3).(*array.StringBuilder).Append("BlockBlob")
				builder.Field(4).(*array.StringBuilder).Append("etag")
				if i%7 == 0 {
					builder.Field(5).AppendNull()
				} else {
					builder.Field(5).(*array.Uint64Builder).Append(uint64(i) * 1024)
				}
				builder.Field(6).(*array.StringBuilder).Append("application/octet-stream")
				builder.Field(7).(*array.BooleanBuilder).Append(i%2 == 0)
				builder.Field(8).(*array.StringBuilder).Append("Hot")
				builder.Field(9).(*array.StringBuilder).Append("available")
				builder.Field(10).(*array.StringBuilder).Append("unlocked")
			}
			rec := builder.NewRecordBatch()
			builder.Release()
			var buf bytes.Buffer
			writer := ipc.NewWriter(&buf, ipc.WithSchema(schema))
			if err := writer.Write(rec); err != nil {
				b.Fatal(err)
			}
			rec.Release()
			if err := writer.Close(); err != nil {
				b.Fatal(err)
			}
			data := buf.Bytes()
			b.ReportAllocs()
			b.SetBytes(int64(len(data)))
			b.ResetTimer()
			for b.Loop() {
				page, err := HandleFlatListResponse(makeHTTPResponse(data, nil))
				if err != nil {
					b.Fatal(err)
				}
				if len(page.Segment.BlobItems) != rows {
					b.Fatal("lost rows")
				}
			}
		})
	}
}

func BenchmarkArrowJSON(b *testing.B) {
	input := `[` + strings.Repeat(`"é/雪<>&\"",null,`, 999) + `"last"]`
	b.Run("decode", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(input)))
		for b.Loop() {
			a, _, err := array.FromJSON(memory.DefaultAllocator, arrow.BinaryTypes.String, strings.NewReader(input))
			if err != nil {
				b.Fatal(err)
			}
			a.Release()
		}
	})
	b.Run("encode", func(b *testing.B) {
		a, _, err := array.FromJSON(memory.DefaultAllocator, arrow.BinaryTypes.String, strings.NewReader(input))
		if err != nil {
			b.Fatal(err)
		}
		defer a.Release()
		b.ReportAllocs()
		b.SetBytes(int64(len(input)))
		b.ResetTimer()
		for b.Loop() {
			if _, err := a.MarshalJSON(); err != nil {
				b.Fatal(err)
			}
		}
	})
}
