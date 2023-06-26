package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/joho/godotenv"
	"github.com/sclevine/agouti"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type YahooNews struct {
	Title   string      `json:"title"`
	Time    string      `json:"time"`
	Content string      `json:"content"`
	Images  []NewsImage `json:"images"`
	Source  string      `json:"source"`
	Link    string      `json:"link"`
}

type NewsImage struct {
	Link        string
	Description string
}

type MongoConfig struct {
	Username string
	Password string
	Host     string
	Port     string
}

type S3Config struct {
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
}

var (
	coll     *mongo.Collection
	svc      *s3.S3
	s3Config S3Config
	imageDir string = "files/download"
)

func init() {
	link1 := "https://tw.stock.yahoo.com/news/%E6%95%99%E5%AD%B8-%E5%85%A8%E6%94%AF%E4%BB%98%E6%98%AF%E4%BB%80%E9%BA%BC-%E5%A6%82%E4%BD%95%E8%A8%BB%E5%86%8A%E7%B6%81%E5%AE%9A-2022%E5%84%AA%E6%83%A0%E6%B4%BB%E5%8B%95-%E7%B6%81%E9%8A%80%E8%A1%8C%E5%B8%B3%E6%88%B6-041900616.html"

	driver := agouti.ChromeDriver()

	if errs := driver.Start(); errs != nil {
		log.Fatal("Failed to start ChromeDriver:", errs)
	}
	defer driver.Stop()

	page, errs := driver.NewPage(agouti.Browser("chrome"))
	if errs != nil {
		log.Fatal("Failed to open page:", errs)
	}

	if errs := page.Navigate(link1); errs != nil {
		log.Fatal("Failed to navigate:", errs)
	}

	time.Sleep(5 * time.Second) // Give enough time for the page to load and images to render

	imgSrcs, errs := page.All("div.caas-body img").Attribute("src")
	if errs != nil {
		log.Fatal("Failed to retrieve image source URLs:", errs)
	}

	for _, imgSrc := range imgSrcs {
		fmt.Println(imgSrc)
	}

	err := godotenv.Load()
	if err != nil {
		log.Fatal("無法讀取 .env 檔案:", err)
	}

	mongoConfig := MongoConfig{
		Username: os.Getenv("USERNAME"),
		Password: os.Getenv("PASSWORD"),
		Host:     os.Getenv("HOST"),
		Port:     os.Getenv("PORT"),
	}

	credential := options.Credential{
		Username: "admin",
		Password: mongoConfig.Password,
	}

	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://" + mongoConfig.Host + ":" + mongoConfig.Port).SetAuth(credential))
	if err != nil {
		log.Fatal(err)
	}

	err = client.Connect(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	coll = client.Database("stock").Collection("news")

	s3Config = S3Config{
		Region:          os.Getenv("REGION"),
		Bucket:          os.Getenv("BUCKET"),
		AccessKeyID:     os.Getenv("S3ACCESSKEYID"),
		SecretAccessKey: os.Getenv("S3SECRETACCESSKEY"),
	}

	// 建立新的 AWS 會話
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(s3Config.Region),
		Credentials: credentials.NewStaticCredentials(s3Config.AccessKeyID, s3Config.SecretAccessKey, ""),
	})
	if err != nil {
		log.Fatal("Failed to create AWS session", err)
	}

	// 建立 S3 服務的客戶端
	svc = s3.New(sess)
}

func main() {
	// 要爬取的網頁 URL
	url := "https://tw.stock.yahoo.com/news/"

	// 發送 HTTP 請求並獲取響應
	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	// 使用 goquery 解析 HTML
	doc1, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	// 提取標題並輸出
	results := make(map[string]bool)
	doc1.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			linkExists := strings.Contains(href, "/news/")
			if linkExists {
				domainExists := strings.Contains(href, "https://")

				if !domainExists {
					href = "https://tw.stock.yahoo.com" + href
				}
				results[href] = true
			}
		}
	})

	var wg sync.WaitGroup
	newsList := make([]interface{}, 0)
	wg.Add(len(results))

	for link := range results {
		go func(des string) {
			defer wg.Done()
			nextRes, err := http.Get(des)
			if err != nil {
				log.Fatal(err)
			}
			defer nextRes.Body.Close()
			nextDoc, err := goquery.NewDocumentFromReader(nextRes.Body)
			if err != nil {
				log.Fatal(err)
			}
			var news YahooNews
			nextDoc.Find("h1").Each(func(i int, s *goquery.Selection) {
				newsTitle, exists := s.Attr("data-test-locator")
				if newsTitle == "headline" && exists {
					news.Title = s.Text()
				}
			})
			nextDoc.Find("noscript").Remove()
			news.Time = nextDoc.Find("time").Text()
			var newsImages []NewsImage

			// 提取圖片URL並下載
			nextDoc.Find("div.caas-body img").Each(func(i int, s *goquery.Selection) {
				imgSrc, exists := s.Attr("src")
				if !exists {
					return
				}

				s.AppendHtml("<p>{{img" + strconv.Itoa(i) + "}}<p>")
				// 下載圖片
				downloadedImage, err := downloadImage(imgSrc, imageDir)
				if err != nil {
					log.Printf("Failed to download image %s: %s", imgSrc, err)
				}

				pwd, _ := os.Getwd()
				filePath := pwd + "\\files\\download\\" + downloadedImage
				fileKey := "images/" + downloadedImage
				// 打開欲上傳的圖片檔案
				file, err := os.Open(filePath)
				if err != nil {
					log.Fatal("Failed to open file", err)
				}
				defer file.Close()

				// 設定上傳參數
				params := &s3.PutObjectInput{
					Bucket: aws.String(s3Config.Bucket),
					Key:    aws.String(fileKey),
					Body:   file,
				}

				// 上傳圖片至 S3
				_, err = svc.PutObject(params)
				if err != nil {
					log.Fatal("Failed to upload file to S3", err)
				}

				// 取得公開連結
				publicURL := fmt.Sprintf("https://%s.s3-%s.amazonaws.com/%s", s3Config.Bucket, s3Config.Region, fileKey)
				description := s.ParentsUntil("figure.caas-figure").Find("figcaption.caption-collapse").Text()
				newImage := NewsImage{
					Link:        publicURL,
					Description: description,
				}
				newsImages = append(newsImages, newImage)
			})

			news.Content = nextDoc.Find("div.caas-body").Text()
			news.Images = newsImages
			news.Source = "Yahoo"
			news.Link = des
			newsList = append(newsList, news)

		}(link)
	}
	wg.Wait()

	result, err := coll.InsertMany(context.TODO(), newsList)

	if err != nil {
		panic(err)
	}

	insertedCount := len(result.InsertedIDs)

	deleteError := deleteFilesInFolder(imageDir)
	if deleteError != nil {
		fmt.Printf("Error deleting files: %v", err)
		return
	}

	fmt.Println("Inserted count:", insertedCount)
}

// 下載圖片
func downloadImage(url string, directory string) (string, error) {
	var filePath string
	response, err := http.Get(url)
	if err != nil {
		return filePath, err
	}
	defer response.Body.Close()

	contentType := response.Header.Get("Content-Type")
	fileExtension := mimeToExtension(contentType)

	// 確定響應狀態碼為200 OK
	if response.StatusCode != http.StatusOK {
		return filePath, fmt.Errorf("received non-200 status code: %d", response.StatusCode)
	}

	// 從URL中提取圖片文件名
	fileName := filepath.Base(url) + fileExtension
	filePath = filepath.Join(directory, fileName)

	// 建立文件
	file, err := os.Create(filePath)
	if err != nil {
		return filePath, err
	}
	defer file.Close()

	// 將圖片寫入文件
	_, err = io.Copy(file, response.Body)
	if err != nil {
		return filePath, err
	}

	return fileName, err
}

func deleteFilesInFolder(folderPath string) error {
	files, err := os.ReadDir(folderPath)
	if err != nil {
		return err
	}

	for _, file := range files {
		err = os.RemoveAll(folderPath + "/" + file.Name())
		if err != nil {
			return err
		}
	}

	return nil
}

func mimeToExtension(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	default:
		return ""
	}
}
