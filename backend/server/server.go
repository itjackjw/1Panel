package server

import (
	"encoding/gob"
	"fmt"
	"time"

	"github.com/1Panel-dev/1Panel/backend/init/app"
	"github.com/1Panel-dev/1Panel/backend/init/business"

	"github.com/1Panel-dev/1Panel/backend/cron"
	"github.com/1Panel-dev/1Panel/backend/init/cache"
	"github.com/1Panel-dev/1Panel/backend/init/session"
	"github.com/1Panel-dev/1Panel/backend/init/session/psession"

	"github.com/1Panel-dev/1Panel/backend/global"
	"github.com/1Panel-dev/1Panel/backend/init/db"
	"github.com/1Panel-dev/1Panel/backend/init/hook"
	"github.com/1Panel-dev/1Panel/backend/init/log"
	"github.com/1Panel-dev/1Panel/backend/init/migration"
	"github.com/1Panel-dev/1Panel/backend/init/router"
	"github.com/1Panel-dev/1Panel/backend/init/validator"
	"github.com/1Panel-dev/1Panel/backend/init/viper"

	"github.com/fvbock/endless"
	"github.com/gin-gonic/gin"
)

func Start() {
	viper.Init()
	log.Init()
	app.Init()
	db.Init()
	migration.Init()
	validator.Init()
	gob.Register(psession.SessionUser{})
	cache.Init()
	session.Init()
	gin.SetMode("debug")
	cron.Run()
	business.Init()
	hook.Init()

	rootRouter := router.Routers()
	address := fmt.Sprintf(":%s", global.CONF.System.Port)
	s := initServer(address, rootRouter)
	global.LOG.Infof("server run success on %s", global.CONF.System.Port)
	if err := s.ListenAndServe(); err != nil {
		global.LOG.Error(err)
		panic(err)
	}
}

type server interface {
	ListenAndServe() error
}

func initServer(address string, router *gin.Engine) server {
	s := endless.NewServer(address, router)
	s.ReadHeaderTimeout = 20 * time.Second
	s.WriteTimeout = 60 * time.Second
	s.MaxHeaderBytes = 1 << 20
	return s
}
