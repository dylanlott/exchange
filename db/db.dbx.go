// AUTOGENERATED BY gopkg.in/spacemonkeygo/dbx.v1
// DO NOT EDIT.

package db

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/lib/pq"
)

// Prevent conditional imports from causing build failures
var _ = strconv.Itoa
var _ = strings.LastIndex
var _ = fmt.Sprint
var _ sync.Mutex

var (
	WrapErr = func(err *Error) error { return err }
	Logger  func(format string, args ...interface{})

	errTooManyRows       = errors.New("too many rows")
	errUnsupportedDriver = errors.New("unsupported driver")
	errEmptyUpdate       = errors.New("empty update")
)

func logError(format string, args ...interface{}) {
	if Logger != nil {
		Logger(format, args...)
	}
}

type ErrorCode int

const (
	ErrorCode_Unknown ErrorCode = iota
	ErrorCode_UnsupportedDriver
	ErrorCode_NoRows
	ErrorCode_TxDone
	ErrorCode_TooManyRows
	ErrorCode_ConstraintViolation
	ErrorCode_EmptyUpdate
)

type Error struct {
	Err         error
	Code        ErrorCode
	Driver      string
	Constraint  string
	QuerySuffix string
}

func (e *Error) Error() string {
	return e.Err.Error()
}

func wrapErr(e *Error) error {
	if WrapErr == nil {
		return e
	}
	return WrapErr(e)
}

func makeErr(err error) error {
	if err == nil {
		return nil
	}
	e := &Error{Err: err}
	switch err {
	case sql.ErrNoRows:
		e.Code = ErrorCode_NoRows
	case sql.ErrTxDone:
		e.Code = ErrorCode_TxDone
	}
	return wrapErr(e)
}

func unsupportedDriver(driver string) error {
	return wrapErr(&Error{
		Err:    errUnsupportedDriver,
		Code:   ErrorCode_UnsupportedDriver,
		Driver: driver,
	})
}

func emptyUpdate() error {
	return wrapErr(&Error{
		Err:  errEmptyUpdate,
		Code: ErrorCode_EmptyUpdate,
	})
}

func tooManyRows(query_suffix string) error {
	return wrapErr(&Error{
		Err:         errTooManyRows,
		Code:        ErrorCode_TooManyRows,
		QuerySuffix: query_suffix,
	})
}

func constraintViolation(err error, constraint string) error {
	return wrapErr(&Error{
		Err:        err,
		Code:       ErrorCode_ConstraintViolation,
		Constraint: constraint,
	})
}

type driver interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

var (
	notAPointer     = errors.New("destination not a pointer")
	lossyConversion = errors.New("lossy conversion")
)

type DB struct {
	*sql.DB
	dbMethods

	Hooks struct {
		Now func() time.Time
	}
}

func Open(driver, source string) (db *DB, err error) {
	var sql_db *sql.DB
	switch driver {
	case "postgres":
		sql_db, err = openpostgres(source)
	default:
		return nil, unsupportedDriver(driver)
	}
	if err != nil {
		return nil, makeErr(err)
	}
	defer func(sql_db *sql.DB) {
		if err != nil {
			sql_db.Close()
		}
	}(sql_db)

	if err := sql_db.Ping(); err != nil {
		return nil, makeErr(err)
	}

	db = &DB{
		DB: sql_db,
	}
	db.Hooks.Now = time.Now

	switch driver {
	case "postgres":
		db.dbMethods = newpostgres(db)
	default:
		return nil, unsupportedDriver(driver)
	}

	return db, nil
}

func (obj *DB) Close() (err error) {
	return obj.makeErr(obj.DB.Close())
}

func (obj *DB) Open(ctx context.Context) (*Tx, error) {
	tx, err := obj.DB.Begin()
	if err != nil {
		return nil, obj.makeErr(err)
	}

	return &Tx{
		Tx:        tx,
		txMethods: obj.wrapTx(tx),
	}, nil
}

func (obj *DB) NewRx() *Rx {
	return &Rx{db: obj}
}

func DeleteAll(ctx context.Context, db *DB) (int64, error) {
	tx, err := db.Open(ctx)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err == nil {
			err = db.makeErr(tx.Commit())
			return
		}

		if err_rollback := tx.Rollback(); err_rollback != nil {
			logError("delete-all: rollback failed: %v", db.makeErr(err_rollback))
		}
	}()
	return tx.deleteAll(ctx)
}

type Tx struct {
	Tx *sql.Tx
	txMethods
}

type dialectTx struct {
	tx *sql.Tx
}

func (tx *dialectTx) Commit() (err error) {
	return makeErr(tx.tx.Commit())
}

func (tx *dialectTx) Rollback() (err error) {
	return makeErr(tx.tx.Rollback())
}

type postgresImpl struct {
	db      *DB
	dialect __sqlbundle_postgres
	driver  driver
}

func (obj *postgresImpl) Rebind(s string) string {
	return obj.dialect.Rebind(s)
}

func (obj *postgresImpl) logStmt(stmt string, args ...interface{}) {
	postgresLogStmt(stmt, args...)
}

func (obj *postgresImpl) makeErr(err error) error {
	constraint, ok := obj.isConstraintError(err)
	if ok {
		return constraintViolation(err, constraint)
	}
	return makeErr(err)
}

type postgresDB struct {
	db *DB
	*postgresImpl
}

func newpostgres(db *DB) *postgresDB {
	return &postgresDB{
		db: db,
		postgresImpl: &postgresImpl{
			db:     db,
			driver: db.DB,
		},
	}
}

func (obj *postgresDB) Schema() string {
	return `CREATE TABLE campaigns (
	pk bigserial NOT NULL,
	id text NOT NULL,
	created_at timestamp with time zone NOT NULL,
	updated_at timestamp with time zone NOT NULL,
	industry text NOT NULL,
	PRIMARY KEY ( pk ),
	UNIQUE ( id )
);
CREATE TABLE leads (
	pk bigserial NOT NULL,
	created_at timestamp with time zone NOT NULL,
	updated_at timestamp with time zone NOT NULL,
	industry text NOT NULL,
	PRIMARY KEY ( pk )
);
CREATE TABLE users (
	pk bigserial NOT NULL,
	created_at timestamp with time zone NOT NULL,
	updated_at timestamp with time zone NOT NULL,
	id text NOT NULL,
	name text NOT NULL,
	PRIMARY KEY ( pk ),
	UNIQUE ( id ),
	UNIQUE ( name )
);`
}

func (obj *postgresDB) wrapTx(tx *sql.Tx) txMethods {
	return &postgresTx{
		dialectTx: dialectTx{tx: tx},
		postgresImpl: &postgresImpl{
			db:     obj.db,
			driver: tx,
		},
	}
}

type postgresTx struct {
	dialectTx
	*postgresImpl
}

func postgresLogStmt(stmt string, args ...interface{}) {
	// TODO: render placeholders
	if Logger != nil {
		out := fmt.Sprintf("stmt: %s\nargs: %v\n", stmt, pretty(args))
		Logger(out)
	}
}

type pretty []interface{}

func (p pretty) Format(f fmt.State, c rune) {
	fmt.Fprint(f, "[")
nextval:
	for i, val := range p {
		if i > 0 {
			fmt.Fprint(f, ", ")
		}
		rv := reflect.ValueOf(val)
		if rv.Kind() == reflect.Ptr {
			if rv.IsNil() {
				fmt.Fprint(f, "NULL")
				continue
			}
			val = rv.Elem().Interface()
		}
		switch v := val.(type) {
		case string:
			fmt.Fprintf(f, "%q", v)
		case time.Time:
			fmt.Fprintf(f, "%s", v.Format(time.RFC3339Nano))
		case []byte:
			for _, b := range v {
				if !unicode.IsPrint(rune(b)) {
					fmt.Fprintf(f, "%#x", v)
					continue nextval
				}
			}
			fmt.Fprintf(f, "%q", v)
		default:
			fmt.Fprintf(f, "%v", v)
		}
	}
	fmt.Fprint(f, "]")
}

type Campaign struct {
	Pk        int64
	Id        string
	CreatedAt time.Time
	UpdatedAt time.Time
	Industry  string
}

func (Campaign) _Table() string { return "campaigns" }

type Campaign_Update_Fields struct {
}

type Campaign_Pk_Field struct {
	_set   bool
	_null  bool
	_value int64
}

func Campaign_Pk(v int64) Campaign_Pk_Field {
	return Campaign_Pk_Field{_set: true, _value: v}
}

func (f Campaign_Pk_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (Campaign_Pk_Field) _Column() string { return "pk" }

type Campaign_Id_Field struct {
	_set   bool
	_null  bool
	_value string
}

func Campaign_Id(v string) Campaign_Id_Field {
	return Campaign_Id_Field{_set: true, _value: v}
}

func (f Campaign_Id_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (Campaign_Id_Field) _Column() string { return "id" }

type Campaign_CreatedAt_Field struct {
	_set   bool
	_null  bool
	_value time.Time
}

func Campaign_CreatedAt(v time.Time) Campaign_CreatedAt_Field {
	return Campaign_CreatedAt_Field{_set: true, _value: v}
}

func (f Campaign_CreatedAt_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (Campaign_CreatedAt_Field) _Column() string { return "created_at" }

type Campaign_UpdatedAt_Field struct {
	_set   bool
	_null  bool
	_value time.Time
}

func Campaign_UpdatedAt(v time.Time) Campaign_UpdatedAt_Field {
	return Campaign_UpdatedAt_Field{_set: true, _value: v}
}

func (f Campaign_UpdatedAt_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (Campaign_UpdatedAt_Field) _Column() string { return "updated_at" }

type Campaign_Industry_Field struct {
	_set   bool
	_null  bool
	_value string
}

func Campaign_Industry(v string) Campaign_Industry_Field {
	return Campaign_Industry_Field{_set: true, _value: v}
}

func (f Campaign_Industry_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (Campaign_Industry_Field) _Column() string { return "industry" }

type Lead struct {
	Pk        int64
	CreatedAt time.Time
	UpdatedAt time.Time
	Industry  string
}

func (Lead) _Table() string { return "leads" }

type Lead_Update_Fields struct {
}

type Lead_Pk_Field struct {
	_set   bool
	_null  bool
	_value int64
}

func Lead_Pk(v int64) Lead_Pk_Field {
	return Lead_Pk_Field{_set: true, _value: v}
}

func (f Lead_Pk_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (Lead_Pk_Field) _Column() string { return "pk" }

type Lead_CreatedAt_Field struct {
	_set   bool
	_null  bool
	_value time.Time
}

func Lead_CreatedAt(v time.Time) Lead_CreatedAt_Field {
	return Lead_CreatedAt_Field{_set: true, _value: v}
}

func (f Lead_CreatedAt_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (Lead_CreatedAt_Field) _Column() string { return "created_at" }

type Lead_UpdatedAt_Field struct {
	_set   bool
	_null  bool
	_value time.Time
}

func Lead_UpdatedAt(v time.Time) Lead_UpdatedAt_Field {
	return Lead_UpdatedAt_Field{_set: true, _value: v}
}

func (f Lead_UpdatedAt_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (Lead_UpdatedAt_Field) _Column() string { return "updated_at" }

type Lead_Industry_Field struct {
	_set   bool
	_null  bool
	_value string
}

func Lead_Industry(v string) Lead_Industry_Field {
	return Lead_Industry_Field{_set: true, _value: v}
}

func (f Lead_Industry_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (Lead_Industry_Field) _Column() string { return "industry" }

type User struct {
	Pk        int64
	CreatedAt time.Time
	UpdatedAt time.Time
	Id        string
	Name      string
}

func (User) _Table() string { return "users" }

type User_Update_Fields struct {
}

type User_Pk_Field struct {
	_set   bool
	_null  bool
	_value int64
}

func User_Pk(v int64) User_Pk_Field {
	return User_Pk_Field{_set: true, _value: v}
}

func (f User_Pk_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (User_Pk_Field) _Column() string { return "pk" }

type User_CreatedAt_Field struct {
	_set   bool
	_null  bool
	_value time.Time
}

func User_CreatedAt(v time.Time) User_CreatedAt_Field {
	return User_CreatedAt_Field{_set: true, _value: v}
}

func (f User_CreatedAt_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (User_CreatedAt_Field) _Column() string { return "created_at" }

type User_UpdatedAt_Field struct {
	_set   bool
	_null  bool
	_value time.Time
}

func User_UpdatedAt(v time.Time) User_UpdatedAt_Field {
	return User_UpdatedAt_Field{_set: true, _value: v}
}

func (f User_UpdatedAt_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (User_UpdatedAt_Field) _Column() string { return "updated_at" }

type User_Id_Field struct {
	_set   bool
	_null  bool
	_value string
}

func User_Id(v string) User_Id_Field {
	return User_Id_Field{_set: true, _value: v}
}

func (f User_Id_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (User_Id_Field) _Column() string { return "id" }

type User_Name_Field struct {
	_set   bool
	_null  bool
	_value string
}

func User_Name(v string) User_Name_Field {
	return User_Name_Field{_set: true, _value: v}
}

func (f User_Name_Field) value() interface{} {
	if !f._set || f._null {
		return nil
	}
	return f._value
}

func (User_Name_Field) _Column() string { return "name" }

func toUTC(t time.Time) time.Time {
	return t.UTC()
}

func toDate(t time.Time) time.Time {
	// keep up the minute portion so that translations between timezones will
	// continue to reflect properly.
	return t.Truncate(time.Minute)
}

//
// runtime support for building sql statements
//

type __sqlbundle_SQL interface {
	Render() string

	private()
}

type __sqlbundle_Dialect interface {
	Rebind(sql string) string
}

type __sqlbundle_RenderOp int

const (
	__sqlbundle_NoFlatten __sqlbundle_RenderOp = iota
	__sqlbundle_NoTerminate
)

func __sqlbundle_Render(dialect __sqlbundle_Dialect, sql __sqlbundle_SQL, ops ...__sqlbundle_RenderOp) string {
	out := sql.Render()

	flatten := true
	terminate := true
	for _, op := range ops {
		switch op {
		case __sqlbundle_NoFlatten:
			flatten = false
		case __sqlbundle_NoTerminate:
			terminate = false
		}
	}

	if flatten {
		out = __sqlbundle_flattenSQL(out)
	}
	if terminate {
		out += ";"
	}

	return dialect.Rebind(out)
}

func __sqlbundle_flattenSQL(x string) string {
	// trim whitespace from beginning and end
	s, e := 0, len(x)-1
	for s < len(x) && (x[s] == ' ' || x[s] == '\t' || x[s] == '\n') {
		s++
	}
	for s <= e && (x[e] == ' ' || x[e] == '\t' || x[e] == '\n') {
		e--
	}
	if s > e {
		return ""
	}
	x = x[s : e+1]

	// check for whitespace that needs fixing
	wasSpace := false
	for i := 0; i < len(x); i++ {
		r := x[i]
		justSpace := r == ' '
		if (wasSpace && justSpace) || r == '\t' || r == '\n' {
			// whitespace detected, start writing a new string
			var result strings.Builder
			result.Grow(len(x))
			if wasSpace {
				result.WriteString(x[:i-1])
			} else {
				result.WriteString(x[:i])
			}
			for p := i; p < len(x); p++ {
				for p < len(x) && (x[p] == ' ' || x[p] == '\t' || x[p] == '\n') {
					p++
				}
				result.WriteByte(' ')

				start := p
				for p < len(x) && !(x[p] == ' ' || x[p] == '\t' || x[p] == '\n') {
					p++
				}
				result.WriteString(x[start:p])
			}

			return result.String()
		}
		wasSpace = justSpace
	}

	// no problematic whitespace found
	return x
}

// this type is specially named to match up with the name returned by the
// dialect impl in the sql package.
type __sqlbundle_postgres struct{}

func (p __sqlbundle_postgres) Rebind(sql string) string {
	out := make([]byte, 0, len(sql)+10)

	j := 1
	for i := 0; i < len(sql); i++ {
		ch := sql[i]
		if ch != '?' {
			out = append(out, ch)
			continue
		}

		out = append(out, '$')
		out = append(out, strconv.Itoa(j)...)
		j++
	}

	return string(out)
}

// this type is specially named to match up with the name returned by the
// dialect impl in the sql package.
type __sqlbundle_sqlite3 struct{}

func (s __sqlbundle_sqlite3) Rebind(sql string) string {
	return sql
}

type __sqlbundle_Literal string

func (__sqlbundle_Literal) private() {}

func (l __sqlbundle_Literal) Render() string { return string(l) }

type __sqlbundle_Literals struct {
	Join string
	SQLs []__sqlbundle_SQL
}

func (__sqlbundle_Literals) private() {}

func (l __sqlbundle_Literals) Render() string {
	var out bytes.Buffer

	first := true
	for _, sql := range l.SQLs {
		if sql == nil {
			continue
		}
		if !first {
			out.WriteString(l.Join)
		}
		first = false
		out.WriteString(sql.Render())
	}

	return out.String()
}

type __sqlbundle_Condition struct {
	// set at compile/embed time
	Name  string
	Left  string
	Equal bool
	Right string

	// set at runtime
	Null bool
}

func (*__sqlbundle_Condition) private() {}

func (c *__sqlbundle_Condition) Render() string {
	// TODO(jeff): maybe check if we can use placeholders instead of the
	// literal null: this would make the templates easier.

	switch {
	case c.Equal && c.Null:
		return c.Left + " is null"
	case c.Equal && !c.Null:
		return c.Left + " = " + c.Right
	case !c.Equal && c.Null:
		return c.Left + " is not null"
	case !c.Equal && !c.Null:
		return c.Left + " != " + c.Right
	default:
		panic("unhandled case")
	}
}

type __sqlbundle_Hole struct {
	// set at compiile/embed time
	Name string

	// set at runtime
	SQL __sqlbundle_SQL
}

func (*__sqlbundle_Hole) private() {}

func (h *__sqlbundle_Hole) Render() string { return h.SQL.Render() }

//
// end runtime support for building sql statements
//

func (impl postgresImpl) isConstraintError(err error) (
	constraint string, ok bool) {
	if e, ok := err.(*pq.Error); ok {
		if e.Code.Class() == "23" {
			return e.Constraint, true
		}
	}
	return "", false
}

func (obj *postgresImpl) deleteAll(ctx context.Context) (count int64, err error) {
	var __res sql.Result
	var __count int64
	__res, err = obj.driver.Exec("DELETE FROM users;")
	if err != nil {
		return 0, obj.makeErr(err)
	}

	__count, err = __res.RowsAffected()
	if err != nil {
		return 0, obj.makeErr(err)
	}
	count += __count
	__res, err = obj.driver.Exec("DELETE FROM leads;")
	if err != nil {
		return 0, obj.makeErr(err)
	}

	__count, err = __res.RowsAffected()
	if err != nil {
		return 0, obj.makeErr(err)
	}
	count += __count
	__res, err = obj.driver.Exec("DELETE FROM campaigns;")
	if err != nil {
		return 0, obj.makeErr(err)
	}

	__count, err = __res.RowsAffected()
	if err != nil {
		return 0, obj.makeErr(err)
	}
	count += __count

	return count, nil

}

type Rx struct {
	db *DB
	tx *Tx
}

func (rx *Rx) UnsafeTx(ctx context.Context) (unsafe_tx *sql.Tx, err error) {
	tx, err := rx.getTx(ctx)
	if err != nil {
		return nil, err
	}
	return tx.Tx, nil
}

func (rx *Rx) getTx(ctx context.Context) (tx *Tx, err error) {
	if rx.tx == nil {
		if rx.tx, err = rx.db.Open(ctx); err != nil {
			return nil, err
		}
	}
	return rx.tx, nil
}

func (rx *Rx) Rebind(s string) string {
	return rx.db.Rebind(s)
}

func (rx *Rx) Commit() (err error) {
	if rx.tx != nil {
		err = rx.tx.Commit()
		rx.tx = nil
	}
	return err
}

func (rx *Rx) Rollback() (err error) {
	if rx.tx != nil {
		err = rx.tx.Rollback()
		rx.tx = nil
	}
	return err
}

type Methods interface {
}

type TxMethods interface {
	Methods

	Rebind(s string) string
	Commit() error
	Rollback() error
}

type txMethods interface {
	TxMethods

	deleteAll(ctx context.Context) (int64, error)
	makeErr(err error) error
}

type DBMethods interface {
	Methods

	Schema() string
	Rebind(sql string) string
}

type dbMethods interface {
	DBMethods

	wrapTx(tx *sql.Tx) txMethods
	makeErr(err error) error
}

func openpostgres(source string) (*sql.DB, error) {
	return sql.Open("postgres", source)
}