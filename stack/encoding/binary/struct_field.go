package binary

import (
	"reflect"
	"strconv"
	"strings"
)

type structField struct {
	Name          string
	Value         reflect.Value
	BitSize       byte   // read from `opcua:"bits=x"`
	SwitchValue   int64  // read from `opcua:"switchValue=x"`
	SwitchField   string // retrieved based on `opcua:"switchField=x"`
	LengthField   string // retrieved based on `opcua:"lengthField=x"`
	SwitchOperand string // retrieved based on `opcua:"switchOperand=x"`
}

func gatherFields(fields []structField, rv reflect.Value) ([]structField, error) {
	var err error

	rt := rv.Type()
	l := rt.NumField()

	for i := 0; i < l; i++ {
		rf := rt.Field(i)
		if rf.PkgPath != "" {
			// Skip unexported field.
		} else if rf.Anonymous {
			// Recursively gather inherited fields.
			fields, err = gatherFields(fields, rv.Field(i))
			if err != nil {
				return nil, err
			}
		} else {
			// Gather information about field.
			field, err := readStructField(rv, rf)
			if err != nil {
				return nil, err
			}
			fields = append(fields, *field)
		}

	}
	return fields, nil
}

func readStructField(parent reflect.Value, rf reflect.StructField) (*structField, error) {
	sf := structField{
		Name:        rf.Name,
		Value:       parent.FieldByIndex(rf.Index),
		SwitchValue: -1,
	}
	tag, ok := rf.Tag.Lookup("opcua")
	if ok && tag != "" {
		if err := readTag(&sf, tag); err != nil {
			return nil, wrapError(err, sf.Name)
		}
	}
	return &sf, nil
}

func readTag(sf *structField, tag string) error {
	const (
		bitsPrefix          = "bits="
		lengthFieldPrefix   = "lengthField="
		switchFieldPrefix   = "switchField="
		switchValuePrefix   = "switchValue="
		switchOperandPrefix = "switchOperand="
	)

	for _, s := range strings.Split(tag, ",") {
		if strings.HasPrefix(s, bitsPrefix) {
			i, err := strconv.Atoi(s[len(bitsPrefix):])
			if err != nil {
				return ErrInvalidTag
			}
			sf.BitSize = byte(i)
		} else if strings.HasPrefix(s, lengthFieldPrefix) {
			sf.LengthField = s[len(lengthFieldPrefix):]
		} else if strings.HasPrefix(s, switchFieldPrefix) {
			sf.SwitchField = s[len(switchFieldPrefix):]
		} else if strings.HasPrefix(s, switchValuePrefix) {
			i, err := strconv.Atoi(s[len(switchValuePrefix):])
			if err != nil {
				return ErrInvalidTag
			}
			sf.SwitchValue = int64(i)
		} else if strings.HasPrefix(s, switchOperandPrefix) {
			sf.SwitchOperand = s[len(switchOperandPrefix):]
		} else {
			return ErrInvalidTag
		}
	}
	return nil
}
