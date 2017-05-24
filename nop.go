package guma

//nopMarshaler is a no operation BinartyMarshaler.
type nopMarshaler struct{}

func (n nopMarshaler) MarshalBinary() ([]byte, error) {
	return nil, nil
}

//nopUnmarshaler is a no operation BinartyUnmarshaler.
type nopUnmarshaler struct{}

func (n nopUnmarshaler) UnmarshalBinary([]byte) error {
	return nil
}
