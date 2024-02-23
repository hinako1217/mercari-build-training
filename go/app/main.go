package main

import (
	"crypto/sha256"
	"database/sql"
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
	_ "github.com/mattn/go-sqlite3"
)

const (
	DB_PATH = "../db/mercari.sqlite3"
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

	//画像ファイルを開く
	src, err := imagefile.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	//hash化
	h := sha256.New()
	if _, err := io.Copy(h, src); err != nil { //srcからhへ中身をコピー
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

	//データベースを開く
	db, err := sql.Open("sqlite3", DB_PATH)
	if err != nil {
		return err
	}
	defer db.Close()

	var category_id int

	//categories tableにcategoryが存在しなければ追加し、categoryのidを取得
	if err := db.QueryRow("SELECT id FROM categories WHERE name = $1", item.Category).Scan(&category_id); err != nil {
		if err == sql.ErrNoRows { //QueryRow()の結果が空のとき
			stmt1, err := db.Prepare("INSERT INTO categories (name) VALUES (?)")
			if err != nil {
				return err
			}
			defer stmt1.Close()
			if _, err = stmt1.Exec(item.Category); err != nil {
				return err
			}
			if err := db.QueryRow("SELECT id FROM categories WHERE name = $1", item.Category).Scan(&category_id); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	//items tableへ商品を追加
	stmt2, err := db.Prepare("INSERT INTO items (name, category_id, image_name) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt2.Close()

	if _, err = stmt2.Exec(item.Name, category_id, item.Image); err != nil {
		return err
	}

	c.Logger().Infof("Receive item: %s, %s, %s", item.Name, item.Category, item.Image)
	message := fmt.Sprintf("item received: %s", item.Name)
	res := Response{Message: message}

	return c.JSON(http.StatusOK, res)
}

/*
e.GET("/items". getItemList)
*/
func getItemList(c echo.Context) error {
	//データベースを開く
	db, err := sql.Open("sqlite3", DB_PATH)
	if err != nil {
		return err
	}
	defer db.Close()

	//データベースから商品を取得
	rows, err := db.Query("SELECT items.name, categories.name, items.image_name FROM items INNER JOIN categories on items.category_id = categories.id")
	if err != nil {
		return err
	}
	defer rows.Close()

	var itemlist ItemList
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.Name, &item.Category, &item.Image); err != nil {
			return err
		}
		itemlist.Items = append(itemlist.Items, item)
	}

	return c.JSON(http.StatusOK, itemlist)
}

/*
e.GET("/items/:id", getItemDetail)
*/
func getItemById(c echo.Context) error {
	var item Item
	id, err := strconv.Atoi(c.Param("id")) //string to int
	if err != nil {
		return err
	}

	//データベースを開く
	db, err := sql.Open("sqlite3", DB_PATH)
	if err != nil {
		return err
	}
	defer db.Close()

	//items tableとcategories tableをJOINし、指定したidに対応するデータを取得
	if err := db.QueryRow("SELECT items.name, categories.name, items.image_name FROM items INNER JOIN categories on items.category_id = categories.id  WHERE items.id = $1", id).Scan(&item.Name, &item.Category, &item.Image); err != nil {
		if err == sql.ErrNoRows { //QueryRow()の結果が空のとき
			return err
		} else {
			return err
		}
	}

	return c.JSON(http.StatusOK, item)
}

/*
e.GET("/search", getItemByKeyword)
*/
func getItemByKeyword(c echo.Context) error {
	//データベースを開く
	db, err := sql.Open("sqlite3", DB_PATH)
	if err != nil {
		return err
	}
	defer db.Close()

	//データベースから指定したキーワードを含む商品一覧を取得
	keyword := c.QueryParam("keyword")
	rows, err := db.Query("SELECT items.name, categories.name, items.image_name FROM items INNER JOIN categories on items.category_id = categories.id  WHERE items.name LIKE CONCAT('%', ?, '%')", keyword)
	if err != nil {
		return err
	}
	defer rows.Close()

	var itemlist ItemList
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.Name, &item.Category, &item.Image); err != nil {
			return err
		}
		itemlist.Items = append(itemlist.Items, item)
	}

	return c.JSON(http.StatusOK, itemlist)
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
	e.Logger.SetLevel(log.DEBUG) //log.INFOからlog.DEBUGに変更

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
	e.GET("/items/:id", getItemById)
	e.GET("/search", getItemByKeyword)
	e.GET("/image/:imageFilename", getImg)

	// Start server
	e.Logger.Fatal(e.Start(":9000"))
}
