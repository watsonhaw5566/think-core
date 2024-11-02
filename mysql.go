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
	link      string
	tableName string
	whereStr  string
	fieldStr  string
	values    []interface{}
	tx        *sqlx.Tx
}

func (db *tdb) Begin() *tdb {
	n := *db
	if instance, ok := dbInstance.Load(n.link); ok {
		tx, err := instance.(*sqlx.DB).Beginx()
		if err != nil {
			n.tx = tx
		}
	}
	return &n
}

// Db 如果不传数据源默认走的是配置文件里默认的,传了可以指定任意的数据源
func Db(tableName string, source ...Source) (db *tdb) {
	config := Source{
		Link:        tgcfg.Config.GetMySqlSource("default.link").String(),
		MaxOpen:     int(tgcfg.Config.GetMySqlSource("default.maxOpen").Int()),
		MaxIdle:     int(tgcfg.Config.GetMySqlSource("default.maxIdle").Int()),
		MaxIdleTime: int(tgcfg.Config.GetMySqlSource("default.maxIdleTime").Int()),
		MaxLifeTime: int(tgcfg.Config.GetMySqlSource("default.maxLifeTime").Int()),
	}
	if len(source) > 0 {
		config = Source{
			Link:        source[0].Link,
			MaxOpen:     source[0].MaxOpen,
			MaxIdle:     source[0].MaxIdle,
			MaxIdleTime: source[0].MaxIdleTime,
			MaxLifeTime: source[0].MaxLifeTime,
		}
	}
	db = &tdb{
		link:      config.Link,
		tableName: tableName,
		whereStr:  "",
		fieldStr:  "*",
	}
	if _, ok := dbInstance.Load(config.Link); ok {
		return
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
	return
}

func (db *tdb) Field(fields string, distinct ...bool) *tdb {
	n := *db
	dis := ""
	if len(distinct) > 0 && distinct[0] {
		dis = "DISTINCT "
	}
	n.fieldStr = fmt.Sprintf("%s%s", dis, fields)
	return &n
}

func (db *tdb) Where(field string, condition string, value interface{}) *tdb {
	n := *db
	n.whereStr += fmt.Sprintf("WHERE %s %s ?", field, condition)
	n.values = append(n.values, value)
	return &n
}

func (db *tdb) WhereAnd(field string, condition string, value interface{}) *tdb {
	n := *db
	n.whereStr += fmt.Sprintf(" AND %s %s ?", field, condition)
	n.values = append(n.values, value)
	return &n
}

func (db *tdb) WhereOr(field string, condition string, value interface{}) *tdb {
	n := *db
	n.whereStr += fmt.Sprintf(" OR %s %s ?", field, condition)
	n.values = append(n.values, value)
	return &n
}

func (db *tdb) FindOne(scan any) error {
	n := *db
	if instance, ok := dbInstance.Load(n.link); ok {
		query := fmt.Sprintf("SELECT %s FROM %s %s LIMIT 1", n.fieldStr, n.tableName, n.whereStr)
		stmt, err := instance.(*sqlx.DB).Preparex(query)
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
		err = udb.Get(scan, n.values...)
		if err != nil {
			return err
		}

	}
	return nil
}

func (db *tdb) Select(scan any) error {
	n := *db
	if instance, ok := dbInstance.Load(n.link); ok {
		query := fmt.Sprintf("SELECT %s FROM %s %s", n.fieldStr, n.tableName, n.whereStr)
		stmt, err := instance.(*sqlx.DB).Preparex(query)
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
		err = udb.Select(scan, n.values...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *tdb) Commit() {}

func (db *tdb) Rollback() {}
