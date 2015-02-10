package command

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"errors"
	"path"
	"path/filepath"
	"strings"
	"strconv"

	json "encoding/json"
	yaml "gopkg.in/yaml.v2"

	"github.com/codegangsta/cli"
	"github.com/glerchundi/fix-attrs/os/group"
)

const (
	Json    = "json"
	Yaml    = "yml"
)

type idOrError struct {
	id  int
	err error
}
var uidmap map[string]idOrError
var gidmap map[string]idOrError

type key string
type value struct {
	recursive bool
	dirAttr attr
	fileAttr attr
}

type attr struct {
	uid, gid int
	perm os.FileMode
}

func NewFixCommand() cli.Command {
	return cli.Command{
		Name:   "fix",
		Usage:  "fixes attributes",
		Flags:  []cli.Flag{
			cli.StringFlag{
				Name:  "format",
				Value: "",
				Usage: "file format (json, yaml), defaults to json",
			},
		},
		Action: handleFix,
	}
}

func handleFix(c *cli.Context) {
	// params
	format := c.String("format")
	cfgPath := c.Args().First()

	// cfg file
	if !fileExists(cfgPath) {
		log.Fatal("please provide a configuration file")
	}

	// format
	if (format == "") {
		format = filepath.Ext(cfgPath)
		if (format != "") {
			format = format[1:]
		} else {
			format = Json
		}
	}
	format = strings.ToLower(format)

	// loop parameters
	var err error
	var m map[string]value

	// uid/gid cache
	uidmap = make(map[string]idOrError)
	gidmap = make(map[string]idOrError)

	// start fixin!
	m = parseFile(cfgPath, format)
	for k, v := range m {
		if (v.recursive) {
			walk := func(path string, info os.FileInfo, err error) error {
				if (err != nil) {
					return err
				}
				err = changeOwnershipAndMode(path, info, v); if err != nil {
					return err
				}
				return nil
			}
			err = filepath.Walk(k, walk); if err != nil {
				log.Fatal(err.Error())
			}
		} else {
			info, err := os.Stat(k); if err != nil {
				log.Fatal(fmt.Sprintf("no such file or directory: %s", k))
			}
			err = changeOwnershipAndMode(k, info, v); if err != nil {
				log.Fatal(err.Error())
			}
		}
	}
}

func changeOwnershipAndMode(path string, info os.FileInfo, v value) error {
	var err error
	var attr attr

	if info.IsDir() {
		attr = v.dirAttr
	} else {
		attr = v.fileAttr
	}
	err = os.Chown(path, attr.uid, attr.gid); if err != nil {
		return err
	}
	err = os.Chmod(path, attr.perm); if err != nil {
		return err
	}

	return nil
}

func parseFile(f string, format string) map[string]value {
	d, err := ioutil.ReadFile(f); if err != nil {
		log.Fatal("unable to open file: ", f)
	}
	return parseContent(d, format)
}

func parseContent(d []byte, format string) map[string]value {
	var i interface{}
	switch format {
	case Json:
		json.Unmarshal([]byte(d), &i)
	case Yaml:
		yaml.Unmarshal([]byte(d), &i)
	default:
		log.Fatal("please provide a valid format")
	}

	fm := make(map[string]value)
	iterRoot(i, fm)
	return fm
}

func iterRoot(m interface{}, fm map[string]value) {
	switch mm := m.(type) {
	case []interface{}:
		for _, v := range mm {
			iterRoot(v, fm)
		}
	case map[string]interface{}, map[interface{}]interface{}:
		iterFile("", prepareFile(mm), fm)
	default:
		fmt.Println(mm)
	}
}

func iterFile(parentPath string, m map[string]interface{}, fm map[string]value) {
	pathVal, err := stringval(m, "path"); if err != nil {
		log.Fatal(err.Error())
	}
	recursive := false
	recursiveVal, err := boolval(m, "recursive"); if err == nil {
		recursive = recursiveVal
	}

	fullPath := path.Join(parentPath, pathVal)
	if recursive {
		attrdirval, err := attrval(m, "attr-dir"); if err != nil {
			log.Fatal(err.Error())
		}
		attrfileval, err := attrval(m, "attr-file"); if err != nil {
			log.Fatal(err.Error())
		}
		fm[fullPath] = value { recursive: true, dirAttr: attrdirval, fileAttr: attrfileval }
	} else {
		/*
		TODO: duplicate file/path's?
		_, ok := fm[fullPath]; if ok {
			log.Fatal("duplicate path: ", fullPath)
		}
		*/
		attrval, err := attrval(m, "attr"); if err != nil {
			log.Fatal(err.Error())
		}
		fm[fullPath] = value { recursive: false, dirAttr: attrval, fileAttr: attrval }
		files, err := arrayval(m, "files"); if err == nil {
			for _, i := range files {
				iterFile(fullPath, prepareFile(i), fm)
			}
		}
	}
}

func prepareFile(i interface{}) map[string]interface{} {
	switch ii := i.(type) {
	case map[string]interface{}:
		return ii
	case map[interface{}]interface{}:
		return fromYamlMap(ii)
	default:
		fmt.Println("unsupported file type")
	}
	return nil
}

func val(m map[string]interface{}, key string) (interface{}, error) {
	i, ok := m[key]; if ok {
		return i, nil
	}

	return "", errors.New(fmt.Sprintf("Key not found: %s", key))
}

func stringval(m map[string]interface{}, key string) (string, error) {
	i, err := val(m, key); if err != nil {
		return "", err
	}
	s, ok := i.(string); if !ok {
		return "", errors.New(fmt.Sprintf("Unable to cast to string: %s", key))
	}

	return s, nil
}

func boolval(m map[string]interface{}, key string) (bool, error) {
	i, err := val(m, key); if err != nil {
		return false, err
	}

	bv, ok := i.(bool); if !ok {
		return false, errors.New(fmt.Sprintf("Unable to cast to bool: %s", key))
	}
	return bv, nil
}

// Disambiguating user names and IDs
// http://www.gnu.org/software/coreutils/manual/html_node/Disambiguating-names-and-IDs.html
// Since the user and group arguments to these commands may be specified as names or numeric IDs, there is an apparent
// ambiguity. What if a user or group name is a string of digits? 1 Should the command interpret it as a user name or
// as an ID? POSIX requires that these commands first attempt to resolve the specified string as a name, and only once
// that fails, then try to interpret it as an ID. This is troublesome when you want to specify a numeric ID, say 42, and
// it must work even in a pathological situation where ‘42’ is a user name that maps to some other user ID, say 1000.
// Simply invoking chown 42 F, will set Fs owner ID to 1000—not what you intended.

// GNU chown, chgrp, chroot, and id provide a way to work around this, that at the same time may result in a significant
// performance improvement by eliminating a database look-up. Simply precede each numeric user ID and/or group ID with
// a ‘+’, in order to force its interpretation as an integer:
// chown +42 F
// chgrp +$numeric_group_id another-file
// chown +0:+0 /
// The name look-up process is skipped for each ‘+’-prefixed string, because a string containing ‘+’ is never a valid
// user or group name. This syntax is accepted on most common Unix systems, but not on Solaris 10.

func uidval(k string) (int, error) {
	v, ok := uidmap[k]; if ok {
		return v.id, v.err
	}
	id, err := uidval_(k)
	uidmap[k] = idOrError { id: id, err: err }
	return id, err
}

func uidval_(v string) (int, error) {
	if (!strings.HasPrefix(v, "+")) {
		u, err := user.Lookup(v); if err == nil {
			v = u.Uid
		}
	} else {
		v = v[1:]
	}

	return strconv.Atoi(v)
}

func gidval(k string) (int, error) {
	v, ok := gidmap[k]; if ok {
		return v.id, v.err
	}
	id, err := gidval_(k)
	gidmap[k] = idOrError { id: id, err: err }
	return id, err
}

func gidval_(v string) (int, error) {
	if (!strings.HasPrefix(v, "+")) {
		g, err := group.Lookup(v); if err == nil {
			return g.Gid, nil
		}
	} else {
		v = v[1:]
	}

	return strconv.Atoi(v)
}

func permval(v string) (os.FileMode, error) {
	p, err := strconv.ParseUint(v, 8, 32); if err != nil {
		return os.FileMode(0), err
	}
	return os.FileMode(p), nil
}

func attrval(m map[string]interface{}, key string) (attr, error) {
	v, err := stringval(m, key); if err != nil {
		return attr{}, err
	}
	parts := strings.Split(v, ":")
	if (len(parts) != 3) {
		return attr{}, errors.New(fmt.Sprintf("Unable to parse attributes: %s", key))
	}
	uid, err := uidval(parts[0]); if err != nil {
		return attr{}, err
	}
	gid, err := gidval(parts[1]); if err != nil {
		return attr{}, err
	}
	perm, err := permval(parts[2]); if err != nil {
		return attr{}, err
	}

	return attr { uid: uid, gid: gid, perm: perm }, nil
}

func arrayval(m map[string]interface{}, key string) ([]interface{}, error) {
	i, err := val(m, key); if err != nil {
		return nil, err
	}
	v, ok := i.([]interface {}); if !ok {
		return nil, errors.New(fmt.Sprintf("Unable to cast to array: %s", key))
	}
	return v, nil
}

func fromYamlMap(m map[interface{}]interface{}) map[string]interface{} {
	r := make(map[string]interface{})
	for k, v := range m {
		ky, ok := k.(string); if !ok {
			fmt.Println("key is not of string type")
		}
		_, ok = r[ky]; if ok {
			fmt.Println("key already exists")
		}
		r[ky] = v
	}
	return r
}

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}
