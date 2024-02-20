package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

const (
	ImgDir = "images"
)

type Response struct {
	Message string `json:"message"`
}

/*
list of item
*/
type ItemList struct {
	Items []Item `json:"items"`
}

/*
name, category and image of goods
*/
type Item struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Image    string `json:"image_name"`
}

/*
e.GET("/", root)
*/
func root(c echo.Context) error {
	res := Response{Message: "Hello, world!"}
	return c.JSON(http.StatusOK, res)
}

var itemlist ItemList

/*
e.POST("/items", addItem)
*/
func addItem(c echo.Context) error {
	var item Item
	// Get form data
	item.Name = c.FormValue("name")
	item.Category = c.FormValue("category")
	imagefile, err := c.FormFile("image")
	if err != nil {
		return err
	}

	c.Logger().Infof("Receive item: %s, %s, %s", item.Name, item.Category, imagefile)
	message := fmt.Sprintf("item received: %s", item.Name)
	res := Response{Message: message}
	//画像ファイルを開く
	src, err := imagefile.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	//hash化
	h := sha256.New()
	if _, err := io.Copy(h, src); err != nil { //srcからhへ中身をコピー
		//log.Fatal(err)
		return err
	}
	str_hash_sha256 := fmt.Sprintf("%x", h.Sum(nil))
	item.Image = str_hash_sha256 + ".jpg"

	//imagesフォルダに画像ファイルを作成
	dst, err := os.Create(fmt.Sprintf("images/%s", item.Image))
	if err != nil {
		return err
	}
	defer dst.Close()
	if _, err = io.Copy(dst, src); err != nil { //srcからdstへ中身をコピー
		return err
	}

	// add item to list
	itemlist.Items = append(itemlist.Items, item)

	//open file  if it doesn't exist, create file
	jsonfile, err := os.OpenFile("items.json", os.O_WRONLY|os.O_CREATE, 0664)
	if err != nil {
		log.Fatal(err)
	}
	defer jsonfile.Close()

	//encode
	encoder := json.NewEncoder(jsonfile)
	if err := encoder.Encode(itemlist); err != nil {
		log.Fatal(err)
	}

	return c.JSON(http.StatusOK, res)
}

func decodeJson() ItemList {
	jsonfile, err := os.Open("items.json")
	if err != nil {
		log.Fatal(err)
	}
	defer jsonfile.Close()

	//decode
	var itemlist ItemList
	decoder := json.NewDecoder(jsonfile)
	if err := decoder.Decode(&itemlist); err != nil {
		log.Fatal(err)
	}

	return itemlist
}

/*
e.GET("/items". getItemList)
*/
func getItemList(c echo.Context) error {
	itemlist := decodeJson()

	return c.JSON(http.StatusOK, itemlist)
}

/*
e.GET("/items/:id", getItemDetail)
*/
func getItemDetail(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id")) //string to int
	if err != nil {
		log.Fatal(err)
	}
	itemlist := decodeJson()

	if id <= 0 || id > len(itemlist.Items) {
		log.Fatal(err)
	}

	return c.JSON(http.StatusOK, itemlist.Items[id-1])
}

/*
e.GET("/image/:imageFilename", getImg)
*/
func getImg(c echo.Context) error {
	// Create image path
	imgPath := path.Join(ImgDir, c.Param("imageFilename"))

	if !strings.HasSuffix(imgPath, ".jpg") {
		res := Response{Message: "Image path does not end with .jpg"}
		return c.JSON(http.StatusBadRequest, res)
	}
	if _, err := os.Stat(imgPath); err != nil {
		c.Logger().Debugf("Image not found: %s", imgPath)
		imgPath = path.Join(ImgDir, "default.jpg")
	}
	return c.File(imgPath)
}

func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Logger.SetLevel(log.INFO)

	frontURL := os.Getenv("FRONT_URL")
	if frontURL == "" {
		frontURL = "http://localhost:3000"
	}
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{frontURL},
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
	}))

	// Routes
	e.GET("/", root)
	e.POST("/items", addItem)
	e.GET("/items", getItemList)
	e.GET("/items/:id", getItemDetail)
	e.GET("/image/:imageFilename", getImg)

	// Start server
	e.Logger.Fatal(e.Start(":9000"))
}
