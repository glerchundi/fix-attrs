package os

type Identity struct {
	ID   int
	Name string
}

// UnknownIdentityError is returned by Lookup when
// a identity cannot be found.
type UnknownIdentityError string

func (e UnknownIdentityError) Error() string {
	return "identity: unknown identity " + string(e)
}
