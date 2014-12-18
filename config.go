package main

import (
	"io/ioutil"
	"log"
	"os"
	"path"

	"gopkg.in/fsnotify.v1"
	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	ApiKeys []string `yaml:"api_keys"`
	Wiring  Wiring   `yaml:"wire"`
}

func LoadConfig(configPath string) (*Config, error) {
	config := &Config{}
	try(config.source(configPath))
	return config, nil
}

func (c *Config) reload(configPath string) error {
	data := try(ioutil.ReadFile(configPath))
	try(yaml.Unmarshal(data, c))
	return nil
}

func (c *Config) source(configPath string) error {
	try(c.reload(configPath))
	go func() {
		first := true
		for {
			watcher, err := fsnotify.NewWatcher()
			if err != nil {
				log.Println(err)
				return
			}
			// turtles all the way down... TODO: handle deletion of config parent
			err = watcher.Add(path.Dir(configPath))
			if err != nil {
				if os.IsNotExist(err) {
					log.Println("Config directory missing, reloader aborted:", path.Dir(configPath))
				} else {
					log.Println("here", err)
				}
				return
			}
			if first {
				first = false
			} else {
				err = c.reload(configPath)
				if err != nil && !os.IsNotExist(err) {
					log.Println("Error reloading config:", err)
				} else {
					log.Println("Reloaded config.")
				}
			}
			for event := range watcher.Events {
				if event.Name == path.Dir(configPath) && event.Op == fsnotify.Remove {
					watcher.Close()
					break
				}
				if event.Name == configPath && (event.Op == fsnotify.Create || event.Op == fsnotify.Write) {
					if err := c.reload(configPath); err != nil {
						log.Println("Error reloading config:", err)
					} else {
						log.Println("Reloaded config.")
					}
				}
			}
		}
	}()
	return nil
}

func (c *Config) Save(path string) error {
	data := try(yaml.Marshal(c))
	try(ioutil.WriteFile(path, data, 0700))
	return nil
}
