package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	pb "ive_fyp/protos"
	"log"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/template/html/v2"
	"github.com/pelletier/go-toml"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type WebConfig struct {
	web    WebInfo
	cam    CamInfo
	server ServerInfo
}

type WebInfo struct {
	url  string
	port int64
}

type CamInfo struct {
	url  string
	port int64
}

type ServerInfo struct {
	url  string
	port int64
}

type Box struct {
	X1        int `json:"x1"`
	Y1        int `json:"y1"`
	X2        int `json:"x2"`
	Y2        int `json:"y2"`
	ClassType int `json:"class_type"`
}

var (
	frame_client pb.AnalysisClient
	box_client   pb.AnalysisClient
	frame_data   []byte
	box_data     []Box
	mu           sync.Mutex // Mutex to protect access to frame_data and box_data
)

func main() {
	config := loadConfig()
	var wg sync.WaitGroup

	// Set up a connection to the FrameGetter server.
	frame_conn, err := grpc.Dial(fmt.Sprintf("%s:%d", config.cam.url, config.cam.port), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect to FrameGetter: %v", err)
	}
	defer frame_conn.Close()
	frame_client = pb.NewAnalysisClient(frame_conn)

	// Set up a connection to the BoxGetter server.

	box_conn, err := grpc.Dial(fmt.Sprintf("%s:%d", config.server.url, config.server.port), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect to BoxGetter: %v", err)
	}
	defer box_conn.Close()
	box_client = pb.NewAnalysisClient(box_conn)

	// Initialize the Fiber app

	engine := html.New("./views", ".html")

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowHeaders:     "*",
		AllowMethods:     "*",
		AllowCredentials: true,
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Render("index", fiber.Map{
			"request": c,
			"url":     fmt.Sprintf("http://%s:%d", config.web.url, config.web.port),
		})
	})

	app.Get("/image", func(c *fiber.Ctx) error {
		mu.Lock()
		defer mu.Unlock()
		c.Set("Content-Type", "image/jpeg")
		return c.Send(frame_data)

	})

	app.Get("/box", func(c *fiber.Ctx) error {
		mu.Lock()
		defer mu.Unlock()
		return c.JSON(box_data)
	})

	// Start the Fiber server
	go func() {
		if err := app.Listen(fmt.Sprintf(":%d", config.web.port)); err != nil {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	wg.Add(2)

	// Goroutine to fetch image data
	go func() {
		defer wg.Done()
		imageStream, err := frame_client.GetImage(context.Background(), &pb.Empty{})
		if err != nil {
			log.Fatalf("could not call GetImage: %v", err)
		}
		for {
			image, err := imageStream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatalf("error receiving from GetImage stream: %v", err)
			}
			mu.Lock()
			decoded_data, err := base64.StdEncoding.DecodeString(string(image.Data[:]))
			frame_data = decoded_data
			if err != nil {
				log.Fatalf("error decoding frame data: %v", err)
			}
			mu.Unlock()
		}
	}()

	// Goroutine to fetch box data
	go func() {
		defer wg.Done()
		stream, err := box_client.Analysis(context.Background(), &pb.Empty{})
		if err != nil {
			log.Fatalf("could not call Analysis: %v", err)
		}
		for {
			response, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatalf("error receiving from Analysis stream: %v", err)
			}
			var localBoxData []Box
			for _, item := range response.Item {

				box := Box{
					X1:        int(item.X1),
					Y1:        int(item.Y1),
					X2:        int(item.X2),
					Y2:        int(item.Y2),
					ClassType: int(item.ClassType),
				}
				localBoxData = append(localBoxData, box)
			}
			mu.Lock()
			box_data = localBoxData
			mu.Unlock()
		}
	}()

	wg.Wait()
}
// get the toml config 
func loadConfig() WebConfig {
	toml_data, _ := toml.LoadFile("config.toml")
	web_info := WebInfo{
		url:  toml_data.Get("web").(*toml.Tree).Get("url").(string),
		port: toml_data.Get("web").(*toml.Tree).Get("port").(int64),
	}
	cam_info := CamInfo{
		url:  toml_data.Get("cam").(*toml.Tree).Get("url").(string),
		port: toml_data.Get("cam").(*toml.Tree).Get("port").(int64),
	}
	server_info := ServerInfo{
		url:  toml_data.Get("server").(*toml.Tree).Get("url").(string),
		port: toml_data.Get("server").(*toml.Tree).Get("port").(int64),
	}
	return WebConfig{
		web:    web_info,
		cam:    cam_info,
		server: server_info,
	}
}
