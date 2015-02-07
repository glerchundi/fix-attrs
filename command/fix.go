package command

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"strconv"

	json "encoding/json"
	yaml "gopkg.in/yaml.v2"

	"github.com/codegangsta/cli"
)

const (
	Json    = "json"
	Yaml    = "yml"
)

type fileAttr struct {
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
				Usage: "file format (json, yaml)",
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
	if fileNotExists(cfgPath) {
		log.Fatal("please provide a configuration file")
	}

	// format
	if (format == "") {
		format = filepath.Ext(cfgPath)
		format = format[1:]
	}
	format = strings.ToLower(format)

	// start fixin!
	var err error = nil
	var m map[string]fileAttr = nil

	m = parseFile(cfgPath, format)
	for k, v := range m {
		if fileNotExists(k) {
			log.Fatal(fmt.Sprintf("no such file or directory: %s", k))
		}
		err = os.Chown(k, v.uid, v.gid); if err != nil {
			log.Fatal("unable to set ownership to: ", k)
		}
		err = os.Chmod(k, v.perm); if err != nil {
			log.Fatal("unable to set permissions to: ", k)
		}
	}
}

func parseFile(f string, format string) map[string]fileAttr {
	d, err := ioutil.ReadFile(f); if err != nil {
		log.Fatal("unable to open file: ", f)
	}
	return parseContent(d, format)
}

func parseContent(d []byte, format string) map[string]fileAttr {
	var i interface{}
	switch format {
	case Json:
		json.Unmarshal([]byte(d), &i)
	case Yaml:
		yaml.Unmarshal([]byte(d), &i)
	default:
		log.Fatal("please provide a valid format")
	}

	fm := make(map[string]fileAttr)
	iterRoot(i, fm)
	return fm
}

func parseAttr(attr string) fileAttr {
	parts := strings.Split(attr, ":")
	if (len(parts) != 3) {
		log.Fatal("unable to parse attributes")
	}

	uid, err := strconv.Atoi(parts[0]); if err != nil {
		log.Fatal(err.Error())
	}

	gid, err := strconv.Atoi(parts[1]); if err != nil {
		log.Fatal(err.Error())
	}

	perm, err := strconv.ParseUint(parts[2], 8, 32); if err != nil {
		log.Fatal(err.Error())
	}

	return fileAttr {
		uid: uid,
		gid: gid,
		perm: os.FileMode(perm),
	}
}

func iterRoot(m interface{}, fm map[string]fileAttr) {
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

func iterFile(parentPath string, m map[string]interface{}, fm map[string]fileAttr) {
	filePath, ok := pathVal(m); if !ok {
		log.Fatal("path key must be provided!")
	}
	fullPath := path.Join(parentPath, filePath)
	attr, ok := attrVal(m); if ok {
		_, ok := fm[fullPath]; if ok {
			log.Fatal("duplicate path: ", fullPath)
		}
		fm[fullPath] = parseAttr(attr)
	}
	files, ok := filesVal(m); if ok {
		for _, i := range files {
			iterFile(fullPath, prepareFile(i), fm)
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

func hitVal(key string, m map[string]interface{}) (string, bool) {
	i, ok := m[key]; if ok {
		v, ok := i.(string); if ok {
			return v, true
		}
	}

	return "", false
}

func pathVal(m map[string]interface{}) (string, bool) {
	return hitVal("path", m)
}

func attrVal(m map[string]interface{}) (string, bool) {
	return hitVal("attr", m)
}

func filesVal(m map[string]interface{}) ([]interface{}, bool) {
	i, ok := m["files"]; if !ok {
		return nil, false
	}
	v, ok := i.([]interface {}); if !ok {
		return nil, false
	}
	return v, true
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

func fileNotExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return os.IsNotExist(err)
}
