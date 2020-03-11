package bdb

import (
	"fmt"

	log "github.com/p9c/logi"

	walletdb "github.com/p9c/wallet/db"
)

const (
	dbType = "bdb"
)

// parseArgs parses the arguments from the walletdb Open/Create methods.
func parseArgs(funcName string, args ...interface{}) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("invalid arguments to %s.%s -- "+
			"expected database path", dbType, funcName)
	}
	dbPath, ok := args[0].(string)
	if !ok {
		return "", fmt.Errorf("first argument to %s.%s is invalid -- "+
			"expected database path string", dbType, funcName)
	}
	return dbPath, nil
}

// openDBDriver is the callback provided during driver registration that opens
// an existing database for use.
func openDBDriver(args ...interface{}) (walletdb.DB, error) {
	dbPath, err := parseArgs("Open", args...)
	if err != nil {
		log.L.Error(err)
		return nil, err
	}
	return openDB(dbPath, false)
}

// createDBDriver is the callback provided during driver registration that
// creates, initializes, and opens a database for use.
func createDBDriver(args ...interface{}) (walletdb.DB, error) {
	dbPath, err := parseArgs("Create", args...)
	if err != nil {
		log.L.Error(err)
		return nil, err
	}
	return openDB(dbPath, true)
}
func init() {
	// Register the driver.
	driver := walletdb.Driver{
		DbType: dbType,
		Create: createDBDriver,
		Open:   openDBDriver,
	}
	if err := walletdb.RegisterDriver(driver); err != nil {
		panic(fmt.Sprintf("Failed to regiser database driver '%s': %v",
			dbType, err))
	}
}
