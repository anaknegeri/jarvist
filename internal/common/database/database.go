package database

import (
	"jarvist/internal/common/config"
	"jarvist/internal/common/models"
	"jarvist/pkg/logger"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const (
	ComponentDB       = "database"
	ComponentMessages = "messages"
	ComponentLogs     = "logs"
)

// DB adalah instans database global
var DB *gorm.DB

// SetupDatabase menginisialisasi koneksi database
func SetupDatabase(cfg *config.Config, logger *logger.ContextLogger) error {
	logger.Info("Setting up database connection")

	gormConfig := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // Gunakan nama tabel singular
		},
		DisableForeignKeyConstraintWhenMigrating: true,
	}

	// Buat koneksi dengan SQLite
	db, err := gorm.Open(sqlite.Open(cfg.GetDBConnectionString()), gormConfig)
	if err != nil {
		logger.Error("Failed to connect to database: %s", err.Error())
		return err
	}

	// Set koneksi database global
	DB = db

	// Konfigurasi koneksi
	sqlDB, err := db.DB()
	if err != nil {
		logger.Error("Failed to get DB instance: %s", err.Error())
		return err
	}

	// Improve SQLite performance and reduce lock chances
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// SQLite specific optimizations
	sqlDB.Exec("PRAGMA journal_mode = WAL;")
	sqlDB.Exec("PRAGMA synchronous = NORMAL;")
	sqlDB.Exec("PRAGMA cache_size = 5000;")
	sqlDB.Exec("PRAGMA temp_store = MEMORY;")
	sqlDB.Exec("PRAGMA busy_timeout = 10000;")

	logger.Info("Database connection established")

	return nil
}

// AutoMigrate melakukan migrasi model database secara otomatis
func AutoMigrate(models ...interface{}) error {
	if DB == nil {
		return nil
	}

	return DB.AutoMigrate(models...)
}

// RunMigrations menjalankan migrasi dan seeding database
func RunMigrations(logger *logger.ContextLogger) error {
	if DB == nil {
		return nil
	}

	// Auto-migrate models
	logger.Info("Running database migrations...")
	if err := DB.AutoMigrate(
		&models.TimeZone{},
		&models.Setting{},
		&models.Location{},
		&models.Camera{},
		&models.LogEntry{},
		&models.PendingMessage{},
		&models.ProcessedFile{},
		&models.SyncedFolder{},
	); err != nil {
		logger.Error("Failed to migrate database: %s", err.Error())
		return err
	}

	// Run seeders
	logger.Info("Running database seeders...")
	if err := SeedTimeZones(DB, logger); err != nil {
		return err
	}

	// Add other seeders here
	// if err := SeedOtherTable(DB, logger); err != nil {
	//     return err
	// }

	logger.Info("Database migrations and seeding completed successfully")
	return nil
}

// GetDB mengembalikan instans database
func GetDB() *gorm.DB {
	return DB
}

// CloseDatabase menutup koneksi database
func CloseDatabase() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

func SeedTimeZones(db *gorm.DB, logger *logger.ContextLogger) error {
	logger.Info("Checking if time_zone table needs seeding...")

	var count int64
	db.Model(&models.TimeZone{}).Count(&count)
	if count > 0 {
		logger.Info("TimeZone table already has data, skipping seed")
		return nil
	}

	logger.Info("Seeding time_zone table...")

	// Define the timezone data to seed
	timeZones := []models.TimeZone{
		{Zone: "Pacific/Midway", UTCOffset: "(GMT-11:00)", Name: "Midway Island"},
		{Zone: "US/Samoa", UTCOffset: "(GMT-11:00)", Name: "Samoa"},
		{Zone: "US/Hawaii", UTCOffset: "(GMT-10:00)", Name: "Hawaii"},
		{Zone: "US/Alaska", UTCOffset: "(GMT-09:00)", Name: "Alaska"},
		{Zone: "US/Pacific", UTCOffset: "(GMT-08:00)", Name: "Pacific Time (US &amp; Canada)"},
		{Zone: "America/Tijuana", UTCOffset: "(GMT-08:00)", Name: "Tijuana"},
		{Zone: "US/Arizona", UTCOffset: "(GMT-07:00)", Name: "Arizona"},
		{Zone: "US/Mountain", UTCOffset: "(GMT-07:00)", Name: "Mountain Time (US &amp; Canada)"},
		{Zone: "America/Chihuahua", UTCOffset: "(GMT-07:00)", Name: "Chihuahua"},
		{Zone: "America/Mazatlan", UTCOffset: "(GMT-07:00)", Name: "Mazatlan"},
		{Zone: "America/Mexico_City", UTCOffset: "(GMT-06:00)", Name: "Mexico City"},
		{Zone: "America/Monterrey", UTCOffset: "(GMT-06:00)", Name: "Monterrey"},
		{Zone: "Canada/Saskatchewan", UTCOffset: "(GMT-06:00)", Name: "Saskatchewan"},
		{Zone: "US/Central", UTCOffset: "(GMT-06:00)", Name: "Central Time (US &amp; Canada)"},
		{Zone: "US/Eastern", UTCOffset: "(GMT-05:00)", Name: "Eastern Time (US &amp; Canada)"},
		{Zone: "US/East-Indiana", UTCOffset: "(GMT-05:00)", Name: "Indiana (East)"},
		{Zone: "America/Bogota", UTCOffset: "(GMT-05:00)", Name: "Bogota"},
		{Zone: "America/Lima", UTCOffset: "(GMT-05:00)", Name: "Lima"},
		{Zone: "America/Caracas", UTCOffset: "(GMT-04:30)", Name: "Caracas"},
		{Zone: "Canada/Atlantic", UTCOffset: "(GMT-04:00)", Name: "Atlantic Time (Canada)"},
		{Zone: "America/La_Paz", UTCOffset: "(GMT-04:00)", Name: "La_Paz"},
		{Zone: "America/Santiago", UTCOffset: "(GMT-04:00)", Name: "Santiago"},
		{Zone: "Canada/Newfoundland", UTCOffset: "(GMT-03:30)", Name: "Newfoundland"},
		{Zone: "America/Buenos_Aires", UTCOffset: "(GMT-03:00)", Name: "Buenos Aires"},
		{Zone: "Greenland", UTCOffset: "(GMT-03:00)", Name: "Greenland"},
		{Zone: "Atlantic/Stanley", UTCOffset: "(GMT-02:00)", Name: "Stanley"},
		{Zone: "Atlantic/Azores", UTCOffset: "(GMT-01:00)", Name: "Azores"},
		{Zone: "Atlantic/Cape_Verde", UTCOffset: "(GMT-01:00)", Name: "Cape Verde Is."},
		{Zone: "Africa/Casablanca", UTCOffset: "(GMT)", Name: "Casablanca"},
		{Zone: "Europe/Dublin", UTCOffset: "(GMT)", Name: "Dublin"},
		{Zone: "Europe/Lisbon", UTCOffset: "(GMT)", Name: "Libson"},
		{Zone: "Europe/London", UTCOffset: "(GMT)", Name: "London"},
		{Zone: "Africa/Monrovia", UTCOffset: "(GMT)", Name: "Monrovia"},
		{Zone: "Europe/Amsterdam", UTCOffset: "(UTC+01:00)", Name: "Amsterdam"},
		{Zone: "Europe/Belgrade", UTCOffset: "(UTC+01:00)", Name: "Belgrade"},
		{Zone: "Europe/Berlin", UTCOffset: "(UTC+01:00)", Name: "Berlin"},
		{Zone: "Europe/Bratislava", UTCOffset: "(UTC+01:00)", Name: "Bratislava"},
		{Zone: "Europe/Brussels", UTCOffset: "(UTC+01:00)", Name: "Brussels"},
		{Zone: "Europe/Budapest", UTCOffset: "(UTC+01:00)", Name: "Budapest"},
		{Zone: "Europe/Copenhagen", UTCOffset: "(UTC+01:00)", Name: "Copenhagen"},
		{Zone: "Europe/Ljubljana", UTCOffset: "(UTC+01:00)", Name: "Ljubljana"},
		{Zone: "Europe/Madrid", UTCOffset: "(UTC+01:00)", Name: "Madrid"},
		{Zone: "Europe/Paris", UTCOffset: "(UTC+01:00)", Name: "Paris"},
		{Zone: "Europe/Prague", UTCOffset: "(UTC+01:00)", Name: "Prague"},
		{Zone: "Europe/Rome", UTCOffset: "(UTC+01:00)", Name: "Rome"},
		{Zone: "Europe/Sarajevo", UTCOffset: "(UTC+01:00)", Name: "Sarajevo"},
		{Zone: "Europe/Skopje", UTCOffset: "(UTC+01:00)", Name: "Skopje"},
		{Zone: "Europe/Stockholm", UTCOffset: "(UTC+01:00)", Name: "Stockholm"},
		{Zone: "Europe/Vienna", UTCOffset: "(UTC+01:00)", Name: "Vienna"},
		{Zone: "Europe/Warsaw", UTCOffset: "(UTC+01:00)", Name: "Warsaw"},
		{Zone: "Europe/Zagreb", UTCOffset: "(UTC+01:00)", Name: "Zagreb"},
		{Zone: "Europe/Athens", UTCOffset: "(UTC+02:00)", Name: "Athens"},
		{Zone: "Europe/Bucharest", UTCOffset: "(UTC+02:00)", Name: "Bucharest"},
		{Zone: "Africa/Cairo", UTCOffset: "(UTC+02:00)", Name: "Cairo"},
		{Zone: "Africa/Harare", UTCOffset: "(UTC+02:00)", Name: "Harere"},
		{Zone: "Europe/Helsinki", UTCOffset: "(UTC+02:00)", Name: "Helsinki"},
		{Zone: "Europe/Istanbul", UTCOffset: "(UTC+02:00)", Name: "Istanbul"},
		{Zone: "Asia/Jerusalem", UTCOffset: "(UTC+02:00)", Name: "Jerusalem"},
		{Zone: "Europe/Kiev", UTCOffset: "(UTC+02:00)", Name: "Kiev"},
		{Zone: "Europe/Minsk", UTCOffset: "(UTC+02:00)", Name: "Minsk"},
		{Zone: "Europe/Riga", UTCOffset: "(UTC+02:00)", Name: "Riga"},
		{Zone: "Europe/Sofia", UTCOffset: "(UTC+02:00)", Name: "Sofia"},
		{Zone: "Europe/Tallinn", UTCOffset: "(UTC+02:00)", Name: "Tallinn"},
		{Zone: "Europe/Vilnius", UTCOffset: "(UTC+02:00)", Name: "Vilnius"},
		{Zone: "Asia/Baghdad", UTCOffset: "(UTC+03:00)", Name: "Baghdad"},
		{Zone: "Asia/Kuwait", UTCOffset: "(UTC+03:00)", Name: "Kuwait"},
		{Zone: "Africa/Nairobi", UTCOffset: "(UTC+03:00)", Name: "Nairobi"},
		{Zone: "Asia/Riyadh", UTCOffset: "(UTC+03:00)", Name: "Riyadh"},
		{Zone: "Asia/Tehran", UTCOffset: "(UTC+03:30)", Name: "Tehran"},
		{Zone: "Europe/Moscow", UTCOffset: "(UTC+04:00)", Name: "Moscow"},
		{Zone: "Asia/Baku", UTCOffset: "(UTC+04:00)", Name: "Baku"},
		{Zone: "Europe/Volgograd", UTCOffset: "(UTC+04:00)", Name: "Volgograd"},
		{Zone: "Asia/Muscat", UTCOffset: "(UTC+04:00)", Name: "Muscat"},
		{Zone: "Asia/Tbilisi", UTCOffset: "(UTC+04:00)", Name: "Tbilisi"},
		{Zone: "Asia/Yerevan", UTCOffset: "(UTC+04:00)", Name: "Yerevan"},
		{Zone: "Asia/Kabul", UTCOffset: "(UTC+04:30)", Name: "Kabul"},
		{Zone: "Asia/Karachi", UTCOffset: "(UTC+05:00)", Name: "Karachi"},
		{Zone: "Asia/Tashkent", UTCOffset: "(UTC+05:00)", Name: "Tashkent"},
		{Zone: "Asia/Kolkata", UTCOffset: "(UTC+05:30)", Name: "Kolkata"},
		{Zone: "Asia/Kathmandu", UTCOffset: "(UTC+05:45)", Name: "Kathmandu"},
		{Zone: "Asia/Yekaterinburg", UTCOffset: "(UTC+06:00)", Name: "Yekaterinburg"},
		{Zone: "Asia/Almaty", UTCOffset: "(UTC+06:00)", Name: "Almaty"},
		{Zone: "Asia/Dhaka", UTCOffset: "(UTC+06:00)", Name: "Dhaka"},
		{Zone: "Asia/Novosibirsk", UTCOffset: "(UTC+07:00)", Name: "Novosibirsk"},
		{Zone: "Asia/Bangkok", UTCOffset: "(UTC+07:00)", Name: "Bangkok"},
		{Zone: "Asia/Jakarta", UTCOffset: "(UTC+07:00)", Name: "Jakarta"},
		{Zone: "Asia/Krasnoyarsk", UTCOffset: "(UTC+08:00)", Name: "Krasnoyarsk"},
		{Zone: "Asia/Chongqing", UTCOffset: "(UTC+08:00)", Name: "Chongqing"},
		{Zone: "Asia/Hong_Kong", UTCOffset: "(UTC+08:00)", Name: "Hong Kong"},
		{Zone: "Asia/Kuala_Lumpur", UTCOffset: "(UTC+08:00)", Name: "Kuala Lumpur"},
		{Zone: "Australia/Perth", UTCOffset: "(UTC+08:00)", Name: "Perth"},
		{Zone: "Asia/Singapore", UTCOffset: "(UTC+08:00)", Name: "Singapore"},
		{Zone: "Asia/Taipei", UTCOffset: "(UTC+08:00)", Name: "Taipei"},
		{Zone: "Asia/Ulaanbaatar", UTCOffset: "(UTC+08:00)", Name: "Ulaan Bataar"},
		{Zone: "Asia/Urumqi", UTCOffset: "(UTC+08:00)", Name: "Urumqi"},
		{Zone: "Asia/Irkutsk", UTCOffset: "(UTC+09:00)", Name: "Irkutsk"},
		{Zone: "Asia/Seoul", UTCOffset: "(UTC+09:00)", Name: "Seoul"},
		{Zone: "Asia/Tokyo", UTCOffset: "(UTC+09:00)", Name: "Tokyo"},
		{Zone: "Australia/Adelaide", UTCOffset: "(UTC+09:30)", Name: "Adelaide"},
		{Zone: "Australia/Darwin", UTCOffset: "(UTC+09:30)", Name: "Darwin"},
		{Zone: "Asia/Yakutsk", UTCOffset: "(UTC+10:00)", Name: "Yakutsk"},
		{Zone: "Australia/Brisbane", UTCOffset: "(UTC+10:00)", Name: "Brisbane"},
		{Zone: "Australia/Canberra", UTCOffset: "(UTC+10:00)", Name: "Canberra"},
		{Zone: "Pacific/Guam", UTCOffset: "(UTC+10:00)", Name: "Guam"},
		{Zone: "Australia/Hobart", UTCOffset: "(UTC+10:00)", Name: "Hobart"},
		{Zone: "Australia/Melbourne", UTCOffset: "(UTC+10:00)", Name: "Melbourne"},
		{Zone: "Pacific/Port_Moresby", UTCOffset: "(UTC+10:00)", Name: "Port Moresby"},
		{Zone: "Australia/Sydney", UTCOffset: "(UTC+10:00)", Name: "Sydney"},
		{Zone: "Asia/Vladivostok", UTCOffset: "(UTC+11:00)", Name: "Vladivostok"},
		{Zone: "Asia/Magadan", UTCOffset: "(UTC+12:00)", Name: "Magadan"},
		{Zone: "Pacific/Auckland", UTCOffset: "(UTC+12:00)", Name: "Auckland"},
		{Zone: "Pacific/Fiji", UTCOffset: "(UTC+12:00)", Name: "Fiji"},
	}

	// Create the records in the database
	result := db.CreateInBatches(timeZones, 10)
	if result.Error != nil {
		logger.Error("Failed to seed time_zone table: %s", result.Error.Error())
		return result.Error
	}

	logger.Info("Successfully seeded time_zone table with %d records", len(timeZones))
	return nil
}
