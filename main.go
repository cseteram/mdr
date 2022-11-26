package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"google.golang.org/api/googleapi/transport"
	"google.golang.org/api/youtube/v3"

	"database/sql"
	_ "github.com/lib/pq"

	"github.com/cseteram/mdr/config"
)

func handleError(err error, message string) {
	if err != nil {
		log.Fatalf(message+": %v", err.Error())
	}
}

func executeWebhook(url string, username string, avatarUrl string, content string) (retres *http.Response, reterr error) {
	values := map[string]string{
		"username":   username,
		"avatar_url": avatarUrl,
		"content":    content,
	}
	jsonValue, _ := json.Marshal(values)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonValue))
	return resp, err
}

func main() {
	log.Println("===== MDR start =====")

	config, err := config.Parse("config.yaml")
	if err != nil {
		handleError(err, "Failed to parse configuration file")
	}

	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		config.Postgres.Host,
		config.Postgres.Port,
		config.Postgres.Username,
		config.Postgres.Password,
		config.Postgres.Dbname,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		handleError(err, "Failed to connect database")
	}
	log.Println("Successfully connected to database")

	client := &http.Client{
		Transport: &transport.APIKey{Key: config.Secrets.DeveloperKey},
	}

	service, err := youtube.New(client)
	if err != nil {
		handleError(err, "Failed to create new Youtube client")
	}

	for _, notification := range config.Notifications {
		name, channelId, webhookUrl := notification.Name, notification.ChannelID, notification.WebhookURL
		log.Printf("Processing the channel %s (%v)", name, channelId)

		nextPageToken := ""
		for {
			call := service.Activities.List([]string{"snippet", "contentDetails"}).
				ChannelId(channelId).
				MaxResults(50).
				PublishedAfter(time.Now().AddDate(0, 0, -3).Format(time.RFC3339))
			if nextPageToken != "" {
				call = call.PageToken(nextPageToken)
			}
			response, err := call.Do()
			handleError(err, "Failed to make API call")

			for _, item := range response.Items {
				channelId := item.Snippet.ChannelId
				channelTitle := item.Snippet.ChannelTitle
				videoId := item.ContentDetails.Upload.VideoId
				title := item.Snippet.Title
				publishedAt := item.Snippet.PublishedAt

				count := 0
				err := db.QueryRow(
					"select count(*) from youtube_videos where video_id = $1",
					videoId).Scan(&count)
				if err != nil {
					handleError(err, "Failed to get metadata from database")
				}

				if count > 0 {
					log.Printf("\tSkipping the video (%v, %v)\n", videoId, publishedAt)
					continue
				}

				id := -1
				err = db.QueryRow(
					"insert into youtube_videos (channel_id, channel_title, video_id, title, published_at) values ($1, $2, $3, $4, $5) returning id",
					channelId, channelTitle, videoId, title, publishedAt).Scan(&id)
				handleError(err, "Failed to insert metadata into database")

				if id > 0 {
					log.Printf("\tInserting the video (%v, %v) into %d\n", videoId, publishedAt, id)
				}

				nickname := config.Profile.Nickname
				avatarUrl := config.Profile.AvatarURL
				content := fmt.Sprintf("https://youtu.be/%s", videoId)

				for {
					resp, err := executeWebhook(webhookUrl, nickname, avatarUrl, content)
					if err != nil {
						handleError(err, "Failed to execute webhook")
					}

					if resp.StatusCode == 429 {
						seconds, err := time.ParseDuration(resp.Header["X-Ratelimit-Reset-After"][0] + "s")
						handleError(err, "Failed to parse rate limit from header")
						log.Printf("\tReached to rate limit: Sleep %v\n", seconds)
						time.Sleep(seconds)
					}

					break
				}
			}

			nextPageToken = response.NextPageToken
			if nextPageToken == "" {
				break
			}
		}
	}

	err = db.Close()
	if err != nil {
		handleError(err, "Failed to close connection")
	}
	log.Println("Successfully disconnected to database")

	log.Println("===== MDR end =====")
}
