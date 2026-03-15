package database

import (
	"os"
)

// env returns the current environment, defaulting to "development"
var env = func() string {
	e := os.Getenv("APP_ENV")
	if e == "" {
		return "development"
	}
	return e
}()

// dbConfigFromEnv builds a DBConfig from the correct env vars based on the environment
func dbConfigFromEnv() DBConfig {
	switch env {
	case "development":
		return DBConfig{
			Host:     os.Getenv("DB_HOST_DEV"),
			Port:     os.Getenv("DB_PORT_DEV"),
			Username: os.Getenv("DB_USER_DEV"),
			Password: os.Getenv("DB_PASS_DEV"),
			Name:     os.Getenv("DB_NAME_DEV"),
		}
	case "test":
		return DBConfig{
			Host:     os.Getenv("DB_HOST_TEST"),
			Port:     os.Getenv("DB_PORT_TEST"),
			Username: os.Getenv("DB_USER_TEST"),
			Password: os.Getenv("DB_PASSWORD_TEST"),
			Name:     os.Getenv("DB_NAME_TEST"),
		}
	case "staging", "production":
		return DBConfig{
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
			Username: os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			Name:     os.Getenv("DB_NAME"),
		}
	default:
		// fallback to development
		return DBConfig{
			Host:     os.Getenv("DB_HOST_DEV"),
			Port:     os.Getenv("DB_PORT_DEV"),
			Username: os.Getenv("DB_USER_DEV"),
			Password: os.Getenv("DB_PASS_DEV"),
			Name:     os.Getenv("DB_NAME_DEV"),
		}
	}
}

// replicaConfigFromEnv builds the replica DBConfig (same across all environments)
func replicaConfigFromEnv() DBConfig {
	return DBConfig{
		Host:     os.Getenv("DB_HOST_REPLICA"),
		Port:     os.Getenv("DB_PORT_REPLICA"),
		Username: os.Getenv("DB_USER_REPLICA"),
		Password: os.Getenv("DB_PASS_REPLICA"),
		Name:     os.Getenv("DB_NAME_REPLICA"),
	}
}
