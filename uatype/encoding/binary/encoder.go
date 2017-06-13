package binary

import (
	"encoding"
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
	"runtime/debug"
	"time"

	"github.com/searis/guma/uatype"
)

// bitSlice is a helper struct that can be used to marshal slices of 1-8 bits.
type bitSlice struct {
	Data      byte
	BitLength byte
}

// A Encoder writes OPC UA Binary content to an output stream.
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
	case bool:
		enc.byteMarshaler.SetData(iv)
		m = &enc.byteMarshaler
	case uint8, int8, uint16, int16, int32, uint32, int64, uint64, float32, float64:
		enc.byteMarshaler.SetData(iv)
		m = &enc.byteMarshaler
	case string:
		m = uaString(iv)
	case time.Time:
		m = dateTime(iv)
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
			if err := enc.bitMarshaler.SetBits((interface{})(iv).(byte), byte(nBits)); err != nil {
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
		case reflect.Slice, reflect.Array:
			m = listMarshaler(&enc.byteMarshaler, rv)
			if m == nil {
				return enc.encodeList(rv)
			}
		case reflect.Struct:
			return enc.encodeStruct(rv)
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
		if f.SwitchValue != 0 {
			if f.SwitchField == "" {
				return wrapError(ErrInvalidTag, f.Name)
			}
			v := fields[prevIndices[f.SwitchField]].Value.Int()
			if v != f.SwitchValue {
				continue
			}
		} else if f.SwitchField != "" {
			if !fields[prevIndices[f.SwitchField]].Value.Bool() {
				continue
			}
		}

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
	switch rv.Index(0).Interface().(type) {
	case bool, int8, uint8, int16, uint16, int32, uint32, float32, int64, uint64, float64:
		bm.SetData(rv.Interface())
		return bm
	}
	return nil
}
