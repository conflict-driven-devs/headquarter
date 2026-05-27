package config

func DefaultDBConfig() *DBConfig {
	return &DBConfig{
		Host:     getEnv("DB_HOST", "postgres"),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", ""),
		Name:     getEnv("DB_NAME", "postgres"),
		Port:     getEnv("DB_PORT", "5432"),
	}
}
