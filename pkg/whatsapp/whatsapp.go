package whatsapp

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"runtime"
	"strings"

	qrCode "github.com/skip2/go-qrcode"
	"google.golang.org/protobuf/proto"

	"go.mau.fi/whatsmeow"
	wabin "go.mau.fi/whatsmeow/binary"
	waproto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"

	"github.com/dimaskiddo/go-whatsapp-multidevice-rest/pkg/env"
	"github.com/dimaskiddo/go-whatsapp-multidevice-rest/pkg/log"
)

var WhatsAppDatastore *sqlstore.Container
var WhatsAppClient = make(map[string]*whatsmeow.Client)

var (
	WhatsAppClientProxyURL string
)

func init() {
	var err error

	dbType, err := env.GetEnvString("WHATSAPP_DATASTORE_TYPE")
	if err != nil {
		log.Print(nil).Fatal("Error Parse Environment Variable for WhatsApp Client Datastore Type")
	}

	dbURI, err := env.GetEnvString("WHATSAPP_DATASTORE_URI")
	if err != nil {
		log.Print(nil).Fatal("Error Parse Environment Variable for WhatsApp Client Datastore URI")
	}

	datastore, err := sqlstore.New(dbType, dbURI, nil)
	if err != nil {
		log.Print(nil).Fatal("Error Connect WhatsApp Client Datastore")
	}

	WhatsAppClientProxyURL, _ = env.GetEnvString("WHATSAPP_CLIENT_PROXY_URL")

	WhatsAppDatastore = datastore
}

func WhatsAppInitClient(device *store.Device, jid string) {
	var err error
	wabin.IndentXML = true

	if WhatsAppClient[jid] == nil {
		if device == nil {
			// Initialize New WhatsApp Client Device in Datastore
			device = WhatsAppDatastore.NewDevice()
		}

		// Set Client Properties
		store.DeviceProps.Os = proto.String(WhatsAppGetUserOS())
		store.DeviceProps.PlatformType = WhatsAppGetUserAgent("chrome").Enum()
		store.DeviceProps.RequireFullSync = proto.Bool(false)

		// Set Client Versions
		version.Major, err = env.GetEnvInt("WHATSAPP_VERSION_MAJOR")
		if err == nil {
			store.DeviceProps.Version.Primary = proto.Uint32(uint32(version.Major))
		}
		version.Minor, err = env.GetEnvInt("WHATSAPP_VERSION_MINOR")
		if err == nil {
			store.DeviceProps.Version.Secondary = proto.Uint32(uint32(version.Minor))
		}
		version.Patch, err = env.GetEnvInt("WHATSAPP_VERSION_PATCH")
		if err == nil {
			store.DeviceProps.Version.Tertiary = proto.Uint32(uint32(version.Patch))
		}

		// Initialize New WhatsApp Client
		// And Save it to The Map
		WhatsAppClient[jid] = whatsmeow.NewClient(device, nil)

		// Set WhatsApp Client Proxy Address if Proxy URL is Provided
		if len(WhatsAppClientProxyURL) > 0 {
			WhatsAppClient[jid].SetProxyAddress(WhatsAppClientProxyURL)
		}

		// Set WhatsApp Client Auto Reconnect
		WhatsAppClient[jid].EnableAutoReconnect = true

		// Set WhatsApp Client Auto Trust Identity
		WhatsAppClient[jid].AutoTrustIdentity = true

		// Disable Self Broadcast
		WhatsAppClient[jid].DontSendSelfBroadcast = true
	}
}

func WhatsAppGetUserAgent(agentType string) waproto.DeviceProps_PlatformType {
	switch strings.ToLower(agentType) {
	case "desktop":
		return waproto.DeviceProps_DESKTOP
	case "mac":
		return waproto.DeviceProps_CATALINA
	case "android":
		return waproto.DeviceProps_ANDROID_AMBIGUOUS
	case "android-phone":
		return waproto.DeviceProps_ANDROID_PHONE
	case "andorid-tablet":
		return waproto.DeviceProps_ANDROID_TABLET
	case "ios-phone":
		return waproto.DeviceProps_IOS_PHONE
	case "ios-catalyst":
		return waproto.DeviceProps_IOS_CATALYST
	case "ipad":
		return waproto.DeviceProps_IPAD
	case "wearos":
		return waproto.DeviceProps_WEAR_OS
	case "ie":
		return waproto.DeviceProps_IE
	case "edge":
		return waproto.DeviceProps_EDGE
	case "chrome":
		return waproto.DeviceProps_CHROME
	case "safari":
		return waproto.DeviceProps_SAFARI
	case "firefox":
		return waproto.DeviceProps_FIREFOX
	case "opera":
		return waproto.DeviceProps_OPERA
	case "uwp":
		return waproto.DeviceProps_UWP
	case "aloha":
		return waproto.DeviceProps_ALOHA
	case "tv-tcl":
		return waproto.DeviceProps_TCL_TV
	default:
		return waproto.DeviceProps_UNKNOWN
	}
}

func WhatsAppGetUserOS() string {
	switch runtime.GOOS {
	case "windows":
		return "Windows"
	case "darwin":
		return "macOS"
	default:
		return "Linux"
	}
}

func WhatsAppGenerateQR(qrChan <-chan whatsmeow.QRChannelItem) (string, int) {
	qrChanCode := make(chan string)
	qrChanTimeout := make(chan int)

	// Get QR Code Data and Timeout
	go func() {
		for evt := range qrChan {
			if evt.Event == "code" {
				qrChanCode <- evt.Code
				qrChanTimeout <- int(evt.Timeout.Seconds())
			}
		}
	}()

	// Generate QR Code Data to PNG Image
	qrTemp := <-qrChanCode
	qrPNG, _ := qrCode.Encode(qrTemp, qrCode.Medium, 256)

	// Return QR Code PNG in Base64 Format and Timeout Information
	return base64.StdEncoding.EncodeToString(qrPNG), <-qrChanTimeout
}

func WhatsAppLogin(jid string) (string, int, error) {
	if WhatsAppClient[jid] != nil {
		// Make Sure WebSocket Connection is Disconnected
		WhatsAppClient[jid].Disconnect()

		if WhatsAppClient[jid].Store.ID == nil {
			// Device ID is not Exist
			// Generate QR Code
			qrChanGenerate, _ := WhatsAppClient[jid].GetQRChannel(context.Background())

			// Connect WebSocket while Initialize QR Code Data to be Sent
			err := WhatsAppClient[jid].Connect()
			if err != nil {
				return "", 0, err
			}

			// Get Generated QR Code and Timeout Information
			qrImage, qrTimeout := WhatsAppGenerateQR(qrChanGenerate)

			// Return QR Code in Base64 Format and Timeout Information
			return "data:image/png;base64," + qrImage, qrTimeout, nil
		} else {
			// Device ID is Exist
			// Reconnect WebSocket
			err := WhatsAppReconnect(jid)
			if err != nil {
				return "", 0, err
			}

			return "WhatsApp Client is Reconnected", 0, nil
		}
	}

	// Return Error WhatsApp Client is not Valid
	return "", 0, errors.New("WhatsApp Client is not Valid")
}

func WhatsAppLoginPair(jid string) (string, int, error) {
	if WhatsAppClient[jid] != nil {
		// Make Sure WebSocket Connection is Disconnected
		WhatsAppClient[jid].Disconnect()

		if WhatsAppClient[jid].Store.ID == nil {
			// Connect WebSocket while also Requesting Pairing Code
			err := WhatsAppClient[jid].Connect()
			if err != nil {
				return "", 0, err
			}

			// Request Pairing Code
			code, err := WhatsAppClient[jid].PairPhone(jid, true, whatsmeow.PairClientChrome, "Chrome ("+WhatsAppGetUserOS()+")")
			if err != nil {
				return "", 0, err
			}

			return code, 160, nil
		} else {
			// Device ID is Exist
			// Reconnect WebSocket
			err := WhatsAppReconnect(jid)
			if err != nil {
				return "", 0, err
			}

			return "WhatsApp Client is Reconnected", 0, nil
		}
	}

	// Return Error WhatsApp Client is not Valid
	return "", 0, errors.New("WhatsApp Client is not Valid")
}

func WhatsAppReconnect(jid string) error {
	if WhatsAppClient[jid] != nil {
		// Make Sure WebSocket Connection is Disconnected
		WhatsAppClient[jid].Disconnect()

		// Make Sure Store ID is not Empty
		// To do Reconnection
		if WhatsAppClient[jid] != nil {
			err := WhatsAppClient[jid].Connect()
			if err != nil {
				return err
			}

			return nil
		}

		return errors.New("WhatsApp Client Store ID is Empty, Please Re-Login and Scan QR Code Again")
	}

	return errors.New("WhatsApp Client is not Valid")
}

func WhatsAppLogout(jid string) error {
	if WhatsAppClient[jid] != nil {
		// Make Sure Store ID is not Empty
		if WhatsAppClient[jid] != nil {
			var err error

			// Set WhatsApp Client Presence to Unavailable
			WhatsAppPresence(jid, false)

			// Logout WhatsApp Client and Disconnect from WebSocket
			err = WhatsAppClient[jid].Logout()
			if err != nil {
				// Force Disconnect
				WhatsAppClient[jid].Disconnect()

				// Manually Delete Device from Datastore Store
				err = WhatsAppClient[jid].Store.Delete()
				if err != nil {
					return err
				}
			}

			// Free WhatsApp Client Map
			WhatsAppClient[jid] = nil
			delete(WhatsAppClient, jid)

			return nil
		}

		return errors.New("WhatsApp Client Store ID is Empty, Please Re-Login and Scan QR Code Again")
	}

	// Return Error WhatsApp Client is not Valid
	return errors.New("WhatsApp Client is not Valid")
}

func WhatsAppIsClientOK(jid string) error {
	// Make Sure WhatsApp Client is Connected
	if !WhatsAppClient[jid].IsConnected() {
		return errors.New("WhatsApp Client is not Connected")
	}

	// Make Sure WhatsApp Client is Logged In
	if !WhatsAppClient[jid].IsLoggedIn() {
		return errors.New("WhatsApp Client is not Logged In")
	}

	return nil
}

func WhatsAppGetJID(jid string, id string) types.JID {
	if WhatsAppClient[jid] != nil {
		var ids []string

		ids = append(ids, "+"+id)
		infos, err := WhatsAppClient[jid].IsOnWhatsApp(ids)
		if err == nil {
			// If WhatsApp ID is Registered Then
			// Return ID Information
			if infos[0].IsIn {
				return infos[0].JID
			}
		}
	}

	// Return Empty ID Information
	return types.EmptyJID
}

func WhatsAppCheckJID(jid string, id string) (types.JID, error) {
	if WhatsAppClient[jid] != nil {
		// Compose New Remote JID
		remoteJID := WhatsAppComposeJID(id)
		if remoteJID.Server != types.GroupServer {
			// Validate JID if Remote JID is not Group JID
			if WhatsAppGetJID(jid, remoteJID.String()).IsEmpty() {
				return types.EmptyJID, errors.New("WhatsApp Personal ID is Not Registered")
			}
		}

		// Return Remote ID Information
		return remoteJID, nil
	}

	// Return Empty ID Information
	return types.EmptyJID, nil
}

func WhatsAppComposeJID(id string) types.JID {
	// Decompose WhatsApp ID First Before Recomposing
	id = WhatsAppDecomposeJID(id)

	// Check if ID is Group or Not By Detecting '-' for Old Group ID
	// Or By ID Length That Should be 18 Digits or More
	if strings.ContainsRune(id, '-') || len(id) >= 18 {
		// Return New Group User JID
		return types.NewJID(id, types.GroupServer)
	}

	// Return New Standard User JID
	return types.NewJID(id, types.DefaultUserServer)
}

func WhatsAppDecomposeJID(id string) string {
	// Check if WhatsApp ID Contains '@' Symbol
	if strings.ContainsRune(id, '@') {
		// Split WhatsApp ID Based on '@' Symbol
		// and Get Only The First Section Before The Symbol
		buffers := strings.Split(id, "@")
		id = buffers[0]
	}

	// Check if WhatsApp ID First Character is '+' Symbol
	if id[0] == '+' {
		// Remove '+' Symbol from WhatsApp ID
		id = id[1:]
	}

	return id
}

func WhatsAppPresence(jid string, isAvailable bool) {
	if isAvailable {
		_ = WhatsAppClient[jid].SendPresence(types.PresenceAvailable)
	} else {
		_ = WhatsAppClient[jid].SendPresence(types.PresenceUnavailable)
	}
}

func WhatsAppComposeStatus(jid string, rjid types.JID, isComposing bool, isAudio bool) {
	// Set Compose Status
	var typeCompose types.ChatPresence
	if isComposing {
		typeCompose = types.ChatPresenceComposing
	} else {
		typeCompose = types.ChatPresencePaused
	}

	// Set Compose Media Audio (Recording) or Text (Typing)
	var typeComposeMedia types.ChatPresenceMedia
	if isAudio {
		typeComposeMedia = types.ChatPresenceMediaAudio
	} else {
		typeComposeMedia = types.ChatPresenceMediaText
	}

	// Send Chat Compose Status
	_ = WhatsAppClient[jid].SendChatPresence(rjid, typeCompose, typeComposeMedia)
}

func WhatsAppCheckRegistered(jid string, id string) error {
	if WhatsAppClient[jid] != nil {
		var err error

		// Make Sure WhatsApp Client is OK
		err = WhatsAppIsClientOK(jid)
		if err != nil {
			return err
		}

		// Make Sure WhatsApp ID is Registered
		remoteJID, err := WhatsAppCheckJID(jid, id)
		if err != nil {
			return err
		}

		// Make Sure WhatsApp ID is Not Empty or It is Not Group ID
		if remoteJID.IsEmpty() || remoteJID.Server == types.GroupServer {
			return errors.New("WhatsApp Personal ID is Not Registered")
		}

		return nil
	}

	// Return Error WhatsApp Client is not Valid
	return errors.New("WhatsApp Client is not Valid")
}

func WhatsAppSendText(ctx context.Context, jid string, rjid string, message string) (string, error) {
	if WhatsAppClient[jid] != nil {
		var err error

		// Make Sure WhatsApp Client is OK
		err = WhatsAppIsClientOK(jid)
		if err != nil {
			return "", err
		}

		// Make Sure WhatsApp ID is Registered
		remoteJID, err := WhatsAppCheckJID(jid, rjid)
		if err != nil {
			return "", err
		}

		// Set Chat Presence
		WhatsAppPresence(jid, true)
		WhatsAppComposeStatus(jid, remoteJID, true, false)
		defer func() {
			WhatsAppComposeStatus(jid, remoteJID, false, false)
			WhatsAppPresence(jid, false)
		}()

		// Compose WhatsApp Proto
		msgExtra := whatsmeow.SendRequestExtra{
			ID: WhatsAppClient[jid].GenerateMessageID(),
		}
		msgContent := &waproto.Message{
			Conversation: proto.String(message),
		}

		// Send WhatsApp Message Proto
		_, err = WhatsAppClient[jid].SendMessage(ctx, remoteJID, msgContent, msgExtra)
		if err != nil {
			return "", err
		}

		return msgExtra.ID, nil
	}

	// Return Error WhatsApp Client is not Valid
	return "", errors.New("WhatsApp Client is not Valid")
}

func WhatsAppSendLocation(ctx context.Context, jid string, rjid string, latitude float64, longitude float64) (string, error) {
	if WhatsAppClient[jid] != nil {
		var err error

		// Make Sure WhatsApp Client is OK
		err = WhatsAppIsClientOK(jid)
		if err != nil {
			return "", err
		}

		// Make Sure WhatsApp ID is Registered
		remoteJID, err := WhatsAppCheckJID(jid, rjid)
		if err != nil {
			return "", err
		}

		// Set Chat Presence
		WhatsAppPresence(jid, true)
		WhatsAppComposeStatus(jid, remoteJID, true, false)
		defer func() {
			WhatsAppComposeStatus(jid, remoteJID, false, false)
			WhatsAppPresence(jid, false)
		}()

		// Compose WhatsApp Proto
		msgExtra := whatsmeow.SendRequestExtra{
			ID: WhatsAppClient[jid].GenerateMessageID(),
		}
		msgContent := &waproto.Message{
			LocationMessage: &waproto.LocationMessage{
				DegreesLatitude:  proto.Float64(latitude),
				DegreesLongitude: proto.Float64(longitude),
			},
		}

		// Send WhatsApp Message Proto
		_, err = WhatsAppClient[jid].SendMessage(ctx, remoteJID, msgContent, msgExtra)
		if err != nil {
			return "", err
		}

		return msgExtra.ID, nil
	}

	// Return Error WhatsApp Client is not Valid
	return "", errors.New("WhatsApp Client is not Valid")
}

func WhatsAppSendContact(ctx context.Context, jid string, rjid string, contactName string, contactNumber string) (string, error) {
	if WhatsAppClient[jid] != nil {
		var err error

		// Make Sure WhatsApp Client is OK
		err = WhatsAppIsClientOK(jid)
		if err != nil {
			return "", err
		}

		// Make Sure WhatsApp ID is Registered
		remoteJID, err := WhatsAppCheckJID(jid, rjid)
		if err != nil {
			return "", err
		}

		// Set Chat Presence
		WhatsAppPresence(jid, true)
		WhatsAppComposeStatus(jid, remoteJID, true, false)
		defer func() {
			WhatsAppComposeStatus(jid, remoteJID, false, false)
			WhatsAppPresence(jid, false)
		}()

		// Compose WhatsApp Proto
		msgExtra := whatsmeow.SendRequestExtra{
			ID: WhatsAppClient[jid].GenerateMessageID(),
		}
		msgVCard := fmt.Sprintf("BEGIN:VCARD\nVERSION:3.0\nN:;%v;;;\nFN:%v\nTEL;type=CELL;waid=%v:+%v\nEND:VCARD",
			contactName, contactName, contactNumber, contactNumber)
		msgContent := &waproto.Message{
			ContactMessage: &waproto.ContactMessage{
				DisplayName: proto.String(contactName),
				Vcard:       proto.String(msgVCard),
			},
		}

		// Send WhatsApp Message Proto
		_, err = WhatsAppClient[jid].SendMessage(ctx, remoteJID, msgContent, msgExtra)
		if err != nil {
			return "", err
		}

		return msgExtra.ID, nil
	}

	// Return Error WhatsApp Client is not Valid
	return "", errors.New("WhatsApp Client is not Valid")
}

func WhatsAppGroupGet(jid string) ([]types.GroupInfo, error) {
	if WhatsAppClient[jid] != nil {
		var err error

		// Make Sure WhatsApp Client is OK
		err = WhatsAppIsClientOK(jid)
		if err != nil {
			return nil, err
		}

		// Get Joined Group List
		groups, err := WhatsAppClient[jid].GetJoinedGroups()
		if err != nil {
			return nil, err
		}

		// Put Group Information in List
		var gids []types.GroupInfo
		for _, group := range groups {
			gids = append(gids, *group)
		}

		// Return Group Information List
		return gids, nil
	}

	// Return Error WhatsApp Client is not Valid
	return nil, errors.New("WhatsApp Client is not Valid")
}

func WhatsAppGroupJoin(jid string, link string) (string, error) {
	if WhatsAppClient[jid] != nil {
		var err error

		// Make Sure WhatsApp Client is OK
		err = WhatsAppIsClientOK(jid)
		if err != nil {
			return "", err
		}

		// Join Group By Invitation Link
		gid, err := WhatsAppClient[jid].JoinGroupWithLink(link)
		if err != nil {
			return "", err
		}

		// Return Joined Group ID
		return gid.String(), nil
	}

	// Return Error WhatsApp Client is not Valid
	return "", errors.New("WhatsApp Client is not Valid")
}

func WhatsAppGroupLeave(jid string, gjid string) error {
	if WhatsAppClient[jid] != nil {
		var err error

		// Make Sure WhatsApp Client is OK
		err = WhatsAppIsClientOK(jid)
		if err != nil {
			return err
		}

		// Make Sure WhatsApp ID is Registered
		groupJID, err := WhatsAppCheckJID(jid, gjid)
		if err != nil {
			return err
		}

		// Make Sure WhatsApp ID is Group Server
		if groupJID.Server != types.GroupServer {
			return errors.New("WhatsApp Group ID is Not Group Server")
		}

		// Leave Group By Group ID
		return WhatsAppClient[jid].LeaveGroup(groupJID)
	}

	// Return Error WhatsApp Client is not Valid
	return errors.New("WhatsApp Client is not Valid")
}
