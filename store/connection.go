package store

import (
	"github.com/jmoiron/sqlx"
	//_ "github.com/sijms/go-ora/v2"
	go_ora "github.com/sijms/go-ora/v2"
	"log"
)

type Client struct {
	conn *sqlx.DB
}

func GetConnection() (*Client, error) {
	host := "adb.eu-amsterdam-1.oraclecloud.com"
	port := 1522
	service := "g68bb372c582cce_prakharadb_low.adb.oraclecloud.com"
	user := "admin"
	pass := ""
	wallet := "./wallet"
	url := go_ora.BuildUrl(host, port, service, user, pass, map[string]string{
		"wallet":     wallet,
		"ssl":        "true",
		"ssl_verify": "true",
		//"server":     "pooled",
		//"TRACE FILE": "trace.log",
	})

	db, err := sqlx.Open("oracle", url)
	if err != nil {
		log.Println("Open", err)
		return nil, err
	}

	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(1)

	err = db.Ping()
	if err != nil {
		log.Println("Ping", err)
		return nil, err
	}

	return &Client{conn: db}, nil
}

func (c *Client) Close() {
	c.conn.Close()
}
