package database

import (
	"fmt"
	"os"
	"sync"

	applogger "colossa-pm/logger"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

// DBConfig holds connection details for a single database node
type DBConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	Name     string
}

func (c DBConfig) dsn() string {
	sslMode := "require"
	if os.Getenv("CHECK_SSL") == "0" {
		sslMode = "disable"
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.Username, c.Password, c.Host, c.Port, c.Name, sslMode,
	)
}

func (c DBConfig) label() string {
	return fmt.Sprintf("%s:%s/%s", c.Host, c.Port, c.Name)
}

// DbConnection is the singleton database connection manager
type DbConnection struct {
	db *gorm.DB
}

var (
	dbInstance *DbConnection
	dbOnce     sync.Once
)

// Instance returns the singleton DbConnection.
// Config is resolved automatically from environment variables.
func Instance() *DbConnection {
	dbOnce.Do(func() {
		dbInstance = &DbConnection{}
		dbInstance.db = buildConnection(dbConfigFromEnv(), replicaConfigFromEnv())
	})
	return dbInstance
}

// DB returns the underlying *gorm.DB instance
func (d *DbConnection) DB() *gorm.DB {
	return d.db
}

// buildConnection wires up GORM with optional read replica support
func buildConnection(master DBConfig, replica DBConfig) *gorm.DB {
	gormLogger := applogger.NewGormLogger()

	db, err := gorm.Open(postgres.Open(master.dsn()), &gorm.Config{
		Logger:                 gormLogger,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to open db connection: %v", err))
	}

	// Wire up replica if ENABLE_REPLICA is not "120"
	// (mirrors the TS logic: "0" means disable replica)
	if os.Getenv("ENABLE_REPLICA") != "0" {
		replicaDSN := postgres.Open(replica.dsn())
		err = db.Use(dbresolver.Register(dbresolver.Config{
			Replicas: []gorm.Dialector{replicaDSN},
			Policy:   dbresolver.RandomPolicy{},
		}))
		if err != nil {
			panic(fmt.Sprintf("failed to register db replica: %v", err))
		}
	}

	// Always configure the connection pool
	sqlDB, err := db.DB()
	if err != nil {
		panic(fmt.Sprintf("failed to get sql.DB: %v", err))
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)

	return db
}

// InitializeDb pings the database and logs the active connection info.
// Call this once at startup after calling Instance().
func (d *DbConnection) InitializeDb() error {
	log := applogger.Instance()

	master := dbConfigFromEnv()
	replica := replicaConfigFromEnv()

	sqlDB, err := d.db.DB()
	if err != nil {
		errMsg := fmt.Sprintf("db-error: %v", err)
		log.ErrorMsg(errMsg)
		return err
	}

	if err := sqlDB.Ping(); err != nil {
		errMsg := fmt.Sprintf("db-error: %v", err)
		log.ErrorMsg(errMsg)
		return err
	}

	var connectionLog string
	if os.Getenv("ENABLE_REPLICA") != "0" {
		connectionLog = fmt.Sprintf("%s and %s", master.label(), replica.label())
	} else {
		connectionLog = master.label()
	}

	log.Log("db-connection " + connectionLog)
	fmt.Println("===db-connection===>", connectionLog)

	return nil
}

// DisconnectDb closes the underlying connection pool
func (d *DbConnection) DisconnectDb() error {
	log := applogger.Instance()

	sqlDB, err := d.db.DB()
	if err != nil {
		log.ErrorMsg(fmt.Sprintf("db-disconnection-error: %v", err))
		return err
	}

	if err := sqlDB.Close(); err != nil {
		log.ErrorMsg(fmt.Sprintf("db-disconnection-error: %v", err))
		return err
	}

	return nil
}
