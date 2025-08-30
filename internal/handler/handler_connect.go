package handler

import (
	"context"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/store"
	"google.golang.org/protobuf/proto"
)

var (
	ErrorConnectingWhatsApp error = fmt.Errorf("Error connecting to WhatsApp.")
	ErrorTimeExpired        error = fmt.Errorf("Pairing timeout. Try again later.")
)

func (i *EventHandler) Connect(ctx context.Context) error {
	if i.Client.Store.ID == nil {
		var paringNumber *string
		var isCodeEnabled bool
		flag.BoolVar(&isCodeEnabled, "code", false, "")
		flag.Parse()
		isCodeEnabled = isCodeEnabled || i.Config.PairWithCode

		if isCodeEnabled {
			regexNum := regexp.MustCompile(`\D+`)
			args := flag.Args()
			paringNumber = proto.String(regexNum.ReplaceAllLiteralString(strings.Join(args, ""), ""))
			if len(*paringNumber) < 8 {
				for {
					var num string
					fmt.Print("Enter your WhatsApp number (e.g., 5511987654321): ")
					fmt.Scanln(&num)
					num = regexNum.ReplaceAllLiteralString(num, "")
					if len(num) < 8 {
						continue
					}
					paringNumber = proto.String(num)
					break
				}
			}
		}

		if paringNumber != nil {
			for range 3 {
				err := i.Client.Connect()
				if err != nil {
					return ErrorConnectingWhatsApp
				}
				code, err := i.Client.PairPhone(ctx, *paringNumber, true, whatsmeow.PairClientFirefox, "Firefox (Linux)")
				if err != nil {
					return err
				}
				i.Logger.Info().Msg("Your pairing code is: " + code)
				select {
				case <-time.After(time.Second * 120):
					i.Client.Disconnect()
					continue
				case err = <-i.WaitPair():
					if err != nil {
						return err
					}
				case <-ctx.Done():
					i.Client.Disconnect()
					return ErrorTimeExpired
				}
			}
		} else {
			store.DeviceProps.PlatformType = waCompanionReg.DeviceProps_ANDROID_PHONE.Enum()

			qrChannel, _ := i.Client.GetQRChannel(context.Background())
			err := i.Client.Connect()
			if err != nil {
				return ErrorConnectingWhatsApp
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute*3)
			defer cancel()

			go func() {
				for evt := range qrChannel {
					if evt.Event == "code" {
						i.Logger.Info().Msg("Scan the QR code below using WhatsApp")
						qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
					}
				}
			}()

			select {
			case err = <-i.WaitPair():
				if err != nil {
					return err
				}
			case <-ctx.Done():
				return ErrorTimeExpired
			}
		}
	} else {
		err := i.Client.Connect()
		if err != nil {
			return ErrorConnectingWhatsApp
		}
	}

	return nil
}

func (i *EventHandler) WaitAuthenticate() <-chan struct{} {
	ch := make(chan struct{})
	if !i.Client.IsLoggedIn() {
		i.authChannel = append(i.authChannel, ch)
	} else {
		go func() {
			ch <- struct{}{}
			close(ch)
		}()
	}
	return ch
}

func (i *EventHandler) WaitPair() <-chan error {
	ch := make(chan error)
	i.pairedChannel = append(i.pairedChannel, ch)
	return ch
}

func (i *EventHandler) WaitLogout() <-chan struct{} {
	ch := make(chan struct{})
	i.logoutChannel = append(i.logoutChannel, ch)
	return ch
}
