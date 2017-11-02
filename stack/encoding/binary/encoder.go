package binary

import (
	"bytes"
	"encoding"
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
	"runtime/debug"
	"time"

	"github.com/searis/guma/stack/uatype"
)

// bitSlice is a helper struct that can be used to marshal slices of 1-8 bits.
type bitSlice struct {
	Data      byte
	BitLength byte
}

// Marshal encodes v into the OPC UA Binary Encoding format, and returns it as
// a slice of bytes.
func Marshal(v interface{}) ([]byte, error) {
	var buff bytes.Buffer
	enc := NewEncoder(&buff)
	err := enc.Encode(v)
	return buff.Bytes(), err

}

// An Encoder writes OPC UA Binary content to an output stream.
type Encoder struct {
	w             io.Writer
	n             int64
	bitMarshaler  bitCacheMarshaler
	byteMarshaler byteMarshaler
}

// NewEncoder takes a writer object where OPC UA data will be written on calls
// to Encode.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

// BytesWritten returns the number of bytes written since the BinaryEncoder was
// initialized.
func (enc *Encoder) BytesWritten() int64 {
	return enc.n
}

// Encode encodes uatype structs into a binary representation used for transfer.
func (enc *Encoder) Encode(v interface{}) (err error) {
	defer func() {
		if e := recover(); e != nil {
			debugLogger.Printf("recovered from panic: %s:\n%s", e, debug.Stack())
			err = fmt.Errorf("recovered from panic: %s", e)
		}
	}()

	rv := reflect.ValueOf(v)

	switch err := enc.encode(rv).(type) {
	case transcoderError:
		typeName := rv.Type().Name()
		return EncoderError{err, typeName}
	default:
		return err
	}
}

func (enc *Encoder) encode(rv reflect.Value) error {
	var err error
	var m encoding.BinaryMarshaler

	// Pick binary marshaler.
	switch iv := rv.Interface().(type) {
	case []byte:
		if len(iv) > 0 {
			m = (byteSlice)(iv)
		} else {
			m = nopMarshaler{}
		}
	case bool:
		enc.byteMarshaler.SetData(iv)
		m = &enc.byteMarshaler
	case uint8, int8, uint16, int16, int32, uint32, int64, uint64, float32, float64:
		enc.byteMarshaler.SetData(iv)
		m = &enc.byteMarshaler
	case []bool, []int8, []uint16, []int16, []int32, []uint32, []int64, []uint64, []float32, []float64:
		if rv.Len() > 0 {
			enc.byteMarshaler.SetData(iv)
			m = &enc.byteMarshaler
		} else {
			m = nopMarshaler{}
		}
	case string:
		m = uaString(iv)
	case time.Time:
		m = dateTime(iv)
	case time.Duration:
		m = duration(iv)
	case uatype.Bit:
		var err error
		if iv {
			err = enc.bitMarshaler.SetBits(1, 1)
		} else {
			err = enc.bitMarshaler.SetBits(0, 1)
		}
		if err != nil {
			return err
		}
		m = &enc.bitMarshaler
	case bitSlice:
		if err := enc.bitMarshaler.SetBits(iv.Data, byte(iv.BitLength)); err != nil {
			return err
		}
		m = &enc.bitMarshaler
	case encoding.BinaryMarshaler:
		// Prefer BinaryMarshaler over BitLengther, if implemented.
		m = iv
	case BitLengther:
		nBits := iv.BitLength()
		if nBits < 8 {
			// If the underlying type is not a byte, we wil panic.
			if rv.Kind() != reflect.Uint8 {
				return fmt.Errorf("bit lenght below 8 must have an underlying byte type, type was %s", rv.Type().Name())
			}
			if err := enc.bitMarshaler.SetBits(uint8(rv.Uint()), byte(nBits)); err != nil {
				return err
			}
			m = &enc.bitMarshaler
		} else if nBits%8 != 0 {
			return fmt.Errorf("bit length above 8 must be aligned to 8 bits; bit length was %d", nBits)
		} else {
			enc.byteMarshaler.SetData(iv)
			enc.byteMarshaler.SetSlice(0, uint(nBits/8))
			m = &enc.byteMarshaler
		}
	default:
		switch rv.Kind() {
		case reflect.Array, reflect.Slice:
			m = listMarshaler(&enc.byteMarshaler, rv)
			if m == nil {
				return enc.encodeList(rv)
			}
		case reflect.Struct:
			return enc.encodeStruct(rv)
		case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
			enc.byteMarshaler.SetData(iv)
			m = &enc.byteMarshaler
		default:
			return ErrUnknownType
		}
	}

	data, err := m.MarshalBinary()
	enc.n += int64(len(data))
	if err != nil || len(data) == 0 {
		return err
	}

	err = binary.Write(enc.w, binary.LittleEndian, data)
	if err != nil {
		return err
	}
	return nil
}
func (enc *Encoder) encodeList(rv reflect.Value) error {
	l := rv.Len()
	for i := 0; i < l; i++ {
		if err := enc.encode(rv.Index(i)); err != nil {
			return wrapError(err, i)
		}
	}
	return nil
}

func (enc *Encoder) encodeStruct(rv reflect.Value) error {
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

		// Assert that length field is set correctly.
		if f.LengthField != "" {
			e := int(fields[prevIndices[f.LengthField]].Value.Int())
			l := f.Value.Len()
			if l != e {
				debugLogger.Printf("length (%d) != expected length (%d)", l, e)
				return wrapError(ErrInvalidLength, f.Name)
			}
		}

		// Get/wrap value to encode.
		var re reflect.Value
		if f.BitSize > 0 && f.BitSize <= 8 {
			re = reflect.ValueOf(bitSlice{
				Data:      f.Value.Interface().(byte),
				BitLength: f.BitSize,
			})
		} else if f.BitSize > 8 {
			return wrapError(ErrInvalidBitLength, f.Name)
		} else {
			re = f.Value
		}

		// Encode value.
		err := enc.encode(re)
		if err != nil {
			return wrapError(err, f.Name)
		}
		prevIndices[f.Name] = i
	}
	return nil
}

// listMarshaler is intended to return a marshaler that would run slightly
// faster than encodeList. When no optimization is found, nil is returned.
func listMarshaler(bm *byteMarshaler, rv reflect.Value) encoding.BinaryMarshaler {
	// TODO: Prove this optimizations with micro-benchmarks.
	l := rv.Len()
	if l == 0 {
		return nil
	}
	switch rv.Index(0).Interface().(type) {
	case bool, uint8, int8, int16, uint16, int32, uint32, float32, int64, uint64, float64:
		bm.SetData(rv.Interface())
		return bm
	}
	return nil
}
