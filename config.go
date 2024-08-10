package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Interface struct {
		Addr string `json:"addr"`
		Port int    `json:"port"`
		TLS  struct {
			Cert string `json:"cert"`
			Key  string `json:"key"`
		} `json:"tls"`
	} `json:"interface"`
	Database struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		User     string `json:"user"`
		Passwd   string `json:"passwd"`
		Database string `json:"database"`
	} `json:"database"`
}

func ReadConfig(path string) (Config, error) {
	var config Config
	if path != "" {
		raw, err := os.ReadFile(path)
		if err != nil {
			return config, err
		}
		err = json.Unmarshal(raw, &config)
		if err != nil {
			return config, err
		}
	}

	var varValue string
	varValue = os.Getenv("GODFATHER_TLS_HOST")
	if varValue != "" {
		config.Interface.Addr = varValue
	}
	varValue = os.Getenv("GODFATHER_TLS_PORT")
	if varValue != "" {
		port, err := strconv.Atoi(varValue)
		if err != nil {
			return config, fmt.Errorf("GODFATHER_TLS_PORT is not an integer")
		}
		config.Interface.Port = port
	}
	varValue = os.Getenv("GODFATHER_TLS_CERT")
	if varValue != "" {
		config.Interface.TLS.Cert = varValue
	}
	varValue = os.Getenv("GODFATHER_TLS_KEY")
	if varValue != "" {
		config.Interface.TLS.Key = varValue
	}
	varValue = os.Getenv("GODFATHER_DB_NAME")
	if varValue != "" {
		config.Database.Database = varValue
	}
	varValue = os.Getenv("GODFATHER_DB_USER")
	if varValue != "" {
		config.Database.User = varValue
	}
	varValue = os.Getenv("GODFATHER_DB_PASSWD")
	if varValue != "" {
		config.Database.Passwd = varValue
	}
	varValue = os.Getenv("GODFATHER_DB_HOST")
	if varValue != "" {
		config.Database.Host = varValue
	}
	varValue = os.Getenv("GODFATHER_DB_PORT")
	if varValue != "" {
		port, err := strconv.Atoi(varValue)
		if err != nil {
			return config, fmt.Errorf("GODFATHER_DB_PORT is not an integer")
		}
		config.Database.Port = port
	}

	return config, nil
}
