package server

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	ginlogrus "github.com/toorop/gin-logrus"
)

type Server struct{}

func StartServer() error {
	s := Server{}

	router := gin.New()
	log := logrus.New()
	config := cors.DefaultConfig()
	config.AllowHeaders = append(config.AllowHeaders, "Authorization")
	config.AllowAllOrigins = true

	router.Use(ginlogrus.Logger(log), cors.New(config), gin.Recovery())

	router.POST("/tree/:language", s.generateTree)

	return router.Run(":8011")
}
