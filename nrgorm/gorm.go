package nrgorm

import (
	"fmt"
	"strings"

	newrelic "github.com/newrelic/go-agent"
	"gorm.io/gorm"
)

const (
	txnGormKey   = "newrelicTransaction"
	startTimeKey = "newrelicStartTime"
)

// SetTxnToGorm sets transaction to gorm settings, returns cloned DB
func SetTxnToGorm(txn newrelic.Transaction, db *gorm.DB) *gorm.DB {
	if txn == nil {
		return db
	}
	// Set returns not clonnable db, and if we do two queries on it, second one gets all the clauses of first.
	// Creating session seems to solve this problem
	return db.Set(txnGormKey, txn).Session(&gorm.Session{SkipDefaultTransaction: true})
}

// AddGormCallbacks adds callbacks to NewRelic, you should call SetTxnToGorm to make them work
func AddGormCallbacks(db *gorm.DB) {
	dialect := db.Config.Dialector.Name()
	var product newrelic.DatastoreProduct
	switch dialect {
	case "postgres":
		product = newrelic.DatastorePostgres
	case "mysql":
		product = newrelic.DatastoreMySQL
	case "sqlite":
		product = newrelic.DatastoreSQLite
	case "mssql":
		product = newrelic.DatastoreMSSQL
	default:
		return
	}
	callbacks := newCallbacks(product)
	registerCallbacks(db, "transaction", callbacks)
	registerCallbacks(db, "create", callbacks)
	registerCallbacks(db, "query", callbacks)
	registerCallbacks(db, "update", callbacks)
	registerCallbacks(db, "delete", callbacks)
	registerCallbacks(db, "row", callbacks)
}

type callbacks struct {
	product newrelic.DatastoreProduct
}

func newCallbacks(product newrelic.DatastoreProduct) *callbacks {
	return &callbacks{product}
}

func (c *callbacks) beforeCreate(db *gorm.DB)   { c.before(db) }
func (c *callbacks) afterCreate(db *gorm.DB)    { c.after(db, "INSERT") }
func (c *callbacks) beforeQuery(db *gorm.DB)    { c.before(db) }
func (c *callbacks) afterQuery(db *gorm.DB)     { c.after(db, "SELECT") }
func (c *callbacks) beforeUpdate(db *gorm.DB)   { c.before(db) }
func (c *callbacks) afterUpdate(db *gorm.DB)    { c.after(db, "UPDATE") }
func (c *callbacks) beforeDelete(db *gorm.DB)   { c.before(db) }
func (c *callbacks) afterDelete(db *gorm.DB)    { c.after(db, "DELETE") }
func (c *callbacks) beforeRowQuery(db *gorm.DB) { c.before(db) }
func (c *callbacks) afterRowQuery(db *gorm.DB)  { c.after(db, "") }

func (c *callbacks) before(db *gorm.DB) {
	txn, ok := db.Get(txnGormKey)
	if !ok {
		return
	}
	db.Set(startTimeKey, newrelic.StartSegmentNow(txn.(newrelic.Transaction)))
}

func (c *callbacks) after(db *gorm.DB, operation string) {
	startTime, ok := db.Get(startTimeKey)
	if !ok {
		return
	}
	if operation == "" {
		operation = strings.ToUpper(strings.Split(db.Statement.SQL.String(), " ")[0])
	}
	segmentBuilder(
		startTime.(newrelic.SegmentStartTime),
		c.product,
		db.Statement.SQL.String(),
		operation,
		db.Statement.Table,
	).End()

	// gorm wraps insert&update into transaction automatically
	// add another segment for commit/rollback in such case
	if _, ok := db.InstanceGet("gorm:started_transaction"); !ok {
		db.Set(startTimeKey, nil)
		return
	}
	txn, _ := db.Get(txnGormKey)
	db.Set(startTimeKey, newrelic.StartSegmentNow(txn.(newrelic.Transaction)))
}

func (c *callbacks) commitOrRollback(db *gorm.DB) {
	startTime, ok := db.Get(startTimeKey)
	if !ok || startTime == nil {
		return
	}

	segmentBuilder(
		startTime.(newrelic.SegmentStartTime),
		c.product,
		"",
		"COMMIT/ROLLBACK",
		db.Statement.Table,
	).End()
}

func registerCallbacks(db *gorm.DB, name string, c *callbacks) {
	beforeName := fmt.Sprintf("newrelic:%v_before", name)
	afterName := fmt.Sprintf("newrelic:%v_after", name)
	gormCallbackName := fmt.Sprintf("gorm:%v", name)
	// gorm does some magic, if you pass CallbackProcessor here - nothing works
	switch name {
	case "create":
		db.Callback().Create().Before(gormCallbackName).Register(beforeName, c.beforeCreate)
		db.Callback().Create().After(gormCallbackName).Register(afterName, c.afterCreate)
		db.Callback().Create().
			After("gorm:commit_or_rollback_transaction").
			Register(fmt.Sprintf("newrelic:commit_or_rollback_transaction_%v", name), c.commitOrRollback)
	case "query":
		db.Callback().Query().Before(gormCallbackName).Register(beforeName, c.beforeQuery)
		db.Callback().Query().After(gormCallbackName).Register(afterName, c.afterQuery)
	case "update":
		db.Callback().Update().Before(gormCallbackName).Register(beforeName, c.beforeUpdate)
		db.Callback().Update().After(gormCallbackName).Register(afterName, c.afterUpdate)
		db.Callback().Update().
			After("gorm:commit_or_rollback_transaction").
			Register(fmt.Sprintf("newrelic:commit_or_rollback_transaction_%v", name), c.commitOrRollback)
	case "delete":
		db.Callback().Delete().Before(gormCallbackName).Register(beforeName, c.beforeDelete)
		db.Callback().Delete().After(gormCallbackName).Register(afterName, c.afterDelete)
		db.Callback().Delete().
			After("gorm:commit_or_rollback_transaction").
			Register(fmt.Sprintf("newrelic:commit_or_rollback_transaction_%v", name), c.commitOrRollback)
	case "row":
		db.Callback().Row().Before(gormCallbackName).Register(beforeName, c.beforeRowQuery)
		db.Callback().Row().After(gormCallbackName).Register(afterName, c.afterRowQuery)
	}
}

type segment interface {
	End() error
}

// create segment through function to be able to test it
var segmentBuilder = func(
	startTime newrelic.SegmentStartTime,
	product newrelic.DatastoreProduct,
	query string,
	operation string,
	collection string,
) segment {
	return &newrelic.DatastoreSegment{
		StartTime:          startTime,
		Product:            product,
		ParameterizedQuery: query,
		Operation:          operation,
		Collection:         collection,
	}
}
