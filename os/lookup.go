package os

import (
	"fmt"
	"io"
	"os"
	"os/user"
	"bufio"
	"strconv"
	"strings"
	"errors"
)

func LookupIdentity(file string, id int, name string, lookupByName bool) (*Identity, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	br := bufio.NewReader(f)
	loopAndCompare := func(idxCompare int, valueCompare string, idxReturn int) (string, error) {
		for {
			s, err := br.ReadString('\n')
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", err
			}
			p := strings.Split(s, ":")
			if len(p) >= 3 && p[idxCompare] == valueCompare {
				return p[idxReturn], nil
			}
		}
		return "", errors.New("Could not find group")
	}

	if lookupByName {
		value, err := loopAndCompare(0, name, 2); if err != nil {
			return nil, UnknownIdentityError(name)
		}

		id, err = strconv.Atoi(value); if err != nil {
			return nil, errors.New(fmt.Sprintf("Unable to parse group id: %s", value))
		}
	} else {
		value, err := loopAndCompare(2, fmt.Sprintf("%i", id), 1); if err != nil {
			return nil, UnknownIdentityError(name)
		}
		name = value
	}

	i := &Identity{
		Id:   id,
		Name: name,
	}
	return i, nil
}

// Lookup looks up a user by username. If the user cannot be found, the
// returned error is of type UnknownIdentityError.
func LookupUsername(username string) (*Identity, error) {
	i, err := LookupIdentity("/etc/passwd", -1, username, true); if err == nil {
		return i, nil
	}

	u, err := user.Lookup(username); if err == nil {
		return &Identity {
			Id: 0,
			Name: u.Username,
		}, nil
	}

	return nil, err
}

// Lookup looks up a group by groupname. If the group cannot be found, the
// returned error is of type UnknownIdentityError.
func LookupGroupname(groupname string) (*Identity, error) {
	return LookupIdentity("/etc/group", -1, groupname, true)
}
