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
	url      string
	port     int64
	tim_url  string
	tim_port int64
}

type ServerInfo struct {
	url      string
	port     int64
	tim_url  string
	tim_port int64
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
	box_client       pb.AnalysisClient
	frame_client     pb.AnalysisClient
	frame_tim_client pb.TimAnalysisClient
	box_tim_client   pb.TimAnalysisClient
	image_stream     pb.Analysis_GetImageClient
	box_stream       pb.Analysis_AnalysisClient
	image_tim_stream pb.TimAnalysis_GetImageClient
	box_tim_stream   pb.TimAnalysis_AnalysisClient
	frame_data       []byte
	box_data         []Box
	frame_tim_data   []byte
	box_tim_data     []Box
	db               *surrealdb.DB
	config           WebConfig
	box_conn         *grpc.ClientConn
	frame_conn       *grpc.ClientConn
	app              *fiber.App
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

func setup_clients(cam_url string, server_url string, clientType string) {
	var err error

	frame_conn, err = grpc.Dial(cam_url, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect to FrameGetter: %v", err)
	}
	switch clientType {
	case "tim":
		frame_tim_client = pb.NewTimAnalysisClient(frame_conn)
	case "jason":
		frame_client = pb.NewAnalysisClient(frame_conn)
	default:
		log.Fatalf("Invalid client type: %s", clientType)
	}

	box_conn, err = grpc.Dial(server_url, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	switch clientType {
	case "tim":
		box_tim_client = pb.NewTimAnalysisClient(box_conn)
	case "jason":
		box_client = pb.NewAnalysisClient(box_conn)
	default:
		log.Fatalf("Invalid client type: %s", clientType)
	}

}

// type Client interface {
// 	GetImage(ctx context.Context, in *pb.Empty, opts ...grpc.CallOption) (pb.Analysis_GetImageClient, error)
// 	Analysis(ctx context.Context, in *pb.Empty, opts ...grpc.CallOption) (pb.Analysis_AnalysisClient, error)
// }

// func get_image_data(wg *sync.WaitGroup, client Client, data *[]byte) {
// 	defer wg.Done()
// 	stream, err := client.GetImage(context.Background(), &pb.Empty{})
// 	if err != nil {
// 		log.Printf("could not call GetImage: %v", err)
// 		return
// 	}
// 	for {
// 		image, err := stream.Recv()
// 		if err == io.EOF {
// 			break
// 		}
// 		if err != nil {
// 			log.Printf("error receiving from GetImage stream: %v", err)
// 			return
// 		}
// 		decoded_data, err := base64.StdEncoding.DecodeString(string(image.Data[:]))
// 		if err != nil {
// 			log.Printf("error decoding frame data: %v", err)
// 			return
// 		}
// 		*data = decoded_data
// 	}
// }

func get_image_data(wg *sync.WaitGroup, clientType string, client interface{}, stream pb.Analysis_GetImageClient, data *[]byte) {
	defer wg.Done()
	var err error
	switch clientType {
	case "tim":
		tim_client, ok := client.(pb.TimAnalysisClient)
		if !ok {
			log.Fatalf("Invalid client type for Tim: %s", clientType)
		}
		image_tim_stream, err = tim_client.GetImage(context.Background(), &pb.Empty{})
	case "jason":
		client, ok := client.(pb.AnalysisClient)
		if !ok {
			log.Fatalf("Invalid client type for Jason: %s", clientType)
		}
		image_stream, err = client.GetImage(context.Background(), &pb.Empty{})
	default:
		log.Fatalf("Invalid client type: %s", clientType)
	}

	if err != nil {
		log.Fatalf("could not call GetImage: %v", err)
	}
	for {
		var image *pb.Image
		switch clientType {
		case "tim":
			image, err = image_tim_stream.Recv()
		case "jason":
			image, err = image_stream.Recv()
		default:
			log.Fatalf("Invalid client type: %s", clientType)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("error receiving from GetImage stream: %v", err)
		}
		decoded_data, err := base64.StdEncoding.DecodeString(string(image.Data[:]))
		if err != nil {
			log.Fatalf("error decoding frame data: %v", err)
		}
		*data = decoded_data
	}
}

// func get_box_data(wg *sync.WaitGroup, client Client, data *[]Box) {
// 	defer wg.Done()
// 	stream, err := client.Analysis(context.Background(), &pb.Empty{})
// 	if err != nil {
// 		log.Printf("could not call GetBox: %v", err)
// 		return
// 	}
// 	for {
// 		var localBoxData []Box
// 		response, err := stream.Recv()
// 		if err == io.EOF {
// 			break
// 		}
// 		if err != nil {
// 			log.Printf("error receiving from Box Analysis stream: %v", err)
// 			return
// 		}
// 		for _, item := range response.Item {
// 			box := Box{
// 				X1:        int(item.X1),
// 				Y1:        int(item.Y1),
// 				X2:        int(item.X2),
// 				Y2:        int(item.Y2),
// 				ClassType: int(item.ClassType),
// 			}
// 			localBoxData = append(localBoxData, box)
// 		}
// 		*data = localBoxData
// 	}
// }

func get_box_data(wg *sync.WaitGroup, clientType string, client interface{}, stream pb.Analysis_AnalysisClient, data *[]Box) {
	defer wg.Done()
	var err error

	switch clientType {
	case "tim":
		box_tim_client, ok := client.(pb.TimAnalysisClient)
		if !ok {
			log.Fatalf("Invalid client type for Tim: %s", clientType)
		}
		box_tim_stream, err = box_tim_client.Analysis(context.Background(), &pb.Empty{})
	case "jason":
		box_client, ok := client.(pb.AnalysisClient)
		if !ok {
			log.Fatalf("Invalid client type for Jason: %s", clientType)
		}
		box_stream, err = box_client.Analysis(context.Background(), &pb.Empty{})
	default:
		log.Fatalf("Invalid client type: %s", clientType)
	}
	if err != nil {
		log.Fatalf("could not call GetBox: %v", err)
	}
	for {
		var localBoxData []Box
		switch clientType {
		case "tim":
			response, err := box_tim_stream.Recv()
			if err != nil {
				log.Fatalf("error receiving from Box Analysis stream: %v", err)
			}
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
		case "jason":
			response, err := box_stream.Recv()
			if err != nil {
				log.Fatalf("error receiving from Box Analysis stream: %v", err)
			}
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
		default:
			log.Fatalf("Invalid client type: %s", clientType)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("error receiving from Analysis stream: %v", err)
		}
		*data = localBoxData
	}
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

func get_stream(c *fiber.Ctx) error {
	return c.Render("stream", fiber.Map{
		"request": c,
		"url":     fmt.Sprintf("https://%s", config.web.url),
	})
}

func get_image(c *fiber.Ctx, clientType string) error {
	c.Set("Content-Type", "image/jpeg")
	switch clientType {
	case "tim":
		return c.Send(frame_tim_data)
	case "jason":
		return c.Send(frame_data)
	default:
		log.Fatalf("Invalid client type: %s", clientType)
	}
	return nil
}

func get_box(c *fiber.Ctx, clientType string) error {
	switch clientType {
	case "tim":
		return c.JSON(box_tim_data)
	case "jason":
		return c.JSON(box_data)
	default:
		log.Fatalf("Invalid client type: %s", clientType)
	}
	return nil
}

// func get_tim_box(c *fiber.Ctx) error {
// 	return c.JSON(box_tim_data)
// }

func start_server() {
	if err := app.Listen(fmt.Sprintf(":%d", config.web.port)); err != nil {
		log.Fatalf("Error starting server: %v", err)
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
	user_found := len(result.([]interface{})) == 1
	if err != nil {
		return c.SendString(fmt.Sprintf("Error: %v", err))
	}
	if user_found {
		return c.Redirect("/stream")
	}
	return c.Redirect("/?error=true")
}

func main() {
	config = loadConfig()
	setup_db()
	//tim_client
	setup_clients(config.cam.tim_url, config.server.tim_url, "tim")
	//client
	setup_clients(config.cam.url, config.server.url, "jason")
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

	// app.Get("/tim_image", get_tim_image)
	// app.Get("/tim_box", get_tim_box)

	app.Get("/tim_image", func(c *fiber.Ctx) error {
		return get_image(c, "tim")
	})
	app.Get("/tim_box", func(c *fiber.Ctx) error {
		return get_box(c, "tim")
	})

	app.Get("/image", func(c *fiber.Ctx) error {
		return get_image(c, "jason")
	})
	app.Get("/box", func(c *fiber.Ctx) error {
		return get_box(c, "jason")
	})

	// Start the Fiber server
	go start_server()

	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutines to fetch data

	// go get_image_data(&wg, "tim")
	// go get_box_data(&wg, "tim")

	// go get_image_data(&wg, "tim", frame_tim_client, image_tim_stream, &frame_tim_data)
	// go get_box_data(&wg, "tim", box_tim_client, box_tim_stream, &box_tim_data)

	go get_image_data(&wg, "jason", frame_client, image_stream, &frame_data)
	go get_box_data(&wg, "jason", box_client, box_stream, &box_data)

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
		url:      toml_data.Get("cam").(*toml.Tree).Get("url").(string),
		port:     toml_data.Get("cam").(*toml.Tree).Get("port").(int64),
		tim_url:  toml_data.Get("tim_cam").(*toml.Tree).Get("url").(string),
		tim_port: toml_data.Get("tim_cam").(*toml.Tree).Get("port").(int64),
	}
	server_info := ServerInfo{
		url:      toml_data.Get("server").(*toml.Tree).Get("url").(string),
		port:     toml_data.Get("server").(*toml.Tree).Get("port").(int64),
		tim_url:  toml_data.Get("tim_server").(*toml.Tree).Get("url").(string),
		tim_port: toml_data.Get("tim_server").(*toml.Tree).Get("port").(int64),
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
