package os

import (
	"strconv"
)

type Identity struct {
	Id   int
	Name string
}

// UnknownIdentityIdError is returned by LookupId when
// a identity cannot be found.
type UnknownIdentityIdError int

func (e UnknownIdentityIdError) Error() string {
	return "identity: unknown identityid " + strconv.Itoa(int(e))
}

// UnknownIdentityError is returned by Lookup when
// a identity cannot be found.
type UnknownIdentityError string

func (e UnknownIdentityError) Error() string {
	return "identity: unknown identity " + string(e)
}
