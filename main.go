package main

import (
	"context"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
	"time"

	"os"
	"os/signal"
	"syscall"
)

var hbdlist = map[string]*Reminder{}

//Ex: var hbdlist = map[string]*Reminder{"62xxxxxxxxx": &Reminder{"08-10", false}}
//last field must be false

const MSG_FORMAT = "Selamat ulang tahun ya!"

type Reminder struct {
	date string
	what bool
}

func main() {
	dbLog := waLog.Stdout("Database", "INFO", true)
	// Make sure you add appropriate DB connector imports, e.g. github.com/mattn/go-sqlite3 for SQLite
	container, err := sqlstore.New("sqlite3", "file:contact.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}
	// If you want multiple sessions, remember their JIDs and use .GetDevice(jid) or .GetAllDevices() instead.
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}
	clientLog := waLog.Stdout("Client", "INFO", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	if client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				err = qrcode.WriteFile(evt.Code, qrcode.Medium, 256, "scan.png")
				if err != nil {
					panic(err)
				}
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}

	go func() {
		for {
			//not effective
			if client.IsConnected() {
				for no, reminder := range hbdlist {
					if time.Now().Format("02-01") == reminder.date {
						if !reminder.what {
							fmt.Printf("Hari ini %v ulang tahun!\n", no)
							_, err := client.SendMessage(context.Background(), types.JID{User: no, Server: types.DefaultUserServer}, "", &waProto.Message{Conversation: proto.String(MSG_FORMAT)})
							if err != nil {
								panic(err)
							}
							reminder.what = true
						}
					}
				}
			}
			refresh()
		}
	}()

	// Listen to Ctrl+C (you can also do something else that prevents the program from exiting)
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
}

func refresh() {
	for _, reminder := range hbdlist {
		if time.Now().Format("02-01") != reminder.date {
			reminder.what = false
		}
	}
}
