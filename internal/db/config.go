package db

type Config struct {
	Type            string
	Host            string
	Port            string
	Name            string
	User            string
	Password        string
	SSLMode         string
	MaxIdleConn     int
	MaxOpenConn     int
	ConnMaxLifetime int
	ConnMaxIdleTime int
}
