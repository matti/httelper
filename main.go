package main

import (
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

	r.Run()
}
