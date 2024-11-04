package tg

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/think-go/tg/tgcfg"
	"net/http"
	"reflect"
	"sync"
	"time"
)

var (
	dbInstance = sync.Map{}
)

type Source struct {
	Link        string
	MaxOpen     int // 最大打开连接数
	MaxIdle     int // 最大空闲连接数
	MaxIdleTime int // 连接在空闲状态下的最大存活时间
	MaxLifeTime int // 连接的最大生命周期，从创建到被关闭的总时间
}

type tdb struct {
	instance  *sqlx.DB
	tableName string
	whereStr  string
	fieldStr  string
	values    []interface{}
	tx        *sqlx.Tx
}

type begin struct {
	tx     *sqlx.Tx
	source []Source
}

// 创建连接池
func createInstance(source ...Source) *sqlx.DB {
	config := &Source{
		Link:        tgcfg.Config.GetMySqlSource("default.link").String(),
		MaxOpen:     int(tgcfg.Config.GetMySqlSource("default.maxOpen").Int()),
		MaxIdle:     int(tgcfg.Config.GetMySqlSource("default.maxIdle").Int()),
		MaxIdleTime: int(tgcfg.Config.GetMySqlSource("default.maxIdleTime").Int()),
		MaxLifeTime: int(tgcfg.Config.GetMySqlSource("default.maxLifeTime").Int()),
	}
	if len(source) > 0 {
		config = &Source{
			Link:        source[0].Link,
			MaxOpen:     source[0].MaxOpen,
			MaxIdle:     source[0].MaxIdle,
			MaxIdleTime: source[0].MaxIdleTime,
			MaxLifeTime: source[0].MaxLifeTime,
		}
	}
	if ins, ok := dbInstance.Load(config.Link); ok {
		return ins.(*sqlx.DB)
	}
	instance, err := sqlx.Connect("mysql", config.Link)
	if err != nil {
		panic(Exception{
			StateCode: http.StatusInternalServerError,
			ErrorCode: ErrorCode.MySqlError,
			Message:   "数据库连接异常",
			Error:     err,
		})
	}
	instance.SetMaxOpenConns(config.MaxOpen)
	instance.SetMaxIdleConns(config.MaxIdle)
	instance.SetConnMaxIdleTime(time.Duration(config.MaxIdleTime) * time.Second)
	instance.SetConnMaxLifetime(time.Duration(config.MaxLifeTime) * time.Second)
	dbInstance.Store(config.Link, instance)
	return instance
}

// BeginTransaction 开启事务,如果不传数据源默认走的是配置文件里默认的,传了可以指定任意的数据源
func BeginTransaction(source ...Source) *begin {
	instance := createInstance(source...)
	tx, err := instance.Beginx()
	if err != nil {
		panic(Exception{
			StateCode: http.StatusInternalServerError,
			ErrorCode: ErrorCode.EXCEPTION,
			Message:   "执行Begin出错",
			Error:     err,
		})
	}
	return &begin{
		tx:     tx,
		source: source,
	}
}

// Db 事务去操作数据库
func (b *begin) Db(tableName string) *tdb {
	db := Db(tableName, b.source...)
	db.tx = b.tx
	return db
}

// Commit 提交事务
func (b *begin) Commit() error {
	return b.tx.Commit()
}

// Rollback 事务回滚
func (b *begin) Rollback() error {
	return b.tx.Rollback()
}

// Db 如果不传数据源默认走的是配置文件里默认的,传了可以指定任意的数据源
func Db(tableName string, source ...Source) (db *tdb) {
	instance := createInstance(source...)
	return &tdb{
		instance:  instance,
		tableName: tableName,
		whereStr:  "",
		fieldStr:  "*",
	}
}

// Field 指定查询的字段,默认不去重
func (db *tdb) Field(fields string, distinct ...bool) *tdb {
	dis := ""
	if len(distinct) > 0 && distinct[0] {
		dis = "DISTINCT "
	}
	db.fieldStr = fmt.Sprintf("%s%s", dis, fields)
	return db
}

// Where 指定查询条件, tg.Db().Where("id", "=", 1)
func (db *tdb) Where(field string, condition string, value interface{}) *tdb {
	db.whereStr += fmt.Sprintf("WHERE %s %s ?", field, condition)
	db.values = append(db.values, value)
	return db
}

// WhereAnd 指定多查询条件,前面必须有Where, tg.Db().Where("age", "=", 18).WhereAnd("gender", "=", 1)
func (db *tdb) WhereAnd(field string, condition string, value interface{}) *tdb {
	db.whereStr += fmt.Sprintf(" AND %s %s ?", field, condition)
	db.values = append(db.values, value)
	return db
}

// WhereOr 指定多查询条件,前面必须有Where, tg.Db().Where("age", "=", 18).WhereOr("age", "=", 19)
func (db *tdb) WhereOr(field string, condition string, value interface{}) *tdb {
	db.whereStr += fmt.Sprintf(" OR %s %s ?", field, condition)
	db.values = append(db.values, value)
	return db
}

// FindOne 查询一条数据
func (db *tdb) FindOne(scan any) error {
	query := fmt.Sprintf("SELECT %s FROM %s %s LIMIT 1", db.fieldStr, db.tableName, db.whereStr)
	var stmt *sqlx.Stmt
	var err error
	if db.tx != nil {
		stmt, err = db.tx.Preparex(query)
	} else {
		stmt, err = db.instance.Preparex(query)
	}
	if err != nil {
		return err
	}
	defer stmt.Close()

	v := reflect.ValueOf(scan)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		panic(Exception{
			StateCode: http.StatusInternalServerError,
			ErrorCode: ErrorCode.MySqlError,
			Message:   "必须是指向结构体的指针",
		})
	}

	udb := stmt.Unsafe()
	err = udb.Get(scan, db.values...)
	if err != nil {
		return err
	}
	return nil
}

// Select 查询多条数据
func (db *tdb) Select(scan any) error {
	query := fmt.Sprintf("SELECT %s FROM %s %s", db.fieldStr, db.tableName, db.whereStr)
	var stmt *sqlx.Stmt
	var err error
	if db.tx != nil {
		stmt, err = db.tx.Preparex(query)
	} else {
		stmt, err = db.instance.Preparex(query)
	}
	if err != nil {
		return err
	}
	defer stmt.Close()

	v := reflect.ValueOf(scan)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Slice {
		panic(Exception{
			StateCode: http.StatusInternalServerError,
			ErrorCode: ErrorCode.MySqlError,
			Message:   "必须是指向切片的指针",
		})
	}

	udb := stmt.Unsafe()
	err = udb.Select(scan, db.values...)
	if err != nil {
		return err
	}
	return nil
}
