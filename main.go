package main

import (
	"bytes"
	"fmt"
	"html/template"
	"httelper/cloudmailin2"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

func mailRedisKey(key string, inbox string) string {
	return fmt.Sprintf("httelper:mail:v1:%s:%s", inbox, key)
}

func main() {
	redisURL, ok := os.LookupEnv("REDIS_URL")
	if !ok {
		redisURL = "redis://localhost:6379/0"
	}

	redisOpts, err := redis.ParseURL(redisURL)
	if err != nil {
		panic(err)
	}
	redis := redis.NewClient(&redis.Options{
		Addr:     redisOpts.Addr,
		Password: redisOpts.Password,
	})

	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = "8080"
	}

	r := gin.Default()
	r.LoadHTMLGlob("./views/*")

	r.GET("/healthz", func(c *gin.Context) {
		_, err := redis.Info(c).Result()
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
		} else {
			c.String(http.StatusOK, "ok")
		}
	})

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl", nil)
	})

	r.GET("/mail/clear", func(c *gin.Context) {
		redis.Del(c, "httelper:mail:v1:inbox")

		c.String(http.StatusOK, "cleared")
	})

	r.POST("/mail/cloudmailin/:inbox", func(c *gin.Context) {
		var buf bytes.Buffer
		tee := io.TeeReader(c.Request.Body, &buf)

		body, _ := ioutil.ReadAll(tee)
		fmt.Println(string(body))

		msg, err := cloudmailin2.Decode(&buf)
		if err != nil {
			panic(err)
		}
		redis.LPush(c, mailRedisKey("queue", c.Param("inbox")), msg.HTML)
		c.String(http.StatusOK, "ok")
	})

	r.POST("/mail/raw/:inbox", func(c *gin.Context) {
		bodyBytes, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			panic(err)
		}

		fmt.Println("body", string(bodyBytes))
		redis.LPush(c, mailRedisKey("queue", c.Param("inbox")), string(bodyBytes))
	})

	r.GET("/mail/unlock/:inbox", func(c *gin.Context) {
		redis.Set(c, mailRedisKey("status", c.Param("inbox")), "unlocked", 0)
		c.String(http.StatusOK, "unlocked")
	})

	r.GET("/mail/lock/:inbox", func(c *gin.Context) {
		redis.Set(c, mailRedisKey("status", c.Param("inbox")), "locked", 0)
		c.String(http.StatusOK, "locked")
	})

	r.GET("/mail/status/:inbox", func(c *gin.Context) {
		status, _ := redis.Get(c, mailRedisKey("status", c.Param("inbox"))).Result()
		c.String(http.StatusOK, status)
	})

	r.GET("/mail/next/:inbox", func(c *gin.Context) {
		mode := c.DefaultQuery("mode", "peek")
		status, _ := redis.Get(c, mailRedisKey("status", c.Param("inbox"))).Result()

		if status != "unlocked" {
			c.String(http.StatusLocked, status)
			return
		}

		message := ""
		switch mode {
		case "peek":
			message, _ = redis.LIndex(c, mailRedisKey("queue", c.Param("inbox")), 0).Result()
		case "pop":
			message, _ = redis.RPop(c, mailRedisKey("queue", c.Param("inbox"))).Result()
		default:
			panic("unknown mode " + mode)
		}

		if message == "" {
			c.HTML(http.StatusNotFound, "mail_next.tmpl", gin.H{
				"body": template.HTML("<h1>no mail</h1>"),
			})
		} else {
			c.HTML(http.StatusOK, "mail_next.tmpl", gin.H{
				"body": template.HTML(message),
			})
		}
	})

	r.GET("/sequence/:current", func(c *gin.Context) {
		current, _ := strconv.Atoi(c.Param("current"))
		delay := 0
		delayMin := 0
		delayMax := 0
		clockDelay := c.Query("clockDelay")

		if clockDelay != "" {
			hour, min, sec := time.Now().Clock()

			var delayF float64
			switch clockDelay {
			case "seconds":
				delayF = float64(sec) / float64(60) * 5 * 1000
			case "minutes":
				delayF = float64(min) / float64(60) * 5 * 1000
			case "hours":
				delayF = float64(hour) / float64(60) * 5 * 1000
			}
			delay = int(delayF)
			time.Sleep(time.Millisecond * time.Duration(delay))
		} else {
			delayMin, _ := strconv.Atoi(c.DefaultQuery("delayMin", "0"))
			delayMax, _ := strconv.Atoi(c.DefaultQuery("delayMax", "0"))
			delayDelta := delayMax - delayMin

			if delayDelta > 0 {
				rand.Seed(time.Now().UnixNano())
				delay = rand.Intn(delayDelta) + delayMin
				time.Sleep(time.Millisecond * time.Duration(delay))
			}
		}

		c.HTML(http.StatusOK, "sequence.tmpl", gin.H{
			"current":    current,
			"next":       current + 1,
			"delay":      delay,
			"delayMin":   delayMin,
			"delayMax":   delayMax,
			"clockDelay": clockDelay,
		})
	})

	fmt.Println("listening at :8080")
	r.Run(":" + port)
}
