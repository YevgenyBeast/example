package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	application "task/app"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app := application.NewApp(ctx)
	go app.Start(ctx)

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	<-c

	app.Stop(ctx)
}
