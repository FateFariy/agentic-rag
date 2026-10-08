package infra

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// MySQLConf MySQL 配置
type MySQLConf struct {
	Addr     string `json:",default=127.0.0.1:3306"`
	Username string
	Password string
	DbName   string
}

// NewDB 按配置创建 MySQL GORM 连接；失败直接 panic（启动期致命错误）。
// 各微服务通过它建立各自的数据库连接，同库不同表归属。
func NewDB(c MySQLConf) *gorm.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", c.Username, c.Password, c.Addr, c.DbName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		//Logger:         logger.Default.LogMode(logger.Info),
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
	if err != nil {
		panic(err)
	}
	return db
}
