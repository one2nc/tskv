package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"

	"github.com/tsocial/tskv/storage"
	"gopkg.in/alecthomas/kingpin.v2"
)

const Version = "0.0.1"
const Archive = "archive"

func timestamp() string {
	return fmt.Sprintf("%v", time.Now().UnixNano())
}

var (
	app        = kingpin.New("tskv", "")
	consulAddr = app.Flag("consul", "Consul address").OverrideDefaultFromEnvar("TSKV_CONSUL_ADDR").String()

	getCmd    = app.Command("get", "Get last set value of a key")
	getCmdKey = getCmd.Arg("key", "Key to get").Required().String()

	setCmd    = app.Command("set", "Set a key")
	setCmdKey = setCmd.Arg("key", "Key").Required().String()
	setCmdTag = setCmd.Flag("tag", "Tag").Default(timestamp()).String()
	setCmdVal = setCmd.Arg("value", "Value").Required().File()

	rollbackCmd    = app.Command("rollback", "Rollback value of key to a specified tag")
	rollbackCmdTag = rollbackCmd.Flag("tag", "Tag").Required().String()
	rollbackCmdKey = rollbackCmd.Arg("key", "Key").Required().String()

	listTagCmd = app.Command("list", "List tags")
	listCmdKey = listTagCmd.Arg("key", "Key").Required().String()
)

func MakeValue(key string, value []byte) *Value {
	return &Value{key: key, storage: value}
}

type Value struct {
	storage []byte
	key     string
}

func (w *Value) SaveId(string) {}

func (w *Value) IsCompressed() bool {
	return false
}

func (w *Value) Key() string {
	return w.key
}

func (w *Value) MakePath(t *storage.Tree) string {
	return path.Join(t.MakePath(), w.key)
}

func (w *Value) Unmarshal(b []byte) error {
	w.storage = b
	return nil
}

func (w *Value) Marshal() ([]byte, error) {
	return w.storage, nil
}

func getKey(c storage.Store, key string) []byte {
	tree := storage.MakeTree(Archive)
	w := MakeValue(key, nil)

	err := c.Get(w, tree)
	if err != nil {
		panic(err)
	}
	return w.storage
}

func setKey(c storage.Store, key, tag string, b []byte) {
	//NOTE: Trim the extra newline character
	b = bytes.TrimRight(b, "\n")

	w := MakeValue(key, b)
	if err := c.SaveTag(w, storage.MakeTree(Archive), tag); err != nil {
		panic(err)
	}

	if err := c.SaveTag(w, nil, tag); err != nil {
		panic(err)
	}
}

func rollback(c storage.Store, key, tag string) {
	tree := storage.MakeTree(Archive)
	w := MakeValue(key, nil)
	if err := c.GetVersion(w, tree, tag); err != nil {
		panic(err)
	}

	if err := c.SaveTag(w, tree, timestamp()); err != nil {
		panic(err)
	}
}

func listVersions(c storage.Store, key string) []string {
	tree := storage.MakeTree(Archive)
	w := MakeValue(key, nil)

	l, err := c.GetVersions(w, tree)
	if err != nil {
		panic(err)
	}
	return l
}

func main() {
	app.Version(Version)

	c := storage.MakeConsulStore(*consulAddr)
	if err := c.Setup(); err != nil {
		panic(err)
	}

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {

	case getCmd.FullCommand():
		log.Println(string(getKey(c, *getCmdKey)))

	case setCmd.FullCommand():
		b, err := ioutil.ReadAll(*setCmdVal)
		if err != nil {
			panic(err)
		}
		setKey(c, *setCmdKey, *setCmdTag, b)

	case rollbackCmd.FullCommand():
		rollback(c, *rollbackCmdKey, *rollbackCmdTag)

	case listTagCmd.FullCommand():
		versions := listVersions(c, *listCmdKey)
		log.Println(versions)

	default:
		log.Println(app.Help)
	}
}
