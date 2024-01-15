package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
)

type UserServer struct {
	UserId   int `bun:"user_id,pk"`
	ServerId int `bun:"server_id,pk"`
}

type Server struct {
	Id                int `bun:"id,autoincrement,pk"`
	Name              string
	UserServer        []UserServer        `bun:"rel:has-many,join:id=server_id"`
	Channel           []Channel           `bun:"rel:has-many,join:id=server_id"`
	ServerBotEndpoint []ServerBotEndpoint `bun:"rel:has-many,join:id=server_id"`
}

type User struct {
	Id           int            `json:"id" bun:"id,pk"`
	Name         string         `json:"name" bun:"name"`
	Active       bool           `json:"active" bun:"active"`
	IconImage    string         `json:"iconImage" bun:"icon_image"`
	UserServer   []UserServer   `bun:"rel:has-many,join:id=user_id"`
	Message      []Message      `bun:"rel:has-many,join:id=user_id"`
	UserReaction []UserReaction `bun:"rel:has-many,join:id=user_id"`
}

type Channel struct {
	Id       int `bun:"id,autoincrement,pk"`
	Name     string
	ServerId int       `bun:"server_id"`
	Message  []Message `bun:"rel:has-many,join:id=channel_id"`
}

type Message struct {
	Id           int `bun:"id,autoincrement,pk"`
	Message      string
	CreatedAt    time.Time      `bun:",nullzero,notnull,default:current_timestamp"`
	UserID       int            `bun:"user_id"`
	ChannelID    int            `bun:"channel_id"`
	UserReaction []UserReaction `bun:"rel:has-many,join:id=message_id"`
}

type UserReaction struct {
	Id             int `bun:"id,autoincrement,pk"`
	MessageID      int `bun:"message_id"`
	UserID         int `bun:"user_id"`
	ReactionTypeID int `bun:"reaction_type_id"`
}

type ReactionType struct {
	Id           int            `bun:"id,autoincrement,pk"`
	Emoji        string         `bun:"emoji"`
	UserReaction []UserReaction `bun:"rel:has-many,join:id=reaction_type_id"`
}

type ServerBotEndpoint struct {
	Id            int `bun:"id,autoincrement,pk"`
	BotEndpointID int `bun:"bot_endpoint_id"`
	ServerID      int `bun:"server_id"`
}

type BotEndpoint struct {
	Id                int `bun:"id,autoincrement,pk"`
	Endpoint          string
	ServerBotEndpoint []ServerBotEndpoint `bun:"rel:has-many,join:id=bot_endpoint_id"`
}

type MyDB struct {
	*bun.DB
}

func main() {
	dbPass := "pass"
	dbUser := "user"
	dbName := "dbname"
	dbAddress := "postgres-db"
	if os.Getenv("DB_ENV") == "production" {
		dbUser = os.Getenv("DB_USER")
		if dbUser == "" {
			log.Fatalf("DB_USER is not set: %s", dbUser)
		}
		dbPass = os.Getenv("DB_PASS")
		if dbPass == "" {
			log.Fatalf("DB_PASS is not set: %s", dbPass)
		}
		dbAddress = os.Getenv("DB_ADDRESS")
		if dbAddress == "" {
			log.Fatalf("DB_ADDRESS is not set: %s", dbAddress)
		}
		dbName = os.Getenv("DB_NAME")
		if dbName == "" {
			log.Fatalf("DB_NAME is not set: %s", dbName)
		}
	}
	//本番環境とローカル環境でsslmodeを変更
	sslmode := "disable"
	if os.Getenv("DB_ENV") == "production" {
		sslmode = "verify-full"
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:5432/%s?sslmode=%s", dbUser, dbPass, dbAddress, dbName, sslmode)
	//https://bun.uptrace.dev/postgres/#pgdriver
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))

	db := MyDB{bun.NewDB(sqldb, pgdialect.New())}

	//接続できているか確認
	if err := db.Ping(); err != nil {
		panic(err)
	}
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))
	ctx := context.Background()
	//テーブル作成
	db.NewCreateTable().Model((*User)(nil)).IfNotExists().Exec(ctx)
	db.NewCreateTable().Model((*Server)(nil)).IfNotExists().Exec(ctx)
	db.NewCreateTable().Model((*Channel)(nil)).IfNotExists().Exec(ctx)
	db.NewCreateTable().Model((*Message)(nil)).IfNotExists().Exec(ctx)
	db.NewCreateTable().Model((*UserReaction)(nil)).IfNotExists().Exec(ctx)
	db.NewCreateTable().Model((*ReactionType)(nil)).IfNotExists().Exec(ctx)
	db.NewCreateTable().Model((*BotEndpoint)(nil)).IfNotExists().Exec(ctx)
	db.NewCreateTable().Model((*ServerBotEndpoint)(nil)).IfNotExists().Exec(ctx)
	db.NewCreateTable().Model((*UserServer)(nil)).IfNotExists().Exec(ctx)

	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})
	r.POST("/AddUser", db.InsertUser)

	if err := r.Run(":8080"); err != nil {
		panic(err)
	}

}

func (db *MyDB) InsertUser(ctx *gin.Context) {
	var user User
	if err := ctx.BindJSON(&user); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	db.NewInsert().Model(&user).Exec(ctx)
	ctx.JSON(200, gin.H{"message": "success"})
}
