package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	pb "ive_fyp/protos"
	"log"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/mailgun/mailgun-go/v3"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/template/html/v2"
	"github.com/pelletier/go-toml"
	"github.com/surrealdb/surrealdb.go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

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

type ForgotPassword struct {
	Email string `json:"email" xml:"email" form:"email"`
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
		log.Fatal(err)
	}
	if _, err = db.Use(config.database.namespace, config.database.database); err != nil {
		log.Fatal(err)
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
		frame_data[id] = decoded_data
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
	} else if c.QueryBool("unauth") {
		unauthorized_message = "You must login first!!!"
	}

	sess, err = store.Get(c)
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
	}
	if user_found {

		sess, err = store.Get(c)
		if err != nil {
			log.Fatal(err)
		}

		sess.Set("email", user.Email)

		if err = sess.Save(); err != nil {
			log.Fatal(err)
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
		log.Fatal(err)
	}
	if sess.Get("email") == nil {
		return c.Redirect("/?unauth=true", fiber.StatusSeeOther)
	}
	return c.SendFile("./views/create_user.html")
}

func create_user_api(c *fiber.Ctx) error {
	// new_user := NewUser{}
	var err error
	new_user := map[string]interface{}{}
	if err = c.BodyParser(&new_user); err != nil {
		return err
	}
	// fmt.Print(new_user)

	encrypted_pass := get_encrypted_password(new_user["password"].(string))[0].(map[string]interface{})["result"].(string)

	fmt.Printf("encrypted_pass: %s", encrypted_pass)

	contact, _ := strconv.ParseInt((new_user["contact"].(string)), 10, 64)

	eContact, _ := strconv.ParseInt((new_user["eContact"].(string)), 10, 64)

	create_user_result, _ := db.Create("user", map[string]interface{}{
		"email":                   new_user["email"].(string),
		"password":                encrypted_pass,
		"firstName":               new_user["fName"].(string),
		"lastName":                new_user["lName"].(string),
		"gender":                  new_user["gender"].(string),
		"contact":                 contact,
		"position":                new_user["position"].(string),
		"emergencyContact":        eContact,
		"emergencyFirstName":      new_user["eFName"].(string),
		"emergencyLastName":       new_user["eLName"].(string),
		"emergencyPersonRelation": new_user["ePersonRelation"].(string),
	})

	if create_user_result == nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	setProfileResult, _ := db.Create("setting", map[string]interface{}{
		"email": new_user["email"].(string),
	})

	if setProfileResult == nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	setHardhatRoleResult, _ := db.Create("hardhatRole", map[string]interface{}{
		"email": new_user["email"].(string),
	})

	if setHardhatRoleResult == nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	return c.SendStatus(fiber.StatusOK)
}

func update_user_api(c *fiber.Ctx) error {

	var err error
	sess, err = store.Get(c)
	if err != nil {
		log.Fatal(err)
	}
	if sess.Get("email") == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	userInfo := map[string]interface{}{}
	if err = c.BodyParser(&userInfo); err != nil {
		return err
	}

	contact, _ := strconv.ParseInt((userInfo["contact"].(string)), 10, 64)

	eContact, _ := strconv.ParseInt((userInfo["eContact"].(string)), 10, 64)

	result, _ := db.Query("Update user set contact = $contact, emergencyContact = $emergencyContact, position = $position, emergencyPersonRelation = $emergencyPersonRelation where email = $email", map[string]interface{}{
		"email":                   userInfo["email"].(string),
		"contact":                 contact,
		"emergencyContact":        eContact,
		"position":                userInfo["position"].(string),
		"emergencyPersonRelation": userInfo["ePersonRelation"].(string),
	})

	if result != nil {
		return c.SendStatus(fiber.StatusOK)
	}

	return c.SendStatus(fiber.StatusBadRequest)
}

func delete_user_api(c *fiber.Ctx) error {
	var err error
	sess, err = store.Get(c)
	if err != nil {
		log.Fatal(err)
	}
	if sess.Get("email") == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	userInfo := map[string]interface{}{}

	if err = c.BodyParser(&userInfo); err != nil {
		return err
	}
	fmt.Print(userInfo)

	user_result, _ := db.Query("DELETE FROM user where email = $email", map[string]interface{}{
		"email": userInfo["email"].(string),
	})

	if user_result == nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	setting_result, _ := db.Query("DELETE FROM setting where email = $email", map[string]interface{}{
		"email": userInfo["email"].(string),
	})

	if setting_result == nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	hardhat_role_result, _ := db.Query("DELETE FROM hardhatRole where email = $email", map[string]interface{}{
		"email": userInfo["email"].(string),
	})

	if hardhat_role_result == nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	return c.SendStatus(fiber.StatusOK)
}

func get_encrypted_password(password string) []interface{} {
	result, err := db.Query("RETURN crypto::argon2::generate($password);", map[string]string{
		"password": password,
	})
	if err != nil {
		log.Fatal(err)
	}
	return result.([]interface{})
}

func get_records(c *fiber.Ctx) error {
	var err error
	sess, err = store.Get(c)
	if err != nil {
		log.Fatal(err)
	}
	if sess.Get("email") == nil {
		return c.Redirect("/?unauth=true", fiber.StatusSeeOther)
	}
	return c.SendFile("./views/records.html")
}

func get_records_api(c *fiber.Ctx) error {
	result, err := db.Query("SELECT * FROM violation_record;", map[string]string{})
	if err != nil {
		return c.SendString(fmt.Sprintf("Error: %v", err))
	}
	return c.JSON(result)
}

// func get_warning_year_count_api(c *fiber.Ctx) error {
// 	result, err := db.Query("Select month, count(select * from violation_record where time::month(time) = $parent.month and time::year(time) = time::year(time::now())) from (array::distinct((select time::month(time) as month from violation_record where time::year(time) = time::year(time::now()))));", map[string]string{})
// 	if err != nil {
// 		return c.SendString(fmt.Sprintf("Error: %v", err))
// 	}
// 	return c.JSON(result)
// }

func get_warning_count_filter_api(c *fiber.Ctx) error {

	var err error
	sess, err = store.Get(c)
	if err != nil {
		log.Fatal(err)
	}
	if sess.Get("email") == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	condition_1 := c.Query("condition_1") //hour or day or month
	condition_2 := c.Query("condition_2") // year or month or day

	// fmt.Printf("condition_1: %s, condition_2: %s", condition_1, condition_2)

	if condition_1 != "hour" && condition_1 != "day" && condition_1 != "month" {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	if condition_2 != "year" && condition_2 != "month" && condition_2 != "day" {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	result, err := db.Query(fmt.Sprintf("Select %s, count(select * from violation_record where time::%s(time) = $parent.%s and time::year(time) = time::year(time::now())) from (array::distinct((select time::%s(time) as %s from violation_record where time::%s(time) = time::%s(time::now()))));", condition_1, condition_1, condition_1, condition_1, condition_1, condition_2, condition_2), map[string]string{})
	if err != nil {
		return c.SendString(fmt.Sprintf("Error: %v", err))
	}

	return c.JSON(result)
}

func get_warning_day_count_api(c *fiber.Ctx) error {
	result, err := db.Query("COUNT(SELECT * FROM violation_record WHERE time::day(time) = time::day(time::now()));", map[string]string{})
	if err != nil {
		return c.SendString(fmt.Sprintf("Error: %v", err))
	}
	return c.JSON(result)
}

func warning_record_filter(c *fiber.Ctx) error {
	var err error
	sess, err = store.Get(c)
	if err != nil {
		log.Fatal(err)
	}
	if sess.Get("email") == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	durationString := c.Query("duration")

	if durationString != "day" && durationString != "month" && durationString != "year" {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	result, _ := db.Query(fmt.Sprintf("SELECT * FROM violation_record WHERE time::%s(time) = %s;", durationString, fmt.Sprintf("time::%s(time::now())", durationString)), map[string]string{})

	if err != nil {
		return c.SendString(fmt.Sprintf("Error: %v", err))
	}

	return c.JSON(result)
}

func get_users_list(c *fiber.Ctx) error {
	var err error
	sess, err = store.Get(c)
	if err != nil {
		log.Fatal(err)
	}
	if sess.Get("email") == nil {
		return c.Redirect("/?unauth=true", fiber.StatusSeeOther)
	}
	return c.SendFile("./views/users_list.html")
}

func get_users_api(c *fiber.Ctx) error {

	var err error
	sess, err = store.Get(c)
	if err != nil {
		log.Fatal(err)
	}
	if sess.Get("email") == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	result, err := db.Query("SELECT firstName, lastName, gender, email, contact, position, emergencyContact, emergencyPersonRelation FROM user;", map[string]string{})
	if err != nil {
		log.Fatal(err)
	}

	return c.JSON(result)
}

func get_stream(c *fiber.Ctx) error {
	var err error
	sess, err = store.Get(c)
	if err != nil {
		log.Fatal(err)
	}

	if sess.Get("email") == nil {
		return c.Redirect("/?unauth=true", fiber.StatusSeeOther)
	}
	return c.Render("stream", fiber.Map{
		"web_server_url":          fmt.Sprintf(config.web.url),
		"notification_server_url": fmt.Sprintf(config.notification.url),
	})
}

func get_notification_url_api(c *fiber.Ctx) error {
	var err error
	sess, err = store.Get(c)
	if err != nil {
		log.Fatal(err)
	}
	if sess.Get("email") == nil {
		return c.Redirect("/?unauth=true", fiber.StatusSeeOther)
	}

	return c.JSON(fmt.Sprintf(config.notification.url))

}

func get_dashboard(c *fiber.Ctx) error {
	var err error
	sess, err = store.Get(c)
	if err != nil {
		log.Fatal(err)
	}
	if sess.Get("email") == nil {
		return c.Redirect("/?unauth=true", fiber.StatusSeeOther)
	}
	return c.SendFile("./views/dashboard.html")
}

func get_setting(c *fiber.Ctx) error {
	var err error
	sess, err = store.Get(c)
	if err != nil {
		log.Fatal(err)
	}
	if sess.Get("email") == nil {
		return c.Redirect("/?unauth=true", fiber.StatusSeeOther)
	}
	return c.SendFile("./views/setting.html")
}

func logout(c *fiber.Ctx) error {
	var err error
	sess, err = store.Get(c)
	if err != nil {
		log.Fatal(err)
	}

	sess.Delete("email")

	if err = sess.Destroy(); err != nil {
		log.Fatal(err)
	}

	return c.Redirect("/")
}

func forgot_password(c *fiber.Ctx) error {
	return c.SendFile("./views/forgot_password.html")
}

func verify_email_api(c *fiber.Ctx) error {
	forgot_password := ForgotPassword{}
	c.BodyParser(&forgot_password)
	result, _ := db.Query("SELECT 1 FROM user where email = $email;", map[string]string{
		"email": forgot_password.Email,
	})
	user_found := len(result.([]interface{})[0].(map[string]interface{})["result"].([]interface{})) == 1

	if !user_found {
		return c.SendStatus(fiber.StatusUnauthorized)
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
	return c.SendStatus(fiber.StatusOK)
}

func verify_forgot_otp(c *fiber.Ctx) error {
	sess, err := store.Get(c)

	if err != nil {
		log.Fatal(err)
	}

	if sess.Get("approved_email") != nil {
		return c.SendFile("./views/verify_forgot_otp.html")
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
	if otp == sender_otp.OTP {
		delete(otp_dict, approved_mail)
		return c.SendStatus(fiber.StatusOK)
	}
	return c.SendStatus(fiber.StatusUnauthorized)
}

func reset_password(c *fiber.Ctx) error {
	sess, err := store.Get(c)

	if err != nil {
		log.Fatal(err)
	}

	if sess.Get("approved_email") == nil {
		return c.Redirect("/forgot_password")
	}

	return c.SendFile("./views/reset_password.html")
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
		return c.SendStatus(fiber.StatusOK)
	}
	return c.SendStatus(fiber.StatusUnauthorized)
}

func get_helment_role(c *fiber.Ctx) error {
	var err error
	sess, err = store.Get(c)
	if err != nil {
		log.Fatal(err)
	}
	if sess.Get("email") == nil {
		return c.Redirect("/?unauth=true", fiber.StatusSeeOther)
	}
	result, _ := db.Query("SELECT * FROM hardhatRole where email = $email", map[string]string{
		"email": sess.Get("email").(string),
	})
	if result != nil {
		return c.JSON(result)
	}
	return c.SendStatus(fiber.StatusBadRequest)
}

func get_user_profile_setting_api(c *fiber.Ctx) error {
	var err error
	sess, err = store.Get(c)
	if err != nil {
		log.Fatal(err)
	}
	if sess.Get("email") == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	result, _ := db.Query("SELECT profileSetting, notificaitonProfileSetting FROM setting where email = $email", map[string]string{
		"email": sess.Get("email").(string),
	})
	if result != nil {
		return c.JSON(result)
	}
	return c.SendStatus(fiber.StatusUnauthorized)
}

func set_user_profile_helment_role_api(c *fiber.Ctx) error {
	var err error
	sess, err = store.Get(c)
	if err != nil {
		log.Fatal(err)
	}
	if sess.Get("email") == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	helment_role := map[int]interface{}{}
	if err = c.BodyParser(&helment_role); err != nil {
		return err
	}

	result, _ := db.Query("Update hardhatRole set role =$role where email = $email", map[string]interface{}{
		"email": sess.Get("email").(string),
		"role":  helment_role,
	})
	if result != nil {
		return c.SendStatus(fiber.StatusOK)
	}
	return c.SendStatus(fiber.StatusUnauthorized)
}

func set_user_profile_setting_api(c *fiber.Ctx) error {
	var err error
	sess, err = store.Get(c)
	if err != nil {
		log.Fatal(err)
	}
	if sess.Get("email") == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	user_profile_setting := map[string]interface{}{}
	if err = c.BodyParser(&user_profile_setting); err != nil {
		return err
	}

	result, _ := db.Query("Update setting set profileSetting = $regularSetting, notificaitonProfileSetting = $notificationSetting where email = $email", map[string]interface{}{
		"email":               sess.Get("email").(string),
		"regularSetting":      user_profile_setting["regularSetting"],
		"notificationSetting": user_profile_setting["notificationSetting"],
	})
	if result != nil {
		return c.SendStatus(fiber.StatusOK)
	}
	return c.SendStatus(fiber.StatusBadRequest)
}

func image_ws(c *websocket.Conn) {
	id, _ := strconv.Atoi(c.Query("stream_source"))
	if id >= len(frame_clients) || id < 0 {
		c.WriteMessage(websocket.TextMessage, []byte("No data available"))
		return
	}
	for {
		c.WriteMessage(websocket.BinaryMessage, frame_data[id])
		time.Sleep(1 * time.Millisecond)
	}
}

func box_ws(c *websocket.Conn) {
	id, _ := strconv.Atoi(c.Query("stream_source"))
	if id >= len(box_clients) || id < 0 {
		c.WriteMessage(websocket.TextMessage, []byte("No data available"))
		return
	}
	for {
		c.WriteJSON(box_data[id])
		time.Sleep(1 * time.Millisecond)
	}
}

func start_server() {
	if err := app.Listen(fmt.Sprintf("127.0.0.1:%d", config.web.port)); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}

func setup_static(app *fiber.App) {
	app.Static("/images", "./images")
	app.Static("/navigation", "./views/navigation")
	app.Static("/js", "./js")
	app.Static("/css", "./css")
}

func setup_get(app *fiber.App) {
	app.Get("/", index)
	app.Get("/logout", logout)
	app.Get("/stream", get_stream)
	app.Get("/dashboard", get_dashboard)
	app.Get("/records", get_records)
	app.Get("/records_api", get_records_api)
	app.Get("/create_user", create_user)
	app.Get("/users_list", get_users_list)
	app.Get("/users_list_api", get_users_api)
	app.Get("/get_setting_profile_api", get_user_profile_setting_api)
	app.Get("/setting", get_setting)
	app.Get("/forgot_password", forgot_password)
	app.Get("/verify_forgot_otp", verify_forgot_otp)
	app.Get("/reset_password", reset_password)
	app.Get("/image", websocket.New(image_ws))
	app.Get("/box", websocket.New(box_ws))
	app.Get("/get_helment_roles", get_helment_role)
	// app.Get("/get_warning_year_count_api", get_warning_year_count_api)
	app.Get("/get_warning_day_count_api", get_warning_day_count_api)
	// app.Get("/get_violation_month_record_api", get_violation_month_record_api)
	app.Get("/get_warning_count_filter_api", get_warning_count_filter_api)
	app.Get("/warning_record_filter", warning_record_filter)
	app.Get("/get_notification_url", get_notification_url_api)
}

func setup_post(app *fiber.App) {
	app.Post("/login", login)
	app.Post("/create_user_api", create_user_api)
	app.Post("/verify_email_api", verify_email_api)
	app.Post("/verify_otp_api", verify_otp_api)
	app.Post("/reset_password_api", reset_password_api)
	app.Post("/set_setting_profile_api", set_user_profile_setting_api)
	app.Post("/set_helment_role_api", set_user_profile_helment_role_api)
	app.Post("/update_user_api", update_user_api)
	app.Post("/delete_user_api", delete_user_api)

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

type WebConfig struct {
	web          WebInfo
	cam          CamInfo
	server       ServerInfo
	database     DatabaseInfo
	mailgun      MailgunInfo
	notification NotificationInfo
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

type NotificationInfo struct {
	url string
}

type ResetPasswordInfo struct {
	NewPassword string `json:"password" xml:"password" form:"password"`
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

	notification_info := NotificationInfo{
		url: toml_data.Get("notification").(*toml.Tree).Get("url").(string),
	}

	return WebConfig{
		web:          web_info,
		cam:          cam_info,
		server:       server_info,
		database:     user_info,
		mailgun:      mailgun_info,
		notification: notification_info,
	}
}
