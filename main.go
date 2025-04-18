package main

import (
	"embed"
	"log"
	"net/http"
	"path"
	"runtime"
	"runtime/debug"
	"time"

	"jarvist/internal/common/buildinfo"
	"jarvist/internal/common/config"
	"jarvist/internal/common/database"
	"jarvist/internal/common/ffmpeg"
	applicationservice "jarvist/internal/wails/services/application"
	"jarvist/internal/wails/services/camera"
	configservice "jarvist/internal/wails/services/config"
	"jarvist/internal/wails/services/device"
	licenseservice "jarvist/internal/wails/services/license"
	"jarvist/internal/wails/services/location"
	"jarvist/internal/wails/services/logmanager"
	"jarvist/internal/wails/services/processmanager"
	"jarvist/internal/wails/services/servicemanager"
	"jarvist/internal/wails/services/setting"
	"jarvist/internal/wails/services/site"
	"jarvist/internal/wails/services/stats"
	"jarvist/internal/wails/services/stream"
	"jarvist/internal/wails/services/update"
	"jarvist/pkg/logger"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

var (
	defaultLicenseKey  = "dev_test_license_key_not_for_production"
	defaultLicenseSalt = "dev_test_salt_not_for_production"
	buildMode          = "development"
)

// createSPAHandler membuat handler HTTP untuk Single Page Application
func createSPAHandler(assets embed.FS) http.Handler {
	baseHandler := application.AssetFileServerFS(assets)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if buildMode == "development" {
			baseHandler.ServeHTTP(w, r)
			return
		}

		if ext := path.Ext(r.URL.Path); ext != "" {
			baseHandler.ServeHTTP(w, r)
			return
		}

		r.URL.Path = "/index.html"
		baseHandler.ServeHTTP(w, r)
	})
}

func main() {
	// ==========================================
	// Inisialisasi Konfigurasi dan Database
	// ==========================================

	buildInfoService := buildinfo.NewBuildInfoService()

	appConfig, err := config.LoadConfig(buildMode, buildInfoService)
	if err != nil {
		log.Fatal("Failed to load config: ", err)
	}

	logOptions := logger.DefaultOptions()
	logOptions.LogDir = appConfig.LogDir
	logOptions.EnableDatabase = true
	logOptions.EnableMQTT = true

	if buildMode == "production" {
		logOptions.FileMinLevel = logger.LevelWarn
		logOptions.DbMinLevel = logger.LevelWarn
		logOptions.MQTTMinLevel = logger.LevelWarn
	} else {
		logOptions.FileMinLevel = logger.LevelInfo
		logOptions.DbMinLevel = logger.LevelWarn
		logOptions.MQTTMinLevel = logger.LevelInfo
	}

	appLogger := logger.New(logOptions)

	// Setup database
	err = database.SetupDatabase(appConfig, appLogger.WithComponent("database"))
	if err != nil {
		log.Fatal("Failed to setup database: ", err)
	}
	defer database.CloseDatabase()

	appLogger.SetDB(database.GetDB())

	// Setup FFmpeg
	if err := ffmpeg.SetupFFmpeg(appConfig, appLogger); err != nil {
		log.Printf("Warning: Failed to setup FFmpeg: %v", err)
	}

	// Run database migrations
	if err := database.RunMigrations(appLogger.WithComponent("database")); err != nil {
		log.Fatal("Failed to run migrations: ", err)
	}

	// ==========================================
	// Inisialisasi Services
	// ==========================================
	deviceService := device.New()
	licenseService := licenseservice.New(appConfig, appLogger.WithComponent("licenseservice"), defaultLicenseKey, defaultLicenseSalt)
	settingService := setting.New(database.GetDB(), appConfig, appLogger.WithComponent("settingservice"), licenseService)
	siteService := site.New(database.GetDB(), appConfig, appLogger.WithComponent("siteservice"))
	appService := applicationservice.New(nil)
	locationService := location.New(database.GetDB())
	updateService := update.New(appConfig)
	processManagerService := processmanager.New(appConfig, appLogger.WithComponent("processmanagerservice"))
	cameraService := camera.New(database.GetDB(), settingService, appConfig, appLogger.WithComponent("cameraservice"), processManagerService)
	streamService := stream.New()
	statsService := stats.New(appConfig)
	serviceManager := servicemanager.New(appConfig, appLogger)

	// ==========================================
	// Inisialisasi Aplikasi Wails
	// ==========================================
	app := application.New(application.Options{
		Name:        appConfig.AppName,
		Description: "Jarvist Application",
		Services: []application.Service{
			application.NewService(appService),
			application.NewService(buildInfoService),
			application.NewService(deviceService),
			application.NewService(licenseService),
			application.NewService(settingService),
			application.NewService(siteService),
			application.NewService(configservice.New(appConfig, appLogger.WithComponent("configservice"))),
			application.NewService(cameraService),
			application.NewService(locationService),
			application.NewService(updateService),
			application.NewService(processManagerService),
			application.NewService(streamService),
			application.NewService(statsService),
			application.NewService(logmanager.New(appConfig)),
			application.NewService(serviceManager),
		},
		Assets: application.AssetOptions{
			Handler: createSPAHandler(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// Set app ke service-service yang membutuhkan
	appService.InitService(app)
	cameraService.InitService(app)
	updateService.InitService(app)
	processManagerService.InitService(app)
	streamService.InitService(app)
	statsService.InitService(app)

	// ==========================================
	// Setup Window
	// ==========================================

	// Window Splash Screen
	splashWindow := app.NewWebviewWindowWithOptions(application.WebviewWindowOptions{
		Title:            "Splash Screen",
		Width:            780,
		Height:           520,
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
		Frameless:        true,
		AlwaysOnTop:      false,
	})

	// Window Utama
	mainWindow := app.NewWebviewWindowWithOptions(application.WebviewWindowOptions{
		Title:            "Main Window",
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/dashboard",
		Width:            1166,
		Height:           768,
		Frameless:        true,
		DisableResize:    true,
	})
	mainWindow.Center()
	mainWindow.Hide()

	// Window Aktivasi
	activationWindow := app.NewWebviewWindowWithOptions(application.WebviewWindowOptions{
		Title:            "Activation Window",
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/activation",
		Width:            500,
		Height:           750,
		Frameless:        true,
		DisableResize:    true,
	})
	activationWindow.Center()
	activationWindow.Hide()

	// Window Konfigurasi
	configWindow := app.NewWebviewWindowWithOptions(application.WebviewWindowOptions{
		Title:            "Config Window",
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/config",
		Width:            500,
		Height:           750,
		Frameless:        true,
		DisableResize:    true,
	})
	configWindow.Center()
	configWindow.Hide()

	// ==========================================
	// Setup Event Handlers
	// ==========================================
	app.OnApplicationEvent(events.Common.ApplicationStarted, func(event *application.ApplicationEvent) {
		// Mulai pemeriksaan kamera
		cameraService.SetCheckInterval(3 * time.Minute)
		cameraService.StartBackgroundChecking()

		processManagerService.CheckRunningProcesses()

		for i := range 3 {
			_, err := streamService.StartStream()
			if err == nil {
				break
			}

			log.Printf("Attempt %d: Failed to start stream: %v", i+1, err)
			time.Sleep(1 * time.Second)
		}

		if buildMode == "production" {
			if err := updateService.InstallPendingUpdates(); err != nil {
				log.Printf("Error checking pending updates: %v\n", err)
			}
		}
	})

	// ==========================================
	// Background Tasks
	// ==========================================
	// Timer untuk update waktu
	go func() {
		for {
			now := time.Now().Format(time.RFC1123)
			app.EmitEvent("time", now)
			time.Sleep(time.Second)
		}
	}()

	// Cek lisensi dan tampilkan window yang sesuai
	go func() {
		time.Sleep(10 * time.Second)

		if licenseService.IsLicensed() && settingService.IsConfigured() {
			// Sudah berlisensi dan terkonfigurasi - tampilkan window utama
			mainWindow.Show()
			splashWindow.Close()
		} else if licenseService.IsLicensed() && !settingService.IsConfigured() {
			// Berlisensi tapi belum terkonfigurasi
			splashWindow.Close()
			configWindow.Show()
		} else {
			// Belum berlisensi
			splashWindow.Close()
			activationWindow.Show()
		}
	}()

	if settingService.IsConfigured() {
		result, err := serviceManager.CheckAndInstallService()
		if err != nil {
			app.Logger.Error("Failed to check/install service: " + err.Error())
		} else {
			app.Logger.Info("Service check result: " + result)
			isRunning, err := serviceManager.IsServiceRunning()
			if err != nil {
				app.Logger.Error("Failed to check if service is running: " + err.Error())
			} else if !isRunning {
				app.Logger.Info("Service is not running, starting it...")
				startResult, err := serviceManager.StartService()
				if err != nil {
					app.Logger.Error("Failed to start service: " + err.Error())
				} else {
					app.Logger.Info("Service start result: " + startResult)
				}
			} else {
				app.Logger.Info("Service is already running")
			}
		}
	}

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			<-ticker.C
			runtime.GC()
			debug.FreeOSMemory()
		}
	}()

	// ==========================================
	// Jalankan Aplikasi
	// ==========================================
	err = app.Run()
	if err != nil {
		appLogger.Error("Application error: %s", err.Error())
		log.Fatal(err)
	}

	// ==========================================
	// Cleanup Resources
	// ==========================================
	cameraService.StopBackgroundChecking()
	ffmpeg.KillAllProcesses()
	database.CloseDatabase()
}
