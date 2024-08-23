package whatsapp

import (
	"bytes"
	"io"
	"mime/multipart"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"

	"github.com/dimaskiddo/go-whatsapp-multidevice-rest/pkg/router"
	pkgWhatsApp "github.com/dimaskiddo/go-whatsapp-multidevice-rest/pkg/whatsapp"

	typAuth "github.com/dimaskiddo/go-whatsapp-multidevice-rest/internal/auth/types"
	typWhatsApp "github.com/dimaskiddo/go-whatsapp-multidevice-rest/internal/whatsapp/types"
)

func jwtPayload(c echo.Context) typAuth.AuthJWTClaimsPayload {
	jwtToken := c.Get("user").(*jwt.Token)
	jwtClaims := jwtToken.Claims.(*typAuth.AuthJWTClaims)

	return jwtClaims.Data
}

func convertFileToBytes(file multipart.File) ([]byte, error) {
	// Create Empty Buffer
	buffer := bytes.NewBuffer(nil)

	// Copy File Stream to Buffer
	_, err := io.Copy(buffer, file)
	if err != nil {
		return bytes.NewBuffer(nil).Bytes(), err
	}

	return buffer.Bytes(), nil
}

// Login
// @Summary     Generate QR Code for WhatsApp Multi-Device Login
// @Description Get QR Code for WhatsApp Multi-Device Login
// @Tags        WhatsApp Authentication
// @Accept      multipart/form-data
// @Produce     json
// @Produce     html
// @Param       output    formData  string  false  "Change Output Format in HTML or JSON"  Enums(html, json)  default(html)
// @Success     200
// @Security    BearerAuth
// @Router      /login [post]
func Login(c echo.Context) error {
	var err error
	jid := jwtPayload(c).JID

	var reqLogin typWhatsApp.RequestLogin
	reqLogin.Output = strings.TrimSpace(c.FormValue("output"))

	if len(reqLogin.Output) == 0 {
		reqLogin.Output = "html"
	}

	// Initialize WhatsApp Client
	pkgWhatsApp.WhatsAppInitClient(nil, jid)

	// Get WhatsApp QR Code Image
	qrCodeImage, qrCodeTimeout, err := pkgWhatsApp.WhatsAppLogin(jid)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	// If Return is Not QR Code But Reconnected
	// Then Return OK With Reconnected Status
	if qrCodeImage == "WhatsApp Client is Reconnected" {
		return router.ResponseSuccess(c, qrCodeImage)
	}

	var resLogin typWhatsApp.ResponseLogin
	resLogin.QRCode = qrCodeImage
	resLogin.Timeout = qrCodeTimeout

	if reqLogin.Output == "html" {
		htmlContent := `
    <html>
      <head>
        <title>WhatsApp Multi-Device Login</title>
        <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no" />
      </head>
      <body>
        <img src="` + resLogin.QRCode + `" />
        <p>
          <b>QR Code Scan</b>
          <br/>
          Timeout in ` + strconv.Itoa(resLogin.Timeout) + ` Second(s)
        </p>
      </body>
    </html>`

		return router.ResponseSuccessWithHTML(c, htmlContent)
	}

	return router.ResponseSuccessWithData(c, "Successfully Generated QR Code", resLogin)
}

// PairPhone
// @Summary     Pair Phone for WhatsApp Multi-Device Login
// @Description Get Pairing Code for WhatsApp Multi-Device Login
// @Tags        WhatsApp Authentication
// @Accept      multipart/form-data
// @Produce     json
// @Success     200
// @Security    BearerAuth
// @Router      /login/pair [post]
func LoginPair(c echo.Context) error {
	var err error
	jid := jwtPayload(c).JID

	// Initialize WhatsApp Client
	pkgWhatsApp.WhatsAppInitClient(nil, jid)

	// Get WhatsApp pairing Code text
	pairCode, pairCodeTimeout, err := pkgWhatsApp.WhatsAppLoginPair(jid)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	// If Return is not pairing code but Reconnected
	// Then Return OK With Reconnected Status
	if pairCode == "WhatsApp Client is Reconnected" {
		return router.ResponseSuccess(c, pairCode)
	}

	var resPairing typWhatsApp.ResponsePairing
	resPairing.PairCode = pairCode
	resPairing.Timeout = pairCodeTimeout

	return router.ResponseSuccessWithData(c, "Successfully Generated Pairing Code", resPairing)
}

// Logout
// @Summary     Logout Device from WhatsApp Multi-Device
// @Description Make Device Logout from WhatsApp Multi-Device
// @Tags        WhatsApp Authentication
// @Produce     json
// @Success     200
// @Security    BearerAuth
// @Router      /logout [post]
func Logout(c echo.Context) error {
	var err error
	jid := jwtPayload(c).JID

	err = pkgWhatsApp.WhatsAppLogout(jid)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccess(c, "Successfully Logged Out")
}

// Registered
// @Summary     Check If WhatsApp Personal ID is Registered
// @Description Check WhatsApp Personal ID is Registered
// @Tags        WhatsApp Information
// @Produce     json
// @Param       msisdn    query  string  true  "WhatsApp Personal ID to Check"
// @Success     200
// @Security    BearerAuth
// @Router      /registered [get]
func Registered(c echo.Context) error {
	jid := jwtPayload(c).JID
	remoteJID := strings.TrimSpace(c.QueryParam("msisdn"))

	if len(remoteJID) == 0 {
		return router.ResponseInternalError(c, "Missing Query Value MSISDN")
	}

	err := pkgWhatsApp.WhatsAppCheckRegistered(jid, remoteJID)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccess(c, "WhatsApp Personal ID is Registered")
}

// GetGroup
// @Summary     Get Joined Groups Information
// @Description Get Joined Groups Information from WhatsApp
// @Tags        WhatsApp Group
// @Produce     json
// @Success     200
// @Security    BearerAuth
// @Router      /group [get]
func GetGroup(c echo.Context) error {
	var err error
	jid := jwtPayload(c).JID

	group, err := pkgWhatsApp.WhatsAppGroupGet(jid)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccessWithData(c, "Successfully List Joined Groups", group)
}

// JoinGroup
// @Summary     Join Group From Invitation Link
// @Description Joining to Group From Invitation Link from WhatsApp
// @Tags        WhatsApp Group
// @Produce     json
// @Param       link    formData  string  true  "Group Invitation Link"
// @Success     200
// @Security    BearerAuth
// @Router      /group/join [post]
func JoinGroup(c echo.Context) error {
	var err error
	jid := jwtPayload(c).JID

	var reqGroupJoin typWhatsApp.RequestGroupJoin
	reqGroupJoin.Link = strings.TrimSpace(c.FormValue("link"))

	if len(reqGroupJoin.Link) == 0 {
		return router.ResponseBadRequest(c, "Missing Form Value Link")
	}

	group, err := pkgWhatsApp.WhatsAppGroupJoin(jid, reqGroupJoin.Link)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccessWithData(c, "Successfully Joined Group From Invitation Link", group)
}

// LeaveGroup
// @Summary     Leave Group By Group ID
// @Description Leaving Group By Group ID from WhatsApp
// @Tags        WhatsApp Group
// @Produce     json
// @Param       groupid    formData  string  true  "Group ID"
// @Success     200
// @Security    BearerAuth
// @Router      /group/leave [post]
func LeaveGroup(c echo.Context) error {
	var err error
	jid := jwtPayload(c).JID

	var reqGroupLeave typWhatsApp.RequestGroupLeave
	reqGroupLeave.GID = strings.TrimSpace(c.FormValue("groupid"))

	if len(reqGroupLeave.GID) == 0 {
		return router.ResponseBadRequest(c, "Missing Form Value Group ID")
	}

	err = pkgWhatsApp.WhatsAppGroupLeave(jid, reqGroupLeave.GID)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccess(c, "Successfully Leave Group By Group ID")
}

// SendText
// @Summary     Send Text Message
// @Description Send Text Message to Spesific WhatsApp Personal ID or Group ID
// @Tags        WhatsApp Send Message
// @Accept      multipart/form-data
// @Produce     json
// @Param       msisdn    formData  string  true  "Destination WhatsApp Personal ID or Group ID"
// @Param       message   formData  string  true  "Text Message"
// @Success     200
// @Security    BearerAuth
// @Router      /send/text [post]
func SendText(c echo.Context) error {
	var err error
	jid := jwtPayload(c).JID

	var reqSendMessage typWhatsApp.RequestSendMessage
	reqSendMessage.RJID = strings.TrimSpace(c.FormValue("msisdn"))
	reqSendMessage.Message = strings.TrimSpace(c.FormValue("message"))

	if len(reqSendMessage.RJID) == 0 {
		return router.ResponseBadRequest(c, "Missing Form Value MSISDN")
	}

	if len(reqSendMessage.Message) == 0 {
		return router.ResponseBadRequest(c, "Missing Form Value Message")
	}

	var resSendMessage typWhatsApp.ResponseSendMessage
	resSendMessage.MsgID, err = pkgWhatsApp.WhatsAppSendText(c.Request().Context(), jid, reqSendMessage.RJID, reqSendMessage.Message)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccessWithData(c, "Successfully Send Text Message", resSendMessage)
}

// SendLocation
// @Summary     Send Location Message
// @Description Send Location Message to Spesific WhatsApp Personal ID or Group ID
// @Tags        WhatsApp Send Message
// @Accept      multipart/form-data
// @Produce     json
// @Param       msisdn    formData  string  true  "Destination WhatsApp Personal ID or Group ID"
// @Param       latitude  formData  number  true  "Location Latitude"
// @Param       longitude formData  number  true  "Location Longitude"
// @Success     200
// @Security    BearerAuth
// @Router      /send/location [post]
func SendLocation(c echo.Context) error {
	var err error
	jid := jwtPayload(c).JID

	var reqSendLocation typWhatsApp.RequestSendLocation
	reqSendLocation.RJID = strings.TrimSpace(c.FormValue("msisdn"))

	reqSendLocation.Latitude, err = strconv.ParseFloat(strings.TrimSpace(c.FormValue("latitude")), 64)
	if err != nil {
		return router.ResponseInternalError(c, "Error While Decoding Latitude to Float64")
	}

	reqSendLocation.Longitude, err = strconv.ParseFloat(strings.TrimSpace(c.FormValue("longitude")), 64)
	if err != nil {
		return router.ResponseInternalError(c, "Error While Decoding Longitude to Float64")
	}

	if len(reqSendLocation.RJID) == 0 {
		return router.ResponseBadRequest(c, "Missing Form Value MSISDN")
	}

	var resSendMessage typWhatsApp.ResponseSendMessage
	resSendMessage.MsgID, err = pkgWhatsApp.WhatsAppSendLocation(c.Request().Context(), jid, reqSendLocation.RJID, reqSendLocation.Latitude, reqSendLocation.Longitude)
	if err != nil {
		return router.ResponseInternalError(c, err.Error())
	}

	return router.ResponseSuccessWithData(c, "Successfully Send Location Message", resSendMessage)
}