package main

import (
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.LoadHTMLGlob("./views/*")

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl", nil)
	})

	r.GET("/sequence/:current", func(c *gin.Context) {
		current, _ := strconv.Atoi(c.Param("current"))

		delay := 0
		delayMin, _ := strconv.Atoi(c.DefaultQuery("delayMin", "0"))
		delayMax, _ := strconv.Atoi(c.DefaultQuery("delayMax", "0"))
		delayDelta := delayMax - delayMin

		if delayDelta > 0 {
			rand.Seed(time.Now().UnixNano())
			delay = rand.Intn(delayDelta) + delayMin

			log.Println("delay", delay)
			time.Sleep(time.Millisecond * time.Duration(delay))
		}

		c.HTML(http.StatusOK, "sequence.tmpl", gin.H{
			"current": current,
			"next":    current + 1,
			"delay":   delay,
		})
	})

	r.Run()
}
