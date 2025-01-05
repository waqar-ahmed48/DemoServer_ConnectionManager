package datalayer

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"DemoServer_ConnectionManager/configuration"
	"DemoServer_ConnectionManager/data"
	"DemoServer_ConnectionManager/utilities"

	"go.opentelemetry.io/otel"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	_ "github.com/lib/pq"
)

type PostgresDataSource struct {
	c    *configuration.Config
	l    *slog.Logger
	rodb *gorm.DB
	rwdb *gorm.DB
}

func NewPostgresDataSource(c *configuration.Config, l *slog.Logger) (*PostgresDataSource, error) {

	var sslmode string

	if c.Postgres.SSLMode {
		sslmode = "enable"
	} else {
		sslmode = "disable"
	}

	roDsn := fmt.Sprintf("host=%s user=%s password=%s port=%d sslmode=%s dbname=%s", c.Postgres.Host, c.Postgres.ROUsername, c.Postgres.ROPassword, c.Postgres.Port, sslmode, strings.ToLower(c.DataLayer.NamePrefix))
	rwDsn := fmt.Sprintf("host=%s user=%s password=%s port=%d sslmode=%s", c.Postgres.Host, c.Postgres.RWUsername, c.Postgres.RWPassword, c.Postgres.Port, sslmode)

	db, err := sql.Open("postgres", rwDsn)

	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("CREATE DATABASE %s;", strings.ToLower(c.DataLayer.NamePrefix))
	tx, _ := db.Begin()
	_, _ = db.Exec(query)

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	err = db.Close()
	if err != nil {
		return nil, err
	}

	rwDsn = fmt.Sprintf("host=%s user=%s password=%s port=%d sslmode=%s dbname=%s", c.Postgres.Host, c.Postgres.RWUsername, c.Postgres.RWPassword, c.Postgres.Port, sslmode, strings.ToLower(c.DataLayer.NamePrefix))

	rwdb, err := gorm.Open(postgres.Open(rwDsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqldb, err := rwdb.DB()
	if err != nil {
		return nil, err
	}

	sqldb.SetMaxIdleConns(c.Postgres.RWConnectionPoolSize)
	sqldb.SetMaxOpenConns(c.Postgres.RWConnectionPoolSize)
	sqldb.SetConnMaxLifetime(0)

	err = sqldb.Ping()
	if err != nil {
		return nil, err
	}

	rodb, err := gorm.Open(postgres.Open(roDsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqldb, err = rodb.DB()
	if err != nil {
		return nil, err
	}

	sqldb.SetMaxIdleConns(c.Postgres.ROConnectionPoolSize)
	sqldb.SetMaxOpenConns(c.Postgres.ROConnectionPoolSize)
	sqldb.SetConnMaxLifetime(0)

	err = sqldb.Ping()
	if err != nil {
		return nil, err
	}

	return &PostgresDataSource{c, l, rwdb, rodb}, nil
}

func (d *PostgresDataSource) AutoMigrate() error {
	return d.rwdb.AutoMigrate(&data.AWSConnection{}, &data.AuditRecord{})
}

func (d *PostgresDataSource) RODB() *gorm.DB {
	return d.rodb
}

func (d *PostgresDataSource) RWDB() *gorm.DB {
	return d.rwdb
}

func (d *PostgresDataSource) Ping(ctx context.Context) error {
	tr := otel.Tracer(d.c.Server.PrefixMain)
	_, span := tr.Start(ctx, utilities.GetFunctionName())
	defer span.End()

	sqldb, err := d.rodb.DB()

	if err != nil {
		return err
	}

	err = sqldb.Ping()
	if err != nil {
		return err
	}

	sqldb, err = d.rwdb.DB()
	if err != nil {
		return err
	}

	err = sqldb.Ping()
	return err
}

func UpdateObject[T any](db *gorm.DB, obj *T, ctx context.Context, tracerName string) error {

	tr := otel.Tracer(tracerName)
	_, span := tr.Start(ctx, utilities.GetFunctionName())
	defer span.End()

	// Begin a transaction
	tx := db.Begin()

	// Check if the transaction started successfully
	if tx.Error != nil {
		return tx.Error
	}

	result := tx.Save(obj)

	if result.Error != nil {
		tx.Rollback()
		return result.Error
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func CreateObject[T any](db *gorm.DB, obj *T, ctx context.Context, tracerName string) error {

	tr := otel.Tracer(tracerName)
	_, span := tr.Start(ctx, utilities.GetFunctionName())
	defer span.End()

	// Begin a transaction
	tx := db.Begin()

	// Check if the transaction started successfully
	if tx.Error != nil {
		return tx.Error
	}

	result := tx.Create(obj)

	if result.Error != nil {
		tx.Rollback()
		return result.Error
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
