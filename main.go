package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"gorm.io/gorm"
)

type FileRecord struct {
	Code         string `gorm:"primaryKey"`
	OriginalName string
	Size         int64
	MimeType     string
	FilePath     string
	BurnAfter    bool
	ExpiresAt    time.Time
	CreatedAt    time.Time
}

var db *gorm.DB

func main() {
	var err error
	if err := os.MkdirAll("./data", 0755); err != nil {
		log.Fatal(err)
	}

	db, err = gorm.Open(sqlite.Open("data/beam.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	db.AutoMigrate(&FileRecord{})

	app := fiber.New(fiber.Config{
		BodyLimit: 2 * 1024 * 1024 * 1024, // 2GB Max Upload
		AppName:   "Beam v1.0",
	})

	app.Use(logger.New())
	app.Use(cors.New())

	if err := os.MkdirAll("./uploads", 0755); err != nil {
		log.Fatal(err)
	}

	app.Static("/", "./public")

	api := app.Group("/api")
	api.Post("/upload", handleUpload)
	api.Get("/meta/:code", handleGetMeta)
	api.Get("/download/:code", handleDownload)

	go startCleanupRoutine()

	log.Println("Beam is live on http://localhost:3000")
	log.Fatal(app.Listen(":3000"))
}

func handleUpload(c *fiber.Ctx) error {
	file, err := c.FormFile("document")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "No file received"})
	}

	burnAfter := c.FormValue("burn_after") == "true"
	expireStr := c.FormValue("expire_in")

	duration := time.Hour
	if expireStr == "10m" {
		duration = 10 * time.Minute
	} else if expireStr == "24h" {
		duration = 24 * time.Hour
	}

	if burnAfter {
		log.Println("Upload marked for Burn After Reading")
	}

	var code string
	for {
		code = fmt.Sprintf("%04d", rand.Intn(10000))
		var count int64
		db.Model(&FileRecord{}).Where("code = ?", code).Count(&count)
		if count == 0 {
			break
		}
	}

	ext := filepath.Ext(file.Filename)
	uniqueName := fmt.Sprintf("%s_%d%s", code, time.Now().Unix(), ext)
	savePath := filepath.Join("./uploads", uniqueName)

	if err := c.SaveFile(file, savePath); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to save file"})
	}

	record := FileRecord{
		Code:         code,
		OriginalName: file.Filename,
		Size:         file.Size,
		MimeType:     file.Header.Get("Content-Type"),
		FilePath:     savePath,
		BurnAfter:    burnAfter,
		ExpiresAt:    time.Now().Add(duration),
	}

	result := db.Create(&record)
	if result.Error != nil {
		os.Remove(savePath)
		return c.Status(500).JSON(fiber.Map{"error": "Database error"})
	}

	return c.JSON(fiber.Map{
		"code":    code,
		"expires": duration.String(),
	})
}

func handleGetMeta(c *fiber.Ctx) error {
	code := c.Params("code")
	var record FileRecord

	result := db.First(&record, "code = ?", code)
	if result.Error != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Invalid or expired code"})
	}

	return c.JSON(fiber.Map{
		"name":        record.OriginalName,
		"size":        record.Size,
		"type":        record.MimeType,
		"uploaded_at": record.CreatedAt,
		"burn_after":  record.BurnAfter,
	})
}

type cleanupReader struct {
	io.ReadCloser
	onClose func()
}

func (c *cleanupReader) Close() error {
	err := c.ReadCloser.Close()
	if c.onClose != nil {
		c.onClose()
	}
	return err
}

func handleDownload(c *fiber.Ctx) error {
	code := c.Params("code")
	var record FileRecord

	result := db.First(&record, "code = ?", code)
	if result.Error != nil {
		return c.Status(404).SendString("File not found")
	}

	f, err := os.Open(record.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			db.Delete(&record)
		}
		return c.Status(404).SendString("File not found on disk")
	}

	reader := &cleanupReader{
		ReadCloser: f,
		onClose: func() {
			if record.BurnAfter {
				go func(r FileRecord) {
					log.Printf("Burn timer started for: %s", r.Code)
					time.Sleep(10 * time.Second)
					db.Delete(&r)
					if rmErr := os.Remove(r.FilePath); rmErr != nil {
						log.Printf("Delete warning (might be open): %v", rmErr)
					} else {
						log.Printf("File %s burned successfully.", r.Code)
					}
				}(record)
			}
		},
	}
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, record.OriginalName))
	c.Set("Content-Length", fmt.Sprintf("%d", record.Size))
	c.Set("Content-Type", record.MimeType)
	return c.SendStream(reader)
}

func startCleanupRoutine() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		var expired []FileRecord

		db.Where("expires_at < ?", time.Now()).Find(&expired)

		for _, file := range expired {
			if err := os.Remove(file.FilePath); err != nil {
				log.Printf("Error deleting file %s: %v", file.FilePath, err)
			} else {
				log.Printf("Cleaned up expired file: %s", file.Code)
			}
			db.Delete(&file)
		}
	}
}
