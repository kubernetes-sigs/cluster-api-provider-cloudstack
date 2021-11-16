package cloud

import (
	"fmt"
	"os"
	"path"

	"gopkg.in/ini.v1"
)

// Dumb CloudStack API config reader. Works for now.
func readAPIConfig() (string, string, string) {
	dir := os.Getenv("PROJECT_DIR")
	cc_path := path.Join(dir, "cloud-config")
	cfg, err := ini.Load(cc_path)
	if err != nil {
		fmt.Println(err, "could not read cloud-config", dir)
		os.Exit(1)
	}
	g := cfg.Section("Global")
	return g.Key("api-url").Value(), g.Key("api-key").Value(), g.Key("secret-key").Value()
}

func check(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

type set func(string)

func setIfNotEmpty(str string, setFn set) {
	if str != "" {
		setFn(str)
	}
}
