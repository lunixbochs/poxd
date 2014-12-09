package main

import (
	"io/ioutil"
	"log"

	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	ApiKeys []string `yaml:"api_keys"`
}

func LoadConfig(path string) (*Config, error) {
	config := &Config{}
	return config, config.Load(path)
}

func (c *Config) Load(path string) error {
	data := try(ioutil.ReadFile(path))
	try(yaml.Unmarshal(data, c))
	return nil
}

func (c *Config) Save(path string) error {
	data := try(yaml.Marshal(c))
	try(ioutil.WriteFile(path, data, 0700))
	return nil
}
