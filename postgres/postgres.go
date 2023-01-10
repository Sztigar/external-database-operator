package postgres

import (
	"database/sql"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/lib/pq"
)

type Postgres struct {
	host     string
	port     string
	user     string
	password string
	dbName   string
	db       *sql.DB
	log      logr.Logger
}

func New(host, port, user, password, dbName string, log logr.Logger) (Postgres, error) {
	log.WithName("Postgres")
	connectionString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbName)
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Error(err, "Error connecting to database!")
		return Postgres{}, err
	}
	if err = db.Ping(); err != nil {
		log.Error(err, "Error pinging database")
		return Postgres{}, err
	}
	postgres := &Postgres{
		host:     host,
		port:     port,
		user:     user,
		password: password,
		dbName:   dbName,
		db:       db,
		log:      log,
	}
	return *postgres, nil
}

func (p *Postgres) CloseConnection() error {
	err := p.db.Close()
	if err != nil {
		p.log.Error(err, "Error closing database connection!")
		return err
	}
	return nil
}

func (p *Postgres) CreateDatabase(databaseName string) error {
	_, err := p.db.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, databaseName))
	if err != nil {
		//Error code "42P04" means database already exists
		if err.(*pq.Error).Code == "42P04" {
			p.log.Info("Database already exists")
			return nil
		}
		p.log.Error(err, "Error creating database")
		return err
	}
	p.log.Info("Database created")
	return nil
}

func (p *Postgres) DropDatabase(databaseName string) error {
	_, err := p.db.Exec(fmt.Sprintf(`DROP DATABASE "%s"`, databaseName))
	if err != nil {
		//Error code "3D000" means database doesn't exists
		if err.(*pq.Error).Code == "3D000" {
			p.log.Info("Database was already droped")
		}
		p.log.Error(err, "Error droping database")
		return err
	}
	p.log.Info("Database droped")
	return nil
}

func (p *Postgres) CreateRole(roleName string) error {
	_, err := p.db.Exec(fmt.Sprintf(`CREATE ROLE "%s"`, roleName))
	if err != nil {
		//Error code "42710" means role already exists
		if err.(*pq.Error).Code == "42710" {
			p.log.Info("Role already exists")
			return nil
		}
		p.log.Error(err, "Error when droping database")
		return err
	}
	p.log.Info("Role created")
	return nil
}

func (p *Postgres) DropRole(roleName string) error {
	_, err := p.db.Exec(fmt.Sprintf(`DROP ROLE "%s"`, roleName))
	if err != nil {
		// Error code "42704" means role not found
		if err.(*pq.Error).Code != "42704" {
			p.log.Info("Role already deleted")
			return nil
		}
		p.log.Error(err, "Error when droping role")
		return err
	}
	p.log.Info("Role dropped")
	return nil
}
