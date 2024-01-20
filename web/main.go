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
	"github.com/surrealdb/surrealdb.go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type WebConfig struct {
	web      WebInfo
	cam      CamInfo
	server   ServerInfo
	database DatabaseInfo
}

type DatabaseInfo struct {
	url       string
	user      string
	password  string
	namespace string
	database  string
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

type User struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

var (
	frame_client pb.AnalysisClient
	box_client   pb.AnalysisClient
	frame_data   []byte
	box_data     []Box
	db           *surrealdb.DB
	config       WebConfig
	frame_conn   *grpc.ClientConn
	box_conn     *grpc.ClientConn
	app          *fiber.App
)

func setup_db() {
	var err error
	db, err = surrealdb.New(config.database.url)
	if err != nil {
		log.Fatalf("could not connect to SurrealDB: %v", err)
	}

	if _, err = db.Signin(map[string]interface{}{
		"user": config.database.user,
		"pass": config.database.password,
	}); err != nil {
		panic(err)
	}
	if _, err = db.Use(config.database.namespace, config.database.database); err != nil {
		panic(err)
	}
}

func setup_clients() {
	var err error
	frame_conn, err = grpc.Dial(fmt.Sprintf("%s:%d", config.cam.url, config.cam.port), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect to FrameGetter: %v", err)
	}
	frame_client = pb.NewAnalysisClient(frame_conn)
	box_conn, err = grpc.Dial(fmt.Sprintf("%s:%d", config.server.url, config.server.port), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect to BoxGetter: %v", err)
	}
	box_client = pb.NewAnalysisClient(box_conn)
}

func index(c *fiber.Ctx) error {
	err := ""
	if c.QueryBool("error") {
		err = "Invalid email or password"
	}
	return c.Render("index", fiber.Map{
		"request": c,
		"error":   err,
	})
}

func get_stream(c *fiber.Ctx) error {
	return c.Render("stream", fiber.Map{
		"request": c,
		"url":     fmt.Sprintf("https://%s", config.web.url),
	})
}

func get_records(c *fiber.Ctx) error {
	return c.Render("records", fiber.Map{
		"request": c,
		"url":     fmt.Sprintf("https://%s", config.web.url),
	})
}

func get_records_api(c *fiber.Ctx) error {
	result, err := db.Query("SELECT * FROM violation_record;", map[string]string{})
	if err != nil {
		return c.SendString(fmt.Sprintf("Error: %v", err))
	}
	return c.JSON(result)
}

func get_image(c *fiber.Ctx) error {
	c.Set("Content-Type", "image/jpeg")
	return c.Send(frame_data)
}

func get_box(c *fiber.Ctx) error {
	return c.JSON(box_data)
}

func start_server() {
	if err := app.Listen(fmt.Sprintf(":%d", config.web.port)); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}

func get_image_data(wg *sync.WaitGroup) {
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
		decoded_data, err := base64.StdEncoding.DecodeString(string(image.Data[:]))
		frame_data = decoded_data
		if err != nil {
			log.Fatalf("error decoding frame data: %v", err)
		}
	}
}

func get_box_data(wg *sync.WaitGroup) {
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
		box_data = localBoxData
	}
}

func login(c *fiber.Ctx) error {
	user := User{}
	if err := c.BodyParser(&user); err != nil {
		return err
	}
	result, err := db.Query("SELECT 1 FROM user where email = $email and password = $password;", map[string]string{
		"email":    user.Email,
		"password": user.Password,
	})
	//haha

	user_found := len(result.([]interface{})[0].(map[string]interface{})["result"].([]interface{})) == 1
	// user_test := result.([]interface{})[0].(map[string]interface{})["result"].([]interface{})[0].(map[string]interface{})["1"]
	if err != nil {
		return c.SendString(fmt.Sprintf("Error: %v", err))
	}
	if user_found {
		// c.Request().Header.Set("HX-Redirect", fmt.Sprintf("https://%s", config.web.url))
		return c.Redirect("/stream")
	}
	return c.Redirect("/?error=true")
}

func main() {
	config = loadConfig()
	setup_db()
	setup_clients()
	defer frame_conn.Close()
	defer box_conn.Close()
	engine := html.New("./views", ".html")
	app = fiber.New(fiber.Config{
		Views: engine,
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowHeaders:     "*",
		AllowMethods:     "*",
		AllowCredentials: true,
	}))

	app.Static("/images", "./images")

	app.Static("/js", "./js")

	app.Static("/css", "./css")

	app.Post("/login", login)

	app.Get("/", index)

	app.Get("/stream", get_stream)

	app.Get("/records", get_records)

	app.Get("/records_api", get_records_api)

	app.Get("/image", get_image)

	app.Get("/box", get_box)

	// Start the Fiber server
	go start_server()
	var wg sync.WaitGroup
	wg.Add(2)
	// Goroutine to fetch image data
	go get_image_data(&wg)
	// Goroutine to fetch box data
	go get_box_data(&wg)
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

	user_info := DatabaseInfo{
		url:       toml_data.Get("user_database").(*toml.Tree).Get("url").(string),
		user:      toml_data.Get("user_database").(*toml.Tree).Get("user").(string),
		password:  toml_data.Get("user_database").(*toml.Tree).Get("pass").(string),
		namespace: toml_data.Get("user_database").(*toml.Tree).Get("namespace").(string),
		database:  toml_data.Get("user_database").(*toml.Tree).Get("database").(string),
	}

	return WebConfig{
		web:      web_info,
		cam:      cam_info,
		server:   server_info,
		database: user_info,
	}
}
