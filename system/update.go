package system

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"reflect"
	"sort"
	"strings"

	"github.com/coreos/coreos-cloudinit/config"
)

const (
	locksmithUnit    = "locksmithd.service"
	updateEngineUnit = "update-engine.service"
)

// Update is a top-level structure which contains its underlying configuration,
// config.Update, a function for reading the configuration (the default
// implementation reading from the filesystem), and provides the system-specific
// File() and Unit().
type Update struct {
	Config     config.Update
	ReadConfig func() (io.Reader, error)
}

func DefaultReadConfig() (io.Reader, error) {
	etcUpdate := path.Join("/etc", "coreos", "update.conf")
	usrUpdate := path.Join("/usr", "share", "coreos", "update.conf")

	f, err := os.Open(etcUpdate)
	if os.IsNotExist(err) {
		f, err = os.Open(usrUpdate)
	}
	return f, err
}

// File generates an `/etc/coreos/update.conf` file (if any update
// configuration options are set in cloud-config) by either rewriting the
// existing file on disk, or starting from `/usr/share/coreos/update.conf`
func (uc Update) File() (*File, error) {
	if config.IsZero(uc.Config) {
		return nil, nil
	}
	if err := config.AssertValid(uc.Config); err != nil {
		return nil, err
	}

	// Generate the list of possible substitutions to be performed based on the options that are configured
	subs := map[string]string{}
	uct := reflect.TypeOf(uc.Config)
	ucv := reflect.ValueOf(uc.Config)
	for i := 0; i < uct.NumField(); i++ {
		val := ucv.Field(i).String()
		if val == "" {
			continue
		}
		env := uct.Field(i).Tag.Get("env")
		subs[env] = fmt.Sprintf("%s=%s", env, val)
	}

	conf, err := uc.ReadConfig()
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(conf)

	var out string
	for scanner.Scan() {
		line := scanner.Text()
		for env, value := range subs {
			if strings.HasPrefix(line, env) {
				line = value
				delete(subs, env)
				break
			}
		}
		out += line
		out += "\n"
		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}

	for _, key := range sortedKeys(subs) {
		out += subs[key]
		out += "\n"
	}

	return &File{config.File{
		Path:               path.Join("etc", "coreos", "update.conf"),
		RawFilePermissions: "0644",
		Content:            out,
	}}, nil
}

// Units generates units for the cloud-init initializer to act on:
// - a locksmith Unit, if "reboot-strategy" was set in cloud-config
// - an update_engine Unit, if "group" or "server" was set in cloud-config
func (uc Update) Units() ([]Unit, error) {
	var units []Unit
	if uc.Config.RebootStrategy != "" {
		ls := &Unit{config.Unit{
			Name:    locksmithUnit,
			Command: "restart",
			Mask:    false,
			Runtime: true,
		}}

		if uc.Config.RebootStrategy == "off" {
			ls.Command = "stop"
			ls.Mask = true
		}
		units = append(units, *ls)
	}

	if uc.Config.Group != "" || uc.Config.Server != "" {
		ue := Unit{config.Unit{
			Name:    updateEngineUnit,
			Command: "restart",
		}}
		units = append(units, ue)
	}

	return units, nil
}

func sortedKeys(m map[string]string) (keys []string) {
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return
}
