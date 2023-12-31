package mongo

import (
	"crypto/tls"
	"net"
	"os"
	"time"

	"github.com/globalsign/mgo"
	"github.com/exadevt/go-commont/errors"
	"github.com/exadevt/go-commont/jepson"
)

type Config struct {
	ConnectionString string           `json:"connectionString"`
	Timeout          *jepson.Duration `json:"timeout"`
	Scheme           string           `json:"scheme" envconfig:"TIDEPOOL_STORE_SCHEME" default:"mongodb"`
	User             string           `json:"user" envconfig:"TIDEPOOL_STORE_USERNAME" required:"true"`
	Password         string           `json:"password" envconfig:"TIDEPOOL_STORE_PASSWORD" required:"true"`
	Database         string           `json:"database" envconfig:"TIDEPOOL_STORE_DATABASE" required:"true"`
	Ssl              bool             `json:"ssl" envconfig:"TIDEPOOL_STORE_TLS" default:"true"`
	Hosts            string           `json:"hosts" envconfig:"TIDEPOOL_STORE_ADDRESSES" required:"true"`
	OptParams        string           `json:"optParams" envconfig:"TIDEPOOL_STORE_OPT_PARAMS" default:"`
}

func (config *Config) FromEnv() {
	config.Scheme, _ = os.LookupEnv("TIDEPOOL_STORE_SCHEME")
	config.Hosts, _ = os.LookupEnv("TIDEPOOL_STORE_ADDRESSES")
	config.User, _ = os.LookupEnv("TIDEPOOL_STORE_USERNAME")
	config.Password, _ = os.LookupEnv("TIDEPOOL_STORE_PASSWORD")
	config.Database, _ = os.LookupEnv("TIDEPOOL_STORE_DATABASE")
	config.OptParams, _ = os.LookupEnv("TIDEPOOL_STORE_OPT_PARAMS")
	ssl, found := os.LookupEnv("TIDEPOOL_STORE_TLS")
	config.Ssl = found && ssl == "true"
}

func (config *Config) ToConnectionString() (string, error) {
	if config.ConnectionString != "" {
		return config.ConnectionString, nil
	}
	if config.Database == "" {
		return "", errors.New("Must specify a database in Mongo config")
	}

	var cs string
	if config.Scheme != "" {
		cs = config.Scheme + "://"
	} else {
		cs = "mongodb://"
	}

	if config.User != "" {
		cs += config.User
		if config.Password != "" {
			cs += ":"
			cs += config.Password
		}
		cs += "@"
	}

	if config.Hosts != "" {
		cs += config.Hosts
		cs += "/"
	} else {
		cs += "localhost/"
	}

	if config.Database != "" {
		cs += config.Database
	}

	if config.Ssl {
		cs += "?ssl=true"
	} else {
		cs += "?ssl=false"
	}

	if config.OptParams != "" {
		cs += "&"
		cs += config.OptParams
	}
	return cs, nil
}

func Connect(config *Config) (*mgo.Session, error) {
	connectionString, err := config.ToConnectionString()
	if err != nil {
		return nil, err
	}
	dur := 20 * time.Second
	if config.Timeout != nil {
		dur = time.Duration(*config.Timeout)
	}

	dialInfo, err := mgo.ParseURL(connectionString)
	if err != nil {
		return nil, err
	}
	dialInfo.Timeout = dur

	if dialInfo.DialServer != nil {
		// TODO Ignore server cert for now.  We should install proper CA to verify cert.
		dialInfo.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
			return tls.Dial("tcp", addr.String(), &tls.Config{InsecureSkipVerify: true})
		}
	}
	return mgo.DialWithInfo(dialInfo)
}
