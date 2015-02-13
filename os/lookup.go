package os

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"strconv"
	"strings"
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
		value, err := loopAndCompare(0, name, 2)
		if err != nil {
			return nil, UnknownIdentityError(name)
		}

		id, err = strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("Unable to parse group id: %s", value)
		}
	} else {
		value, err := loopAndCompare(2, fmt.Sprintf("%i", id), 1)
		if err != nil {
			return nil, UnknownIdentityError(name)
		}
		name = value
	}

	i := &Identity{
		ID:   id,
		Name: name,
	}
	return i, nil
}

func LookupUsername(username string) (*Identity, error) {
	i, err := LookupIdentity("/etc/passwd", -1, username, true)
	if err == nil {
		return i, nil
	}

	u, err := user.Lookup(username)
	if err == nil {
		return &Identity{
			ID:   0,
			Name: u.Username,
		}, nil
	}

	return nil, err
}

func LookupGroupname(groupname string) (*Identity, error) {
	return LookupIdentity("/etc/group", -1, groupname, true)
}
