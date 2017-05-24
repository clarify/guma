package guma

import (
	"encoding"
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
	"time"

	"runtime/debug"

	"github.com/searis/guma/uatype"
)

// A BinaryEncoder writes OPC UA Binary content to an output stream.
type BinaryEncoder struct {
	w             io.Writer
	n             int64
	bitMarshaler  bitCacheMarshaler
	byteMarshaler byteMarshaler
}

// NewBinaryEncoder takes a writer object where OPC UA data will be written on
// calls to Encode.
func NewBinaryEncoder(w io.Writer) *BinaryEncoder {
	return &BinaryEncoder{w: w}
}

// BytesWritten returns the number of bytes written since the BinaryEncoder was
// initialized.
func (enc *BinaryEncoder) BytesWritten() int64 {
	return enc.n
}

// Encode encodes uatype structs into a binary representation used for transfer.
func (enc *BinaryEncoder) Encode(v interface{}) (err error) {
	defer func() {
		if e := recover(); e != nil {
			debugLogger.Printf("recovered from panic: %s:\n%s", e, debug.Stack())
			err = fmt.Errorf("recovered from panic: %s", e)
		}
	}()
	switch err := enc.encode(v).(type) {
	case transcoderError:
		typeName := reflect.TypeOf(v).Name()
		return EncoderError{err, typeName}
	default:
		return err
	}
}

func (enc *BinaryEncoder) encode(v interface{}) error {
	var err error
	var m encoding.BinaryMarshaler

	// Pick binary marshaler.
	switch iv := v.(type) {
	case int, uint:
		// reject integers without a specified bit size.
		return ErrUnknownType
	case bool:
		enc.byteMarshaler.SetData(iv)
		m = &enc.byteMarshaler
	case uint8, int8, uint16, int16, int32, uint32, int64, uint64, float32, float64:
		enc.byteMarshaler.SetData(v)
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
	case uatype.BitLengther:
		nBits := iv.BitLength()
		if nBits < 8 {
			// If the underlying type is not a byte, we wil panic.
			if err := enc.bitMarshaler.SetBits(v.(byte), byte(nBits)); err != nil {
				return err
			}
			m = &enc.bitMarshaler
		} else if nBits%8 != 0 {
			return fmt.Errorf("bit length above 8 must be aligned to 8 bits; bit length was %d", nBits)
		} else {
			enc.byteMarshaler.SetData(v)
			enc.byteMarshaler.SetSlice(0, uint(nBits/8))
			m = &enc.byteMarshaler
		}
	default:
		rv := reflect.ValueOf(v)
		m = enc.reflectMarshaler(rv)
		if m == nil {
			return enc.encodeStruct(rv)
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

// reflectMarshaler attempts to find a suitable marshaler via refelection.
// Nested types such as structs do not have marshalers.
func (enc *BinaryEncoder) reflectMarshaler(rv reflect.Value) encoding.BinaryMarshaler {
	switch rv.Kind() {
	case reflect.Array:
		enc.byteMarshaler.SetData(rv.Interface())
		return &enc.byteMarshaler
	case reflect.Slice:
		// Length field should already have been encoded.
		enc.byteMarshaler.SetData(rv.Interface())
		return &enc.byteMarshaler
	}

	return nil
}

// encodeStruct calls encode on each field that's not marked for exclusion.
func (enc *BinaryEncoder) encodeStruct(rv reflect.Value) error {
	if rv.Kind() != reflect.Struct {
		return ErrUnknownType
	}

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

		var v interface{}

		// Wrap with bitSlice if needed.
		if f.BitSize > 0 && f.BitSize <= 8 {
			v = bitSlice{
				Data:      f.Value.Interface().(byte),
				BitLength: f.BitSize,
			}
		} else if f.BitSize > 8 {
			return wrapError(ErrInvalidBitLength, f.Name)
		} else {
			v = f.Value.Interface()
		}

		// Encode value.
		err := enc.encode(v)
		if err != nil {
			return wrapError(err, f.Name)
		}
		prevIndices[f.Name] = i
	}
	return nil
}

// bitSlice is a helper struct that can be used to marshal slices of 1-8 bits.
type bitSlice struct {
	Data      byte
	BitLength byte
}
