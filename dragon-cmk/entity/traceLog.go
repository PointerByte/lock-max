package entity

import (
	"github.com/PointerByte/GoForge/logger/builder"
	"github.com/PointerByte/GoForge/logger/formatter"
	appCommon "github.com/PointerByte/lock-max/dragon-cmk/common"
	"github.com/spf13/viper"
)

type HandlerTrace func(process *formatter.Service)

func TraceClient(ctxLogger *builder.Context, process string) (*formatter.Service, HandlerTrace) {
	service := &formatter.Service{
		System:  viper.GetString("app.name"),
		Process: process,
		Status:  formatter.SUCCESS,
		Server:  appCommon.PostgreSQLSchema(),
	}
	ctxLogger.TraceInit(service)
	return service, ctxLogger.TraceEnd
}
