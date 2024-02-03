package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	pb "ive_fyp/protos"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/mailgun/mailgun-go/v3"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/session"
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
	mailgun  MailgunInfo
}

type MailgunInfo struct {
	domain  string
	api_key string
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
	Email    string `json:"email" xml:"email" form:"email"`
	Password string `json:"password" xml:"password" form:"password"`
}

type NewUser struct {
	FirstName               string `json:"fName" xml:"fName" form:"fName"`
	LastName                string `json:"lName" xml:"lName" form:"lName"`
	Email                   string `json:"email" xml:"email" form:"email"`
	Gender                  string `json:"gender" xml:"gender" form:"gender"`
	Contact                 int    `json:"contact" xml:"contact" form:"contact"`
	Password                string `json:"password" xml:"password" form:"password"`
	EmergencyContact        int    `json:"eContact" xml:"eContact" form:"eContact"`
	EmergencyFirstName      string `json:"eFName" xml:"eFName" form:"eFName"`
	EmergencyLastName       string `json:"eLName" xml:"eLName" form:"eLName"`
	EmergencyPersonRelation string `json:"ePersonRelation" xml:"ePersonRelation" form:"ePersonRelation"`
	Position                string `json:"position" xml:"position" form:"position"`
}

// type Notification struct {
// 	CamID          string
// 	Violation_List []string
// 	Workplace      string
// 	Time           string
// }

type ForgotPassword struct {
	Email string `json:"email" xml:"email" form:"email"`
}

type ResetPasswordInfo struct {
	NewPassword string `json:"password" xml:"password" form:"password"`
}

type VerifyOtp struct {
	OTP int `json:"otp" xml:"otp" form:"otp"`
}

var (
	box_clients   []pb.AnalysisClient
	frame_clients []pb.AnalysisClient
	image_stream  []pb.Analysis_GetImageClient
	box_stream    []pb.Analysis_AnalysisClient
	frame_data    [][]byte
	box_data      [][]Box
	db            *surrealdb.DB
	config        WebConfig
	store         *session.Store
	sess          *session.Session
	box_conn      *grpc.ClientConn
	frame_conn    *grpc.ClientConn
	app           *fiber.App
	// verify_email_dict map[string]bool
	otp_dict  map[string]int
	otp_mutex *sync.Mutex
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

func setup_client_pair(cam_url string, server_url string) {
	var err error
	frame_conn, err = grpc.Dial(cam_url, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect to FrameGetter: %v", err)
	}
	box_conn, err = grpc.Dial(server_url, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	frame_client := pb.NewAnalysisClient(frame_conn)
	box_client := pb.NewAnalysisClient(box_conn)
	frame_clients = append(frame_clients, frame_client)
	box_clients = append(box_clients, box_client)
}

func get_image_data(wg *sync.WaitGroup, id int) {
	defer wg.Done()
	var err error
	image_stream[id], err = frame_clients[id].GetImage(context.Background(), &pb.Empty{}, grpc.MaxCallRecvMsgSize(1024*1024*1024))
	if err != nil {
		log.Fatalf("could not call GetImage: %v", err)
	}
	for {
		image, err := image_stream[id].Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("error receiving from GetImage stream: %v", err)
		}
		decoded_data, _ := base64.StdEncoding.DecodeString(string(image.Data[:]))
		// encoded_data:= base64.StdEncoding.EncodeToString([]byte(image.Data[:]))
		frame_data[id] = decoded_data
		// frame_data[id] = encoded_data
	}
}

func get_box_data(wg *sync.WaitGroup, id int) {
	defer wg.Done()
	box_stream[id], _ = box_clients[id].Analysis(context.Background(), &pb.Empty{}, grpc.MaxCallRecvMsgSize(1024*1024*1024))
	for {
		response, err := box_stream[id].Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("error receiving from Box Analysis stream: %v", err)
		}
		var local_box_data []Box
		for _, item := range response.Item {
			box := Box{
				X1:        int(item.X1),
				Y1:        int(item.Y1),
				X2:        int(item.X2),
				Y2:        int(item.Y2),
				ClassType: int(item.ClassType),
			}
			local_box_data = append(local_box_data, box)
		}
		box_data[id] = local_box_data
	}
}

func index(c *fiber.Ctx) error {
	var err error
	unauthorized_message := ""
	if c.QueryBool("error") {
		unauthorized_message = "Invalid email or password"
	}
	if c.QueryBool("unauth") {
		unauthorized_message = "You must login first!!!"
	}

	sess, err = store.Get(c)
	if err != nil {
		panic(err)
	}

	email := sess.Get("email")
	isLogin := email != nil

	return c.Render("index", fiber.Map{
		"request":             c,
		"isLogin":             isLogin,
		"unauthorizedMessage": unauthorized_message,
	})
}

func login(c *fiber.Ctx) error {
	user := User{}
	var err error
	if err = c.BodyParser(&user); err != nil {
		return err
	}

	result, err := db.Query("SELECT 1 FROM user where email = $email and crypto::argon2::compare(password, $password);", map[string]string{
		"email":    user.Email,
		"password": user.Password,
	})
	user_found := len(result.([]interface{})[0].(map[string]interface{})["result"].([]interface{})) == 1
	if err != nil {
		panic(err)
	}
	if user_found {

		sess, err = store.Get(c)
		if err != nil {
			panic(err)
		}

		sess.Set("email", user.Email)

		if err = sess.Save(); err != nil {
			panic(err)
		}

		sess.Delete("approved_email")

		return c.Redirect("/stream")
	}
	return c.Redirect("/?error=true")
}

func create_user(c *fiber.Ctx) error {
	var err error
	sess, err = store.Get(c)
	if err != nil {
		panic(err)
	}
	if sess.Get("email") == nil {
		return c.Redirect("/?unauth=true")
	}
	return c.Render("create_user", fiber.Map{
		"request": c,
		"url":     fmt.Sprintf("http://%s", config.web.url),
	})
}

func create_user_api(c *fiber.Ctx) error {
	new_user := NewUser{}
	var err error
	if err = c.BodyParser(&new_user); err != nil {
		return err
	}
	encrypted_pass := get_encrypted_password(new_user.Password)[0].(map[string]interface{})["result"].(string)

	result, err := db.Create("user", map[string]interface{}{
		"email":                   new_user.Email,
		"password":                encrypted_pass,
		"firstName":               new_user.FirstName,
		"lastName":                new_user.LastName,
		"gender":                  new_user.Gender,
		"contact":                 new_user.Contact,
		"position":                new_user.Position,
		"emergencyContact":        new_user.EmergencyContact,
		"emergencyFirstName":      new_user.EmergencyFirstName,
		"emergencyLastName":       new_user.EmergencyLastName,
		"emergencyPersonRelation": new_user.EmergencyPersonRelation,
	})
	if err != nil {
		panic(err)
	}
	if result != nil {
		return c.Redirect("/create_user")
	}
	return c.JSON(result)
}

func get_encrypted_password(password string) []interface{} {
	result, err := db.Query("RETURN crypto::argon2::generate($password);", map[string]string{
		"password": password,
	})
	if err != nil {
		fmt.Println(err)
	}
	return result.([]interface{})
}

func get_records(c *fiber.Ctx) error {
	var err error
	sess, err = store.Get(c)
	if err != nil {
		panic(err)
	}
	if sess.Get("email") == nil {
		return c.Redirect("/?unauth=true")
	}

	return c.Render("records", fiber.Map{
		"request": c,
		"url":     fmt.Sprintf("http://%s", config.web.url),
	})
}

func get_records_api(c *fiber.Ctx) error {

	result, err := db.Query("SELECT * FROM violation_record;", map[string]string{})
	if err != nil {
		return c.SendString(fmt.Sprintf("Error: %v", err))
	}
	return c.JSON(result)
}

func get_users_list(c *fiber.Ctx) error {
	var err error
	sess, err = store.Get(c)
	if err != nil {
		panic(err)
	}
	if sess.Get("email") == nil {
		return c.Redirect("/?unauth=true")
	}
	return c.Render("users_list", fiber.Map{
		"request": c,
		"url":     fmt.Sprintf("http://%s", config.web.url),
	})
}

func get_users_api(c *fiber.Ctx) error {
	result, err := db.Query("SELECT * FROM user;", map[string]string{})
	if err != nil {
		return c.SendString(fmt.Sprintf("Error: %v", err))
	}
	return c.JSON(result)
}

func get_stream(c *fiber.Ctx) error {
	var err error
	sess, err = store.Get(c)
	if err != nil {
		panic(err)
	}

	if sess.Get("email") == nil {
		return c.Redirect("/?unauth=true")
	}
	return c.Render("stream", fiber.Map{
		"request": c,
		"url":     fmt.Sprintf("http://%s", config.web.url),
	})
}

func get_image(c *fiber.Ctx) error {
	c.Set("Content-Type", "image/jpeg")
	id := c.QueryInt("stream_source", -1)
	if id > len(box_clients)-1 || id < 0 {
		return fiber.ErrServiceUnavailable
	}
	return c.Send(frame_data[id])
}

func get_box(c *fiber.Ctx) error {
	id := c.QueryInt("stream_source", -1)
	if id > len(box_clients)-1 || id < 0 {
		return fiber.ErrServiceUnavailable
	}
	return c.JSON(box_data[id])
}

func logout(c *fiber.Ctx) error {
	var err error
	sess, err = store.Get(c)
	if err != nil {
		panic(err)
	}

	sess.Delete("email")
	// Destroy session
	if err = sess.Destroy(); err != nil {
		panic(err)
	}

	return c.Redirect("/")
}

func forgot_password(c *fiber.Ctx) error {
	return c.Render("forgot_password", fiber.Map{
		"request": c,
		"url":     fmt.Sprintf("http://%s", config.web.url),
	})
}

func verify_email_api(c *fiber.Ctx) error {
	forgot_password := ForgotPassword{}
	c.BodyParser(&forgot_password)
	result, _ := db.Query("SELECT 1 FROM user where email = $email;", map[string]string{
		"email": forgot_password.Email,
	})
	user_found := len(result.([]interface{})[0].(map[string]interface{})["result"].([]interface{})) == 1

	if !user_found {
		return c.JSON(map[string]bool{"success": false})
	}

	opt := rand.Intn(999999-100000) + 100000
	otp_mutex.Lock()
	otp_dict[forgot_password.Email] = opt
	otp_mutex.Unlock()
	sess, _ = store.Get(c)
	sess.Set("approved_email", forgot_password.Email)
	mg := mailgun.NewMailgun(config.mailgun.domain, config.mailgun.api_key)
	m := mg.NewMessage(
		fmt.Sprintf("<mailgun@%s>", config.mailgun.domain),
		"Verification Code",
		fmt.Sprintf("Your verification code is %d", otp_dict[forgot_password.Email]),
		forgot_password.Email,
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	mg.Send(ctx, m)
	sess.Save()
	return c.Redirect("/verify_forgot_otp")
}

func verify_forgot_otp(c *fiber.Ctx) error {

	sess, err := store.Get(c)
	if err != nil {
		panic(err)
	}

	approved_email := sess.Get("approved_email")

	if approved_email != nil {
		return c.Render("verify_forgot_otp", fiber.Map{
			"request": c,
			"url":     fmt.Sprintf("http://%s", config.web.url),
		})
	}
	return c.Redirect("/forgot_password")
}

func verify_otp_api(c *fiber.Ctx) error {
	sender_otp := VerifyOtp{}
	c.BodyParser(&sender_otp)
	sess, _ := store.Get(c)
	approved_mail := sess.Get("approved_email").(string)
	otp_mutex.Lock()
	otp := otp_dict[approved_mail]
	otp_mutex.Unlock()
	success := otp == sender_otp.OTP
	if success {
		delete(otp_dict, approved_mail)
	}
	return c.JSON(map[string]bool{"success": success})
}

func reset_password(c *fiber.Ctx) error {

	sess, err := store.Get(c)

	if err != nil {
		fmt.Println(err)
	}

	if sess.Get("approved_email") == nil {
		return c.Redirect("/forgot_password")
	}

	return c.Render("reset_password", fiber.Map{
		"request": c,
		"url":     fmt.Sprintf("http://%s", config.web.url),
	})
}

func reset_password_api(c *fiber.Ctx) error {
	reset_password_info := ResetPasswordInfo{}
	sess, _ = store.Get(c)
	if err := c.BodyParser(&reset_password_info); err != nil {
		return err
	}
	encrypted_password := get_encrypted_password(reset_password_info.NewPassword)[0].(map[string]interface{})["result"].(string)
	result, _ := db.Query("Update user set password=$password where email = $email", map[string]interface{}{
		"email":    sess.Get("approved_email").(string),
		"password": encrypted_password,
	})
	if result != nil {
		sess.Delete("approved_email")
		sess.Save()
	}
	return c.JSON(map[string]bool{"success": result != nil})
}

func start_server() {
	if err := app.Listen(fmt.Sprintf(":%d", config.web.port)); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}

func setup_static(app *fiber.App) {
	app.Static("/images", "./images")
	app.Static("/navigation", "./navigation")
	app.Static("/js", "./js")
	app.Static("/css", "./css")
}

func setup_get(app *fiber.App) {
	app.Get("/", index)
	app.Get("/logout", logout)
	app.Get("/stream", get_stream)
	app.Get("/image", get_image)
	app.Get("/box", get_box)
	app.Get("/records", get_records)
	app.Get("/records_api", get_records_api)
	app.Get("/create_user", create_user)
	app.Get("/users_list", get_users_list)
	app.Get("/users_list_api", get_users_api)
	app.Get("/forgot_password", forgot_password)
	app.Get("/verify_forgot_otp", verify_forgot_otp)
	app.Get("/reset_password", reset_password)
}

func setup_post(app *fiber.App) {
	app.Post("/login", login)
	app.Post("/create_user_api", create_user_api)
	app.Post("/verify_email_api", verify_email_api)
	app.Post("/verify_otp_api", verify_otp_api)
	app.Post("/reset_password_api", reset_password_api)
}

func main() {
	config = loadConfig()
	otp_mutex = new(sync.Mutex)
	otp_dict = make(map[string]int)
	setup_db()
	setup_client_pair(config.cam.url, config.server.url)
	// setup_client_pair(config.cam.tim_url, config.server.tim_url)
	defer frame_conn.Close()
	defer box_conn.Close()

	store = session.New(session.Config{
		Expiration: 60 * 60 * time.Second,
	})

	engine := html.New("./views", ".html")
	engine.Reload(true)
	// engine.Debug(true)
	engine.Layout("embed")
	engine.Delims("{{", "}}")

	app = fiber.New(fiber.Config{
		Views: engine,
	})
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowHeaders:     "*",
		AllowMethods:     "*",
		AllowCredentials: true,
	}))

	setup_static(app)
	setup_get(app)
	setup_post(app)

	go start_server()

	var wg sync.WaitGroup
	wg.Add(2 * len(box_clients))
	frame_data = make([][]byte, len(frame_clients))
	box_data = make([][]Box, len(box_clients))
	image_stream = make([]pb.Analysis_GetImageClient, len(frame_clients))
	box_stream = make([]pb.Analysis_AnalysisClient, len(box_clients))
	for i := 0; i < len(box_clients); i++ {
		go get_image_data(&wg, i)
		go get_box_data(&wg, i)
	}
	wg.Wait()
}

// get the toml config
func loadConfig() WebConfig {

	toml_data, err := toml.LoadFile("config.toml")
	if err != nil {
		log.Fatalf("Error loading config file, %s", err)
	}
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

	mailgun_info := MailgunInfo{
		domain:  toml_data.Get("mailgun").(*toml.Tree).Get("domain").(string),
		api_key: toml_data.Get("mailgun").(*toml.Tree).Get("api_key").(string),
	}

	return WebConfig{
		web:      web_info,
		cam:      cam_info,
		server:   server_info,
		database: user_info,
		mailgun:  mailgun_info,
	}
}
