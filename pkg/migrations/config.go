package migrations

import "fmt"

type Config struct {
	DBDriver            DBDriver
	Addr                string
	DBName              string
	User                string
	Password            string
	DisableConfirmation bool
}

func (c Config) Validate() error {
	if !c.DBDriver.Valid() {
		return fmt.Errorf("invalid db driver: %s", c.DBDriver)
	}

	if c.Addr == "" {
		return fmt.Errorf("addr is required")
	}

	if c.DBName == "" {
		return fmt.Errorf("db name is required")
	}

	if c.User == "" {
		return fmt.Errorf("user is required")
	}

	if c.Password == "" {
		return fmt.Errorf("password is required")
	}

	return nil
}
