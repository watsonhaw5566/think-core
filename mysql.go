package tg

import (
	"errors"
	"fmt"
	"github.com/fatih/color"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/think-go/tg/tgcfg"
	"github.com/think-go/tg/tgutl"
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
	Debug       bool   // 是否开启全局SQL打印
	CreateTime  string // 创建时间字段名
	UpdateTime  string // 更新时间字段名
	DeleteTime  string // 删除时间字段名
	MaxOpen     int    // 最大打开连接数
	MaxIdle     int    // 最大空闲连接数
	MaxIdleTime int    // 连接在空闲状态下的最大存活时间
	MaxLifeTime int    // 连接的最大生命周期，从创建到被关闭的总时间
}

type InsertOption struct {
	Debug      *bool  // 是否打印最终执行的SQL语句，默认不打印
	AutoTime   *bool  // 是否开启自动时间戳，默认不开启
	CreateTime string // 创建时间字段名，默认 create_time
	UpdateTime string // 更新时间字段名，默认 update_time
}

type InsertAllOption struct {
	Debug      *bool  // 是否打印最终执行的SQL语句，默认不打印
	AutoTime   *bool  // 是否开启自动时间戳，默认不开启
	CreateTime string // 创建时间字段名，默认 create_time
	UpdateTime string // 更新时间字段名，默认 update_time
}

type UpdateOption struct {
	Debug      *bool  // 是否打印最终执行的SQL语句，默认不打印
	AutoTime   *bool  // 是否开启自动时间戳，默认不开启
	UpdateTime string // 更新时间字段名，默认 update_time
	AllProtect *bool  // 全量更新保护，默认开启，防止忘记写WHERE条件误更新所有数据
}

type DecrOption struct {
	Debug      *bool  // 是否打印最终执行的SQL语句，默认不打印
	AutoTime   *bool  // 是否开启自动时间戳，默认不开启
	UpdateTime string // 更新时间字段名，默认 update_time
	AllProtect *bool  // 全量更新保护，默认开启，防止忘记写WHERE条件误更新所有数据
}

type IncrOption struct {
	Debug      *bool  // 是否打印最终执行的SQL语句，默认不打印
	AutoTime   *bool  // 是否开启自动时间戳，默认不开启
	UpdateTime string // 更新时间字段名，默认 update_time
	AllProtect *bool  // 全量更新保护，默认开启，防止忘记写WHERE条件误更新所有数据
}

type DeleteOption struct {
	IsDeleteFlag  *bool  // 是否是软删除，默认是
	Debug         *bool  // 是否打印最终执行的SQL语句，默认不打印
	DeleteTime    string // 删除时间字段名，默认 delete_time
	DeleteProtect *bool  // 删除保护，默认开启，防止忘记写WHERE条件误删除所有数据，只争对物理删除有效
}

type CountOption struct {
	Debug      *bool  // 是否打印sql默认不打印
	DeleteTime string // 软删除字段名
}

type FindOneOption struct {
	Debug      *bool  // 是否打印sql默认不打印
	DeleteTime string // 软删除字段名
}

type SelectOption struct {
	Debug      *bool  // 是否打印sql默认不打印
	DeleteTime string // 软删除字段名
}

type tdb struct {
	instance  *sqlx.DB
	tableName string
	whereStr  string
	fieldStr  string
	joinStr   string
	lockStr   string
	values    []interface{}
	tx        *sqlx.Tx
	config    Source
}

type begin struct {
	tx     *sqlx.Tx
	source []Source
}

// 创建连接池
func createInstance(source ...Source) (instance *sqlx.DB, config Source) {
	config = Source{
		Link:        tgcfg.Config.GetMySqlSource("default.link").String(),
		Debug:       tgcfg.Config.GetMySqlSource("default.debug").Bool(),
		CreateTime:  tgcfg.Config.GetMySqlSource("default.createTime").String(),
		UpdateTime:  tgcfg.Config.GetMySqlSource("default.updateTime").String(),
		DeleteTime:  tgcfg.Config.GetMySqlSource("default.deleteTime").String(),
		MaxOpen:     int(tgcfg.Config.GetMySqlSource("default.maxOpen").Int()),
		MaxIdle:     int(tgcfg.Config.GetMySqlSource("default.maxIdle").Int()),
		MaxIdleTime: int(tgcfg.Config.GetMySqlSource("default.maxIdleTime").Int()),
		MaxLifeTime: int(tgcfg.Config.GetMySqlSource("default.maxLifeTime").Int()),
	}
	if len(source) > 0 {
		config = Source{
			Link:        source[0].Link,
			Debug:       source[0].Debug,
			CreateTime:  source[0].CreateTime,
			UpdateTime:  source[0].UpdateTime,
			DeleteTime:  source[0].DeleteTime,
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
	instance, _ := createInstance(source...)
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
	instance, config := createInstance(source...)
	return &tdb{
		instance:  instance,
		tableName: tableName,
		whereStr:  "",
		fieldStr:  "*",
		config:    config,
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
	in, _, _ := sqlx.In("IN (?)", value)
	db.whereStr += fmt.Sprintf("%s %s %s", str, field, in)
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

// Join 连表查询, tg.Db('user').Join("role", "user.id = role.user_id").Where("user.id", "=", 1).Select(&user)
// INNER: 如果表中有至少一个匹配，则返回行
// LEFT: 即使右表中没有匹配，也从左表返回所有的行
// RIGHT: 即使左表中没有匹配，也从右表返回所有的行
// FULL: 只要其中一个表中存在匹配，就返回行
func (db *tdb) Join(tableName string, whereStr string, joinType ...string) *tdb {
	joinTypeStr := "LEFT"
	if len(joinType) > 0 {
		joinTypeStr = joinType[0]
	}
	db.joinStr += fmt.Sprintf("%s JOIN %s ON %s", joinTypeStr, tableName, whereStr)
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

// InsertAll 添加多条数据, tg.Db("user").InsertAll(user)
func (db *tdb) InsertAll(data []interface{}, option ...InsertAllOption) (err error) {
	config := InsertAllOption{
		Debug:      &db.config.Debug,     // 是否打印最终执行的SQL语句，默认不打印
		AutoTime:   tgutl.PtrBool(false), // 是否开启自动时间戳，默认不开启
		CreateTime: db.config.CreateTime, // 更新时间字段名，默认 create_time
		UpdateTime: db.config.UpdateTime, // 更新时间字段名，默认 update_time
	}
	if len(option) > 0 {
		if option[0].Debug != nil {
			config.Debug = option[0].Debug
		}
		if option[0].AutoTime != nil {
			config.AutoTime = option[0].AutoTime
		}
		createTime := option[0].CreateTime
		if createTime != "" {
			config.CreateTime = createTime
		}
		updateTime := option[0].UpdateTime
		if updateTime != "" {
			config.UpdateTime = updateTime
		}
	}

	intoStr := ""
	valueStr := ""
	switch v := data[0].(type) {
	case map[string]interface{}:
		for key := range v {
			if intoStr != "" {
				intoStr += ", "
			}
			intoStr += key
			if valueStr != "" {
				valueStr += ", "
			}
			valueStr += "?"
		}
	case struct{}:
		elem := reflect.ValueOf(data).Elem()
		for i := 0; i < elem.NumField(); i++ {
			field := elem.Type().Field(i)
			if intoStr != "" {
				intoStr += ", "
			}
			intoStr += field.Tag.Get("p")
			db.values = append(db.values, elem.Field(i).Interface())
			if valueStr != "" {
				valueStr += ", "
			}
			valueStr += "?"
		}
	}

	if *config.AutoTime {
		if !strings.Contains(intoStr, config.CreateTime) {
			intoStr += fmt.Sprintf(", %s", config.CreateTime)
			valueStr += ", NOW()"
		}
		if !strings.Contains(intoStr, config.UpdateTime) {
			intoStr += fmt.Sprintf(", %s", config.UpdateTime)
			valueStr += ", NOW()"
		}
	}

	valuesStr := ""
	for _, item := range data {
		if valuesStr != "" {
			valuesStr += ", "
		}
		valuesStr += fmt.Sprintf("(%s)", valueStr)
		switch v := item.(type) {
		case map[string]interface{}:
			for _, value := range v {
				db.values = append(db.values, value)
			}
		case struct{}:
			elem := reflect.ValueOf(data).Elem()
			for i := 0; i < elem.NumField(); i++ {
				db.values = append(db.values, elem.Field(i).Interface())
			}
		}
	}

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s", db.tableName, intoStr, valuesStr)

	var stmt *sqlx.Stmt
	if db.tx != nil {
		stmt, err = db.tx.Preparex(sql)
	} else {
		stmt, err = db.instance.Preparex(sql)
	}
	if err != nil {
		return
	}
	_, err = stmt.Exec(db.values...)
	if err != nil {
		return
	}

	// DEBUG sql语句打印
	if *config.Debug {
		fmt.Println("[SQL] " + tgutl.SqlFormat(sql, db.values))
	}

	return nil
}

// Insert 添加数据, tg.Db("user").Insert(user)
func (db *tdb) Insert(data interface{}, option ...InsertOption) (insertId int64, err error) {
	config := InsertOption{
		Debug:      &db.config.Debug,     // 是否打印最终执行的SQL语句，默认不打印
		AutoTime:   tgutl.PtrBool(false), // 是否开启自动时间戳，默认不开启
		CreateTime: db.config.CreateTime, // 更新时间字段名，默认 create_time
		UpdateTime: db.config.UpdateTime, // 更新时间字段名，默认 update_time
	}
	if len(option) > 0 {
		if option[0].Debug != nil {
			config.Debug = option[0].Debug
		}
		if option[0].AutoTime != nil {
			config.AutoTime = option[0].AutoTime
		}
		createTime := option[0].CreateTime
		if createTime != "" {
			config.CreateTime = createTime
		}
		updateTime := option[0].UpdateTime
		if updateTime != "" {
			config.UpdateTime = updateTime
		}
	}

	intoStr := ""
	valueStr := ""
	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			if intoStr != "" {
				intoStr += ", "
			}
			intoStr += key
			db.values = append(db.values, value)
			if valueStr != "" {
				valueStr += ", "
			}
			valueStr += "?"
		}
	case struct{}:
		elem := reflect.ValueOf(data).Elem()
		for i := 0; i < elem.NumField(); i++ {
			field := elem.Type().Field(i)
			if intoStr != "" {
				intoStr += ", "
			}
			intoStr += field.Tag.Get("p")
			db.values = append(db.values, elem.Field(i).Interface())
			if valueStr != "" {
				valueStr += ", "
			}
			valueStr += "?"
		}
	}

	if *config.AutoTime {
		if !strings.Contains(intoStr, config.CreateTime) {
			intoStr += fmt.Sprintf(", %s", config.CreateTime)
			valueStr += ", NOW()"
		}
		if !strings.Contains(intoStr, config.UpdateTime) {
			intoStr += fmt.Sprintf(", %s", config.UpdateTime)
			valueStr += ", NOW()"
		}
	}

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", db.tableName, intoStr, valueStr)

	var stmt *sqlx.Stmt
	if db.tx != nil {
		stmt, err = db.tx.Preparex(sql)
	} else {
		stmt, err = db.instance.Preparex(sql)
	}
	if err != nil {
		return
	}
	res, err := stmt.Exec(db.values...)
	if err != nil {
		return res.LastInsertId()
	}

	// DEBUG sql语句打印
	if *config.Debug {
		fmt.Println("[SQL] " + tgutl.SqlFormat(sql, db.values))
	}

	return res.LastInsertId()
}

// Update 更新数据, tg.Db("user").Where("id", "=", 1).Update(user)
func (db *tdb) Update(data any, option ...UpdateOption) (err error) {
	config := UpdateOption{
		Debug:      &db.config.Debug,     // 是否打印最终执行的SQL语句，默认不打印
		AutoTime:   tgutl.PtrBool(false), // 是否开启自动时间戳，默认不开启
		UpdateTime: db.config.UpdateTime, // 更新时间字段名，默认 update_time
		AllProtect: tgutl.PtrBool(true),  // 全量更新保护，默认开启，防止忘记写WHERE条件误更新所有数据
	}
	if len(option) > 0 {
		if option[0].Debug != nil {
			config.Debug = option[0].Debug
		}
		if option[0].AutoTime != nil {
			config.AutoTime = option[0].AutoTime
		}
		updateTime := option[0].UpdateTime
		if updateTime != "" {
			config.UpdateTime = updateTime
		}
		if option[0].AllProtect != nil {
			config.AllProtect = option[0].AllProtect
		}
	}

	setStr := ""
	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			if setStr != "" {
				setStr += ", "
			}
			setStr += fmt.Sprintf("%s = ?", key)
			db.values = append([]interface{}{value}, db.values...)
		}
	case struct{}:
		elem := reflect.ValueOf(data).Elem()
		for i := 0; i < elem.NumField(); i++ {
			field := elem.Type().Field(i)
			if setStr != "" {
				setStr += ", "
			}
			setStr += fmt.Sprintf("%s = ?", field.Tag.Get("p"))
			db.values = append([]interface{}{elem.Field(i).Interface()}, db.values...)
		}
	}

	if *config.AutoTime {
		if strings.Contains(setStr, config.UpdateTime) {
			setStr = strings.Replace(setStr, fmt.Sprintf("%s = ?", config.UpdateTime), fmt.Sprintf("%s = NOW()", config.UpdateTime), 1)
		} else {
			setStr += fmt.Sprintf("%s = NOW()", config.UpdateTime)
		}
	}

	if *config.AllProtect && !strings.Contains(db.whereStr, "WHERE") {
		warn := "警告：是否忘记增加WHERE条件，如果需要全量更新，请关闭全量更新保护"
		color.Yellow(warn)
		return errors.New(warn)
	}

	sql := fmt.Sprintf("UPDATE %s SET %s %s", db.tableName, setStr, db.whereStr)

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
	_, err = stmt.Exec(db.values...)
	if err != nil {
		return
	}

	// DEBUG sql语句打印
	if *config.Debug {
		fmt.Println("[SQL] " + tgutl.SqlFormat(sql, db.values))
	}

	return nil
}

// Decr 以某个字段递减, tg.Db("user").Where("id", "=", 1).Decr("score", 1)
func (db *tdb) Decr(field string, num int, option ...DecrOption) (err error) {
	config := DecrOption{
		Debug:      &db.config.Debug,     // 是否打印最终执行的SQL语句，默认不打印
		AutoTime:   tgutl.PtrBool(false), // 是否开启自动时间戳，默认不开启
		UpdateTime: db.config.UpdateTime, // 更新时间字段名，默认 update_time
		AllProtect: tgutl.PtrBool(true),  // 全量更新保护，默认开启，防止忘记写WHERE条件误更新所有数据
	}
	if len(option) > 0 {
		if option[0].Debug != nil {
			config.Debug = option[0].Debug
		}
		if option[0].AutoTime != nil {
			config.AutoTime = option[0].AutoTime
		}
		updateTime := option[0].UpdateTime
		if updateTime != "" {
			config.UpdateTime = updateTime
		}
		if option[0].AllProtect != nil {
			config.AllProtect = option[0].AllProtect
		}
	}

	setStr := fmt.Sprintf("%s = %s - ?", field, field)
	if *config.AutoTime {
		setStr += fmt.Sprintf(", %s = NOW()", config.UpdateTime)
	}

	sql := fmt.Sprintf("UPDATE %s SET %s %s %s", db.tableName, setStr, db.whereStr, db.lockStr)
	db.values = append([]interface{}{num}, db.values...)

	if *config.AllProtect && !strings.Contains(db.whereStr, "WHERE") {
		warn := "警告：是否忘记增加WHERE条件，如果需要全量更新，请关闭全量更新保护"
		color.Yellow(warn)
		return errors.New(warn)
	}

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
	_, err = stmt.Exec(db.values...)
	if err != nil {
		return
	}

	// DEBUG sql语句打印
	if *config.Debug {
		fmt.Println("[SQL] " + tgutl.SqlFormat(sql, db.values))
	}
	return nil
}

// Incr 以某个字段递增, tg.Db("user").Where("id", "=", 1).Incr("score", 1)
func (db *tdb) Incr(field string, num int, option ...IncrOption) (err error) {
	config := DecrOption{
		Debug:      &db.config.Debug,     // 是否打印最终执行的SQL语句，默认不打印
		AutoTime:   tgutl.PtrBool(false), // 是否开启自动时间戳，默认不开启
		UpdateTime: db.config.UpdateTime, // 更新时间字段名，默认 update_time
		AllProtect: tgutl.PtrBool(true),  // 全量更新保护，默认开启，防止忘记写WHERE条件误更新所有数据
	}
	if len(option) > 0 {
		if option[0].Debug != nil {
			config.Debug = option[0].Debug
		}
		if option[0].AutoTime != nil {
			config.AutoTime = option[0].AutoTime
		}
		updateTime := option[0].UpdateTime
		if updateTime != "" {
			config.UpdateTime = updateTime
		}
		if option[0].AllProtect != nil {
			config.AllProtect = option[0].AllProtect
		}
	}

	setStr := fmt.Sprintf("%s = %s + ?", field, field)
	if *config.AutoTime {
		setStr += fmt.Sprintf(", %s = NOW()", config.UpdateTime)
	}

	sql := fmt.Sprintf("UPDATE %s SET %s %s", db.tableName, setStr, db.whereStr)
	db.values = append([]interface{}{num}, db.values...)

	if *config.AllProtect && !strings.Contains(db.whereStr, "WHERE") {
		warn := "警告：是否忘记增加WHERE条件，如果需要全量更新，请关闭全量更新保护"
		color.Yellow(warn)
		return errors.New(warn)
	}

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
	_, err = stmt.Exec(db.values...)
	if err != nil {
		return
	}

	// DEBUG sql语句打印
	if *config.Debug {
		fmt.Println("[SQL] " + tgutl.SqlFormat(sql, db.values))
	}
	return nil
}

// Delete 删除数据
func (db *tdb) Delete(option ...DeleteOption) (err error) {
	config := DeleteOption{
		IsDeleteFlag:  tgutl.PtrBool(true),  // 是否是软删除，默认是
		Debug:         &db.config.Debug,     // 是否打印最终执行的SQL语句，默认不打印
		DeleteTime:    db.config.DeleteTime, // 删除时间字段名，默认 delete_time
		DeleteProtect: tgutl.PtrBool(true),  // 删除保护，默认开启，防止忘记写WHERE条件误删除所有数据，只争对物理删除有效
	}
	if len(option) > 0 {
		if option[0].IsDeleteFlag != nil {
			config.IsDeleteFlag = option[0].IsDeleteFlag
		}
		if option[0].Debug != nil {
			config.Debug = option[0].Debug
		}
		deleteTime := option[0].DeleteTime
		if deleteTime != "" {
			config.DeleteTime = deleteTime
		}
		if option[0].DeleteProtect != nil {
			config.DeleteProtect = option[0].DeleteProtect
		}
	}

	sql := ""
	warn := "警告：是否忘记增加WHERE条件，如果需要删除全部，请关闭删除保护"
	var stmt *sqlx.Stmt
	if *config.IsDeleteFlag {
		sql = fmt.Sprintf("UPDATE %s SET %s = NOW() %s %s", db.tableName, config.DeleteTime, db.whereStr, db.lockStr)
		if *config.DeleteProtect && !strings.Contains(db.whereStr, "WHERE") {
			color.Yellow(warn)
			return errors.New(warn)
		}
	} else {
		sql = fmt.Sprintf("DELETE FROM %s %s", db.tableName, db.whereStr)
		if *config.DeleteProtect && !strings.Contains(db.whereStr, "WHERE") {
			color.Yellow(warn)
			return errors.New(warn)
		}
	}

	if db.tx != nil {
		stmt, err = db.tx.Preparex(sql)
	} else {
		stmt, err = db.instance.Preparex(sql)
	}
	if err != nil {
		return
	}
	defer stmt.Close()
	_, err = stmt.Exec(db.values...)
	if err != nil {
		return
	}

	// DEBUG sql语句打印
	if *config.Debug {
		fmt.Println("[SQL] " + tgutl.SqlFormat(sql, db.values))
	}
	return nil
}

// ALL 查询包含软删除的数据, tg.Db("user").ALL().Select(&user)
func (db *tdb) ALL(deleteTime ...string) *tdb {
	delTime := db.config.DeleteTime
	if len(deleteTime) > 0 {
		delTime = deleteTime[0]
	}
	db.whereStr = strings.Replace(db.whereStr, fmt.Sprintf("%s IS NULL", delTime), "", 1)
	return db
}

// Count 查询数量, tg.Db("user").Count()
func (db *tdb) Count(option ...CountOption) (count int, err error) {
	config := CountOption{
		Debug:      &db.config.Debug,
		DeleteTime: db.config.DeleteTime,
	}
	if len(option) > 0 {
		if option[0].Debug != nil {
			config.Debug = option[0].Debug
		}
		deleteTime := option[0].DeleteTime
		if deleteTime != "" {
			config.DeleteTime = deleteTime
		}
	}
	if !strings.Contains(db.whereStr, fmt.Sprintf("%s IS NULL", config.DeleteTime)) {
		db.WhereIsNull(config.DeleteTime)
	}
	sql := fmt.Sprintf("SELECT COUNT(*) FROM %s %s %s %s", db.tableName, db.joinStr, db.whereStr, db.lockStr)
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

	// DEBUG sql语句打印
	if *config.Debug {
		fmt.Println("[SQL] " + tgutl.SqlFormat(sql, db.values))
	}
	return
}

// FindOne 查询一条数据, tg.Db("user").Where("age", ">", 18).FindOne(&user)
func (db *tdb) FindOne(scan any, option ...FindOneOption) (err error) {
	config := FindOneOption{
		Debug:      &db.config.Debug,
		DeleteTime: db.config.DeleteTime,
	}
	if len(option) > 0 {
		if option[0].Debug != nil {
			config.Debug = option[0].Debug
		}
		deleteTime := option[0].DeleteTime
		if deleteTime != "" {
			config.DeleteTime = deleteTime
		}
	}
	if !strings.Contains(db.whereStr, fmt.Sprintf("%s IS NULL", config.DeleteTime)) {
		db.WhereIsNull(config.DeleteTime)
	}
	sql := fmt.Sprintf("SELECT %s FROM %s %s %s %s", db.fieldStr, db.tableName, db.joinStr, db.whereStr, db.lockStr)
	var stmt *sqlx.Stmt
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

	// DEBUG sql语句打印
	if *config.Debug {
		fmt.Println("[SQL] " + tgutl.SqlFormat(sql, db.values))
	}
	return nil
}

// Select 查询多条数据, tg.Db("user").Where("age", ">", 18).Select(&user)
func (db *tdb) Select(scan any, option ...SelectOption) (err error) {
	config := SelectOption{
		Debug:      &db.config.Debug,
		DeleteTime: db.config.DeleteTime,
	}
	if len(option) > 0 {
		if option[0].Debug != nil {
			config.Debug = option[0].Debug
		}
		deleteTime := option[0].DeleteTime
		if deleteTime != "" {
			config.DeleteTime = deleteTime
		}
	}
	if !strings.Contains(db.whereStr, fmt.Sprintf("%s IS NULL", config.DeleteTime)) {
		db.WhereIsNull(config.DeleteTime)
	}
	sql := fmt.Sprintf("SELECT %s FROM %s %s %s %s", db.fieldStr, db.tableName, db.joinStr, db.whereStr, db.lockStr)
	var stmt *sqlx.Stmt
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

	// DEBUG sql语句打印
	if *config.Debug {
		fmt.Println("[SQL] " + tgutl.SqlFormat(sql, db.values))
	}
	return nil
}

// ExecSql 自定义书写sql语句
func ExecSql(source ...Source) *sqlx.DB {
	instance, _ := createInstance(source...)
	return instance
}
