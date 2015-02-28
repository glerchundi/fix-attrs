package command

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	json "encoding/json"
	yaml "gopkg.in/yaml.v2"

	"github.com/codegangsta/cli"
)

const (
	JSON = "json"
	YAML = "yml"
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
	attrs     attrtuple
}

type attr struct {
	uid, gid string
	perm     string
}

type attrtuple struct {
	dirAttr  attr
	fileAttr attr
}

func NewFixCommand() cli.Command {
	return cli.Command{
		Name:  "fix",
		Usage: "fixes attributes",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "format",
				Value: "",
				Usage: "file format (json, yaml), defaults to json",
			},
			cli.StringFlag{
				Name:  "chown-bin",
				Value: "chown",
				Usage: "chown binary",
			},
			cli.StringFlag{
				Name:  "chmod-bin",
				Value: "chmod",
				Usage: "chmod binary",
			},
		},
		Action: handleFix,
	}
}

func handleFix(c *cli.Context) {
	// error
	var err error

	// params
	format := c.String("format")
	chownBin := c.String("chown-bin")
	chmodBin := c.String("chmod-bin")
	cfgPath := c.Args().First()

	// chown binary path
	chownPath, err := exec.LookPath(chownBin)
	if err != nil {
		log.Fatal("please provide a valid chown binary path")
	}

	// chmod binary path
	chmodPath, err := exec.LookPath(chmodBin)
	if err != nil {
		log.Fatal("please provide a valid chmod binary path")
	}

	// cfg file
	if !fileExists(cfgPath) {
		log.Fatal("please provide a configuration file")
	}

	// format
	if format == "" {
		format = filepath.Ext(cfgPath)
		if format != "" {
			format = format[1:]
		} else {
			format = JSON
		}
	}
	format = strings.ToLower(format)

	// loop parameters
	var m map[string]value

	// uid/gid cache
	uidmap = make(map[string]idOrError)
	gidmap = make(map[string]idOrError)

	// start fixin!
	m = parseFile(cfgPath, format)
	for k, v := range m {
		if !v.recursive {
			var files []string
			if strings.Contains(k, "*") {
				files, err = filepath.Glob(k)
				if err != nil {
					log.Fatal(err.Error())
				}
			} else {
				files = append(files, k)
			}
			for _, f := range files {
				info, err := os.Stat(f)
				if err != nil {
					log.Fatal(fmt.Sprintf("no such file or directory: %s", k))
				}
				err = changeOwnershipAndMode(chownPath, chmodPath, f, info, v)
				if err != nil {
					log.Fatal(err.Error())
				}
			}
		} else {
			walk := func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				err = changeOwnershipAndMode(chownPath, chmodPath, path, info, v)
				if err != nil {
					return err
				}
				return nil
			}
			err = filepath.Walk(k, walk)
			if err != nil {
				log.Fatal(err.Error())
			}
		}
	}
}

func execCommand(binPath string, args ...string) error {
	cmd := exec.Command(binPath, args...)
	err := cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	if err != nil {
		return err
	}
	return nil
}

func changeOwnershipAndMode(chownPath, chmodPath string,
	path string, info os.FileInfo, v value) error {
	var err error
	var attr attr

	if info.IsDir() {
		attr = v.attrs.dirAttr
	} else {
		attr = v.attrs.fileAttr
	}
	err = execCommand(chownPath, attr.uid+":"+attr.gid, path)
	if err != nil {
		return err
	}
	err = execCommand(chmodPath, attr.perm, path)
	if err != nil {
		return err
	}

	return nil
}

func parseFile(f string, format string) map[string]value {
	d, err := ioutil.ReadFile(f)
	if err != nil {
		log.Fatal("unable to open file: ", f)
	}
	return parseContent(d, format)
}

func parseContent(d []byte, format string) map[string]value {
	var i interface{}
	switch format {
	case JSON:
		json.Unmarshal([]byte(d), &i)
	case YAML:
		yaml.Unmarshal([]byte(d), &i)
	default:
		log.Fatal("please provide a valid format")
	}

	if i == nil {
		log.Fatal("unable to parse, no content or invalid format provided.")
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
	pathVal, err := stringval(m, "path")
	if err != nil {
		log.Fatal(err.Error())
	}
	recursive := false
	recursiveVal, err := boolval(m, "recursive")
	if err == nil {
		recursive = recursiveVal
	}

	/*
		TODO: duplicate file/path's?
		_, ok := fm[fullPath]; if ok {
			log.Fatal("duplicate path: ", fullPath)
		}
	*/
	fullPath := path.Join(parentPath, pathVal)
	t, err := attrtupleval(m)
	if err != nil {
		log.Fatal(err.Error())
	}

	fm[fullPath] = value{recursive: recursive, attrs: t}
	if !recursive {
		files, err := arrayval(m, "files")
		if err == nil {
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
	i, ok := m[key]
	if ok {
		return i, nil
	}

	return "", fmt.Errorf("Key not found: %s", key)
}

func stringval(m map[string]interface{}, key string) (string, error) {
	i, err := val(m, key)
	if err != nil {
		return "", err
	}
	s, ok := i.(string)
	if !ok {
		return "", fmt.Errorf("Unable to cast to string: %s", key)
	}

	return s, nil
}

func boolval(m map[string]interface{}, key string) (bool, error) {
	i, err := val(m, key)
	if err != nil {
		return false, err
	}

	bv, ok := i.(bool)
	if !ok {
		return false, fmt.Errorf("Unable to cast to bool: %s", key)
	}
	return bv, nil
}

func attrval(m map[string]interface{}, key string) (attr, error) {
	v, err := stringval(m, key)
	if err != nil {
		return attr{}, err
	}

	parts := strings.Split(v, ":")
	if len(parts) != 3 {
		return attr{}, fmt.Errorf("Unable to parse attributes: %s", key)
	}

	uid  := parts[0]
	gid  := parts[1]
	perm := parts[2]

	return attr{uid: uid, gid: gid, perm: perm}, nil
}

func attrtupleval(m map[string]interface{}) (attrtuple, error) {
	a, err := attrval(m, "attr")
	if err == nil {
		return attrtuple{dirAttr: a, fileAttr: a}, nil
	}
	ad, err := attrval(m, "attr-dir")
	if err != nil {
		return attrtuple{}, err
	}
	af, err := attrval(m, "attr-file")
	if err != nil {
		return attrtuple{}, err
	}
	return attrtuple{dirAttr: ad, fileAttr: af}, nil
}

func arrayval(m map[string]interface{}, key string) ([]interface{}, error) {
	i, err := val(m, key)
	if err != nil {
		return nil, err
	}
	v, ok := i.([]interface{})
	if !ok {
		return nil, fmt.Errorf("Unable to cast to array: %s", key)
	}
	return v, nil
}

func fromYamlMap(m map[interface{}]interface{}) map[string]interface{} {
	r := make(map[string]interface{})
	for k, v := range m {
		ky, ok := k.(string)
		if !ok {
			fmt.Println("key is not of string type")
		}
		_, ok = r[ky]
		if ok {
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
