package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "neo/cmds"
	"neo/cmds/group"
	"neo/core"
	"neo/database"

	"github.com/davidbyttow/govips/v2/vips"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

func init() {
	godotenv.Load()

	store.SetOSInfo("macOS", [3]uint32{10, 15, 7})
	store.DeviceProps.PlatformType = waCompanionReg.DeviceProps_SAFARI.Enum()
	store.DeviceProps.HistorySyncConfig.InlineInitialPayloadInE2EeMsg = proto.Bool(false)
	store.DeviceProps.HistorySyncConfig.RecentSyncDaysLimit = nil
	store.DeviceProps.HistorySyncConfig.FullSyncDaysLimit = nil
	store.DeviceProps.HistorySyncConfig.FullSyncSizeMbLimit = nil
}

func main() {
	vips.Startup(nil)
	defer vips.Shutdown()

	go func() {
		database.InitDB(os.Getenv("SQLITE_PATH"))
		database.AutoMigrate()
		group.LoadCache()
	}()

	dbLog := waLog.Stdout("Database", "DEBUG", true)
	ctx := context.Background()
	container, err := sqlstore.New(ctx, "sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on", os.Getenv("SESSION_PATH")), dbLog)
	if err != nil {
		panic(err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		panic(err)
	}

	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	client.ManualHistorySyncDownload = false
	client.DisableManualHistorySyncReceipt = true

	client.AddEventHandler(core.Default.EventHandler(client))

	if client.Store.ID == nil {
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
}
