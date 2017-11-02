package binary

import (
	"encoding"
	"fmt"
	"io"
	"reflect"
	"time"
	"unsafe"

	"github.com/searis/guma/stack/uatype"

	"io/ioutil"

	"runtime/debug"
)

// bitExtractor is a helper struct that can be used to unmarshal slices of 1-8
// bits.
type bitExtractor struct {
	Target    *byte
	BitLength byte
}

// BitLengther is implemented by types that should be encoded into, or has been
// decoded from, a certain number of bits.
type BitLengther interface {
	BitLength() int
}

// Unmarshal parses the OPC UA binary encoded data into the value pointed to by
// v.
func Unmarshal(data []byte, v interface{}) error {
	dec := NewDecoder(nil)
	dec.data = data
	return dec.Decode(v)
}

// A Decoder reads and decodes OPC UA Binary content from an input stream.
type Decoder struct {
	r              io.Reader
	data           []byte
	n              int
	bitUnmarshaler bitCacheUnmarshaler
}

// NewDecoder initializes a Decoder for r.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

// BytesRead return the number of bytes read since initialization.
func (dec Decoder) BytesRead() int {
	return dec.n
}

// Decode reads the next OPC UA binary encoded value from it's input stream and
// stores it in the value pointed to by v.
func (dec *Decoder) Decode(v interface{}) (err error) {
	defer func() {
		if e := recover(); e != nil {
			debugLogger.Printf("recovered from panic: %s:\n%s", e, debug.Stack())
			err = fmt.Errorf("recovered from panic: %s", e)
		}
	}()

	// Check if dec.r is (still) set to allow multiple calls to Decode, as well
	// as an optimized path for the Unmarshal function.
	if dec.r != nil {
		// FIXME: As an initial implementation we read all binary data directly
		// into memory without any steaming. We should improve on this later
		// to prevent consuming data that we do not need.
		if dec.data, err = ioutil.ReadAll(dec.r); err != nil {
			return err
		}

		// get rid of the reader to symbolize that it's been exhausted.
		dec.r = nil
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return ErrNotSetable
	}

	switch err := dec.decode(rv).(type) {
	case transcoderError:
		typeName := rv.Type().Name()
		return DecoderError{err, typeName}
	default:
		return err
	}
}

func (dec *Decoder) decode(rv reflect.Value) error {
	var u encoding.BinaryUnmarshaler
	var size, maxSize int
	var data []byte

	// Pick binary marshaler.
	switch iv := rv.Interface().(type) {
	case *[]byte:
		size = len(*iv)
		if size == 0 {
			u = nopUnmarshaler{}
		} else {
			u = (*byteSlice)(iv)
		}
	case *bool, *uint8, *int8:
		u = byteUnmarshaler{iv}
		size = 1
	case *uint16, *int16:
		u = byteUnmarshaler{iv}
		size = 2
	case *int32, *uint32, *float32:
		u = byteUnmarshaler{iv}
		size = 4
	case *int64, *uint64, *float64:
		u = byteUnmarshaler{iv}
		size = 8
	case *[]bool, *[]int8:
		size = rv.Elem().Len()
		if size == 0 {
			u = nopUnmarshaler{}
		} else {
			u = byteUnmarshaler{iv}
		}
	case *[]uint16, *[]int16:
		size = rv.Elem().Len() * 2
		if size == 0 {
			u = nopUnmarshaler{}
		} else {
			u = byteUnmarshaler{iv}
		}
	case *[]int32, *[]uint32, *[]float32:
		size = rv.Elem().Len() * 4
		if size == 0 {
			u = nopUnmarshaler{}
		} else {
			u = byteUnmarshaler{iv}
		}
	case *[]int64, *[]uint64, *[]float64:
		size = rv.Elem().Len() * 8
		if size == 0 {
			u = nopUnmarshaler{}
		} else {
			u = byteUnmarshaler{iv}
		}
	case *time.Time:
		u = (*dateTime)(iv)
		size = 8
	case *time.Duration:
		u = (*duration)(iv)
	case *uatype.Bit:
		dec.bitUnmarshaler.SetBoolTarget((*bool)(iv))
		u = &dec.bitUnmarshaler
		maxSize = 1
	case *string:
		u = (*uaString)(iv)
	case bitExtractor:
		if err := dec.bitUnmarshaler.SetTarget(iv.Target, iv.BitLength); err != nil {
			return err
		}
		u = &dec.bitUnmarshaler
		maxSize = 1
	case encoding.BinaryUnmarshaler:
		// Prefer BinaryUnmarshaler over BitLengther, if implemented.
		u = iv
	case BitLengther:
		nBits := iv.BitLength()
		if nBits < 8 {
			if rv.Elem().Kind() != reflect.Uint8 {
				return fmt.Errorf("Not uint8")
			}
			if err := dec.bitUnmarshaler.SetTarget((*byte)(unsafe.Pointer(rv.Pointer())), byte(nBits)); err != nil {
				return err
			}
			u = &dec.bitUnmarshaler
			maxSize = 1
		} else if nBits%8 != 0 {
			return fmt.Errorf("bit length above 8 must be aligned to 8 bits; bit length was %d", nBits)
		} else {
			size = nBits / 8
			u = byteUnmarshaler{iv}
		}
	default:
		re := rv.Elem()
		switch re.Kind() {
		case reflect.Array, reflect.Slice:
			u, size = listUnmarshaler(re)
			if u == nil {
				return dec.decodeList(re)
			}
		case reflect.Struct:
			return dec.decodeStruct(re)
		case reflect.Int8, reflect.Uint8:
			u = byteUnmarshaler{re}
			size = 1
		case reflect.Int16, reflect.Uint16:
			u = byteUnmarshaler{re}
			size = 2
		case reflect.Int32, reflect.Uint32:
			u = byteUnmarshaler{iv}
			size = 4
		case reflect.Int64, reflect.Uint64:
			u = byteUnmarshaler{re}
			size = 8

		default:
			return ErrUnknownType
		}

	}

	// Limit input data if the (max) size is known.
	if size != 0 {
		if size > len(dec.data) {
			return io.ErrShortBuffer
		}
		data = dec.data[:size]
	} else if maxSize != 0 && len(dec.data) > maxSize {
		data = dec.data[:maxSize]
	} else {
		data = dec.data
	}
	// Unmarshal data.
	if err := u.UnmarshalBinary(data); err != nil {
		return err
	}

	// Remove consumed data.
	if size == 0 {
		size = marshaledSize(u)
	}
	dec.data = dec.data[size:]

	return nil
}

func (dec *Decoder) decodeList(rv reflect.Value) error {
	l := rv.Len()
	for i := 0; i < l; i++ {
		if err := dec.decode(rv.Index(i).Addr()); err != nil {
			return wrapError(err, i)
		}
	}
	return nil
}

func (dec *Decoder) decodeStruct(rv reflect.Value) error {
	fields, err := gatherFields(nil, rv)
	if err != nil {
		return err
	}

	prevIndices := map[string]int{}

	for i, f := range fields {

		// Continue if switch field value does not match.
		if f.SwitchValue != -1 {
			if f.SwitchField == "" {
				return wrapError(ErrInvalidTag, f.Name)
			}
			var v int64
			switch fields[prevIndices[f.SwitchField]].Value.Kind() {
			case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
				v = fields[prevIndices[f.SwitchField]].Value.Int()
			case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
				v = int64(fields[prevIndices[f.SwitchField]].Value.Uint())
			default:
				return wrapError(ErrInvalidTag, fmt.Errorf("Invalid SwitchValue [%s] for field [%s]", fields[prevIndices[f.SwitchField]].Value.Kind(), f.Name))
			}

			if f.SwitchOperand != "" {
				switch f.SwitchOperand {
				case "Equals":
					if v == f.SwitchValue {
						goto process
					}
				case "GreaterThan":
					if v > f.SwitchValue {
						goto process
					}
				case "LessThan":
					if v < f.SwitchValue {
						goto process
					}
				case "GreaterThanOrEqual":
					if v >= f.SwitchValue {
						goto process
					}
				case "LessThanOrEqual":
					if v <= f.SwitchValue {
						goto process
					}
				case "NotEqual":
					if v != f.SwitchValue {
						goto process
					}
				default:
					return wrapError(ErrInvalidTag, fmt.Errorf("Invalid SwithcOperand [%s] for field [%s]", f.SwitchOperand, f.Name))
				}
				continue
			}
			// There might be switchValiues without any defined Operator, this means that we should
			// continue if the value and field does not match
			if v != f.SwitchValue {
				continue
			}
		} else if f.SwitchField != "" {
			if !fields[prevIndices[f.SwitchField]].Value.Bool() {
				continue
			}
		}
	process:
		// Allocate space for slices.
		if f.LengthField != "" {
			l := int(fields[prevIndices[f.LengthField]].Value.Int())

			et := f.Value.Type().Elem()

			f.Value.Set(reflect.MakeSlice(reflect.SliceOf(et), l, l))
		}

		// Wrap or reference field value before decoding.
		var decodeValue reflect.Value
		if f.BitSize > 0 && f.BitSize <= 8 {
			decodeValue = reflect.ValueOf(bitExtractor{
				Target:    f.Value.Addr().Interface().(*byte),
				BitLength: f.BitSize,
			})
		} else if f.BitSize > 8 {
			return wrapError(ErrInvalidBitLength, f.Name)
		} else if f.Value.Kind() == reflect.Ptr {
			f.Value.Set(reflect.New(f.Value.Type().Elem()))
			decodeValue = f.Value
		} else {
			decodeValue = f.Value.Addr()
		}

		// Decode value.
		err := dec.decode(decodeValue)
		if err != nil {
			return wrapError(err, f.Name)
		}
		prevIndices[f.Name] = i
	}
	return nil

}

// listUnmarshaler is intended to return a marshaler that would run slightly
// faster than decodeList. When no optimization is found, nil is returned.
func listUnmarshaler(rv reflect.Value) (encoding.BinaryUnmarshaler, int) {
	// TODO: Prove this optimizations with micro-benchmarks.
	l := rv.Len()
	if l == 0 {
		return nopUnmarshaler{}, 0
	}
	switch rv.Index(0).Interface().(type) {
	case bool, uint8, int8:
		return byteUnmarshaler{rv.Addr().Interface()}, l
	case int16, uint16:
		return byteUnmarshaler{rv.Addr().Interface()}, 2 * l
	case int32, uint32, float32:
		return byteUnmarshaler{rv.Addr().Interface()}, 4 * l
	case int64, uint64, float64:
		return byteUnmarshaler{rv.Addr().Interface()}, 8 * l
	}
	return nil, 0
}

// marshaledSize returns the binary encoding size of v. In vase v is a pointer,
// the encoding size will be calculated for the value pointed to by v.
func marshaledSize(v interface{}) int {

	if v == nil {
		return 0
	}

	// Convert string to uaString to make it implement BitLengther.
	if s, ok := v.(string); ok {
		v = uaString(s)
	}

	// Get size for any BitLengther implementations.
	if bl, ok := v.(BitLengther); ok {
		return bl.BitLength() / 8
	}

	// If not a BitLengther, dereferenced value and reflect value.
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
		v = rv.Interface()
	}

	// Determin size for Array and Slice values.
	switch rv.Type().Kind() {
	case reflect.Array, reflect.Slice:
		l := rv.Len()
		if l == 0 {
			return 0
		}
		switch rv.Index(0).Interface().(type) {
		case bool, uint8, int8:
			return 1 * l
		case int16, uint16:
			return 2 * l
		case int32, uint32, float32:
			return 4 * l
		case int64, uint64, float64, time.Time:
			return 8 * l
		default:
			var size int
			for i := 0; i < l; i++ {
				size += marshaledSize(rv.Index(i))
			}
			return size
		}
	}

	panic(fmt.Sprintf("Can't find size for type: %s", rv.Type().Name()))
}
