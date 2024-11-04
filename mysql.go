package tg

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/think-go/tg/tgcfg"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"
)

var (
	dbInstance = sync.Map{}
)

type Source struct {
	Link        string
	Debug       bool
	MaxOpen     int // 最大打开连接数
	MaxIdle     int // 最大空闲连接数
	MaxIdleTime int // 连接在空闲状态下的最大存活时间
	MaxLifeTime int // 连接的最大生命周期，从创建到被关闭的总时间
}

type DeleteOption struct {
	IsDeleteFlag  bool   // 是否是软删除，默认是
	Debug         bool   // 是否打印最终执行的SQL语句，默认不打印
	DeleteTime    string // 删除时间字段名，默认 delete_time
	DeleteProtect bool   // 删除保护，默认开启，防止忘记写WHERE条件误删除所有数据，只争对物理删除有效
}

type tdb struct {
	instance  *sqlx.DB
	tableName string
	whereStr  string
	fieldStr  string
	lockStr   string
	values    []interface{}
	tx        *sqlx.Tx
	debug     bool
}

type begin struct {
	tx     *sqlx.Tx
	source []Source
	debug  bool
}

// 创建连接池
func createInstance(source ...Source) (instance *sqlx.DB, config Source) {
	config = Source{
		Link:        tgcfg.Config.GetMySqlSource("default.link").String(),
		Debug:       tgcfg.Config.GetMySqlSource("default.debug").Bool(),
		MaxOpen:     int(tgcfg.Config.GetMySqlSource("default.maxOpen").Int()),
		MaxIdle:     int(tgcfg.Config.GetMySqlSource("default.maxIdle").Int()),
		MaxIdleTime: int(tgcfg.Config.GetMySqlSource("default.maxIdleTime").Int()),
		MaxLifeTime: int(tgcfg.Config.GetMySqlSource("default.maxLifeTime").Int()),
	}
	if len(source) > 0 {
		config = Source{
			Link:        source[0].Link,
			Debug:       source[0].Debug,
			MaxOpen:     source[0].MaxOpen,
			MaxIdle:     source[0].MaxIdle,
			MaxIdleTime: source[0].MaxIdleTime,
			MaxLifeTime: source[0].MaxLifeTime,
		}
	}
	if ins, ok := dbInstance.Load(config.Link); ok {
		return ins.(*sqlx.DB), config
	}
	var err error
	instance, err = sqlx.Connect("mysql", config.Link)
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
	return instance, config
}

// BeginTransaction 开启事务,如果不传数据源默认走的是配置文件里默认的,传了可以指定任意的数据源
func BeginTransaction(source ...Source) *begin {
	instance, config := createInstance(source...)
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
		debug:  config.Debug,
	}
}

// Db 事务去操作数据库
func (b *begin) Db(tableName string) *tdb {
	db := Db(tableName, b.source...)
	db.tx = b.tx
	db.debug = b.debug
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
	instance, config := createInstance(source...)
	return &tdb{
		instance:  instance,
		tableName: tableName,
		whereStr:  "",
		fieldStr:  "*",
		debug:     config.Debug,
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

// WhereAnd 指定多查询条件,前面必须有Where, tg.Db("user").Where("age", "=", 18).WhereAnd("gender", "=", 1)
func (db *tdb) WhereAnd(field string, condition string, value interface{}) *tdb {
	db.whereStr += fmt.Sprintf(" AND %s %s ?", field, condition)
	db.values = append(db.values, value)
	return db
}

// WhereOr 指定多查询条件,前面必须有Where, tg.Db("user").Where("age", "=", 18).WhereOr("age", "=", 19)
func (db *tdb) WhereOr(field string, condition string, value interface{}) *tdb {
	db.whereStr += fmt.Sprintf(" OR %s %s ?", field, condition)
	db.values = append(db.values, value)
	return db
}

// WhereIn 包含查询, tg.Db("user").WhereIn("id", []interface{1, 2, 3})
func (db *tdb) WhereIn(field string, value []interface{}) *tdb {
	str := "WHERE"
	if strings.Contains(db.whereStr, "WHERE") {
		str = " AND"
	}
	placeholders := make([]string, len(value))
	for i := range value {
		placeholders[i] = "?"
	}
	inClause := strings.Join(placeholders, ", ")
	db.whereStr += fmt.Sprintf("%s %s IN (%s)", str, field, inClause)
	db.values = append(db.values, value...)
	return db
}

// WhereLike 模糊查询, tg.Db("user").WhereLike("name", "%建国%")
func (db *tdb) WhereLike(field string, value interface{}) *tdb {
	str := "WHERE"
	if strings.Contains(db.whereStr, "WHERE") {
		str = " AND"
	}
	db.whereStr += fmt.Sprintf("%s %s LIKE ?", str, field)
	db.values = append(db.values, value)
	return db
}

// WhereBetween 区间查询, tg.Db("user").WhereBetween("age", 18, 20)
func (db *tdb) WhereBetween(field string, start interface{}, end interface{}) *tdb {
	str := "WHERE"
	if strings.Contains(db.whereStr, "WHERE") {
		str = " AND"
	}
	db.whereStr += fmt.Sprintf("%s %s BETWEEN ? AND ?", str, field)
	db.values = append(db.values, start, end)
	return db
}

// WhereIsNull 为NULL数据, tg.Db("user").WhereIsNull("name")
func (db *tdb) WhereIsNull(field string) *tdb {
	str := "WHERE"
	if strings.Contains(db.whereStr, "WHERE") {
		str = " AND"
	}
	db.whereStr += fmt.Sprintf("%s %s IS NULL", str, field)
	return db
}

// WhereIsNotNull 不为NULL数据, tg.Db("user").WhereIsNotNull("name")
func (db *tdb) WhereIsNotNull(field string) *tdb {
	str := "WHERE"
	if strings.Contains(db.whereStr, "WHERE") {
		str = " AND"
	}
	db.whereStr += fmt.Sprintf("%s %s IS NOT NULL", str, field)
	return db
}

// Limit 限制查询条数, tg.Db("user").Limit(10).Select(&user)
func (db *tdb) Limit(num int) *tdb {
	db.whereStr += fmt.Sprintf(" LIMIT %d", num)
	return db
}

// Order 排序, tg.Db("user").Order("age", "ASC").Select(&user)
func (db *tdb) Order(field string, sort ...string) *tdb {
	sortStr := "DESC"
	if len(sort) > 0 {
		sortStr = sort[0]
	}
	db.whereStr += fmt.Sprintf(" ORDER BY %s %s", field, sortStr)
	return db
}

// Page 分页查询, tg.Db("user").Page(1, 10).Select(&user)
func (db *tdb) Page(current int, size int) *tdb {
	db.whereStr += fmt.Sprintf(" LIMIT %d, %d", current-1, size)
	return db
}

// Group 分组查询, tg.Db("user").Field("id, max(score)").Group("id").Select(&user)
func (db *tdb) Group(field string) *tdb {
	db.whereStr += fmt.Sprintf(" GROUP BY %s", field)
	return db
}

// Lock 锁 只可以在事务操作中使用, 默认不传是FOR UPDATE
// 排它锁 FOR UPDATE 用于写操作
// 共享锁 LOCK IN SHARE MODE 用于读操作
func (db *tdb) Lock(lockStr ...string) *tdb {
	str := "FOR UPDATE"
	if len(lockStr) > 0 {
		str = lockStr[0]
	}
	db.lockStr = str
	return db
}

// Insert 添加数据, tg.Db("user").Insert()
//func (db *tdb) Insert(data interface{}) (insertId int, err error) {
//	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", db.tableName)
//	var stmt *sqlx.Stmt
//	if db.tx != nil {
//		stmt, err = db.tx.Preparex(sql)
//	} else {
//		stmt, err = db.instance.Preparex(sql)
//	}
//	if err != nil {
//		return
//	}
//	defer stmt.Close()
//}

// Update 更新数据, tg.Db("user").Where("id", "=", 1).Update()
func (db *tdb) Update() {
	sql := fmt.Sprintf("UPDATE %s SET %s", db.tableName, db.whereStr)
	fmt.Println(sql)
}

// Decr 以某个字段递减
func (db *tdb) Decr() {}

// Incr 以某个字段递增
func (db *tdb) Incr() {}

// Delete 删除数据
func (db *tdb) Delete(option ...DeleteOption) error {
	config := DeleteOption{
		IsDeleteFlag:  true,
		Debug:         false,
		DeleteTime:    "delete_time",
		DeleteProtect: true,
	}
	if len(option) > 0 {
		config = DeleteOption{
			IsDeleteFlag:  option[0].IsDeleteFlag,
			Debug:         option[0].Debug,
			DeleteTime:    option[0].DeleteTime,
			DeleteProtect: option[0].DeleteProtect,
		}
	}
	if config.IsDeleteFlag {
		//
	} else {

	}
	return nil
}

// Count 查询数量, tg.Db("user").Count()
func (db *tdb) Count() (count int, err error) {
	sql := fmt.Sprintf("SELECT COUNT(*) FROM %s %s %s", db.tableName, db.whereStr, db.lockStr)
	var stmt *sqlx.Stmt
	if db.tx != nil {
		stmt, err = db.tx.Preparex(sql)
	} else {
		stmt, err = db.instance.Preparex(sql)
	}
	if err != nil {
		return
	}
	defer stmt.Close()
	udb := stmt.Unsafe()
	err = udb.Get(&count, db.values...)
	if err != nil {
		return
	}
	return
}

// FindOne 查询一条数据, tg.Db("user").Where("age", ">", 18).FindOne(&user)
func (db *tdb) FindOne(scan any) error {
	sql := fmt.Sprintf("SELECT %s FROM %s %s %s", db.fieldStr, db.tableName, db.whereStr, db.lockStr)
	var stmt *sqlx.Stmt
	var err error
	if db.tx != nil {
		stmt, err = db.tx.Preparex(sql)
	} else {
		stmt, err = db.instance.Preparex(sql)
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

// Select 查询多条数据, tg.Db("user").Where("age", ">", 18).Select(&user)
// 是否打印sql默认不打印
func (db *tdb) Select(scan any, debug ...bool) error {
	sql := fmt.Sprintf("SELECT %s FROM %s %s %s", db.fieldStr, db.tableName, db.whereStr, db.lockStr)
	var stmt *sqlx.Stmt
	var err error
	if db.tx != nil {
		stmt, err = db.tx.Preparex(sql)
	} else {
		stmt, err = db.instance.Preparex(sql)
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

	isShowSql := false
	if len(debug) > 0 {
		isShowSql = debug[0]
	}
	if db.debug {
		fmt.Println()
	} else if isShowSql {
		fmt.Println()
	}
	return nil
}
