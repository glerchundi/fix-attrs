// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package group

import (
	"fmt"
	"io"
	"os"
	"bufio"
	"syscall"
	"strconv"
	"strings"
	"errors"
)

// Current returns the current user.
func Current() (*Group, error) {
	return lookupUnix(syscall.Getgid(), "", false)
}

// Lookup looks up a group by groupname. If the group cannot be found, the
// returned error is of type UnknownGroupError.
func Lookup(username string) (*Group, error) {
	return lookupUnix(-1, username, true)
}

// LookupId looks up a group by groupid. If the group cannot be found, the
// returned error is of type UnknownGroupIdError.
func LookupId(uid string) (*Group, error) {
	i, e := strconv.Atoi(uid)
	if e != nil {
		return nil, e
	}
	return lookupUnix(i, "", false)
}

func lookupUnix(gid int, groupname string, lookupByName bool) (*Group, error) {
	f, err := os.Open("/etc/group")
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

	var id int
	var name string
	if lookupByName {
		value, err := loopAndCompare(0, groupname, 2); if err != nil {
			return nil, UnknownGroupError(groupname)
		}
		id, err = strconv.Atoi(value); if err != nil {
			return nil, errors.New(fmt.Sprintf("Unable to parse group id: %s", value))
		}
		name = groupname
	} else {
		value, err := loopAndCompare(2, fmt.Sprintf("%i", gid), 1); if err != nil {
			return nil, UnknownGroupError(groupname)
		}
		id = gid
		name = value
		return nil, UnknownGroupIdError(gid)
	}

	g := &Group{
		Gid:       id,
		Groupname: name,
	}
	return g, nil
}
