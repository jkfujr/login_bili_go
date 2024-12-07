package bili

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/skip2/go-qrcode"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

func LaunchGUI() {
	a := app.New()
	w := a.NewWindow("比例比例扫码登录")
	w.Resize(fyne.NewSize(300, 400))
	// w.SetFixedSize(true)

	// UI
	statusLabel := widget.NewLabel("")
	qrImage := canvas.NewImageFromImage(nil)
	qrImage.SetMinSize(fyne.NewSize(300, 300))
	qrImage.FillMode = canvas.ImageFillContain

	startButton := widget.NewButton("开始扫码", nil)

	cookieEntry := widget.NewMultiLineEntry()
	cookieEntry.SetPlaceHolder("登录成功后, cookie显示在这里")
	cookieEntry.Wrapping = fyne.TextWrapWord
	cookieEntry.SetMinRowsVisible(5)

	qrContainer := container.NewStack(
		qrImage,
		container.NewCenter(startButton),
	)

	// 布局
	content := container.NewVBox(
		statusLabel,
		qrContainer,
		cookieEntry,
	)

	w.SetContent(container.NewCenter(content))

	var mu sync.Mutex

	startButton.OnTapped = func() {
		startButton.Hide()
		statusLabel.SetText("")
		cookieEntry.SetText("")
		qrImage.Image = nil
		qrImage.Refresh()

		go func() {
			loginKey, loginURL := get_login_key_and_login_url()

			pngData, err := qrcode.Encode(loginURL, qrcode.Medium, 256)
			if err != nil {
				dialog.ShowError(err, w)
				showStartButton(startButton, &mu)
				return
			}
			img, err := png.Decode(bytes.NewReader(pngData))
			if err != nil {
				dialog.ShowError(err, w)
				showStartButton(startButton, &mu)
				return
			}

			mu.Lock()
			qrImage.Image = img
			qrImage.Refresh()
			mu.Unlock()

			statusLabel.SetText("请使用bilibili客户端扫描二维码")

			verify_login(loginKey)

			// 登录成功
			mu.Lock()
			statusLabel.SetText("扫码成功")
			grayImg := toGrayScale(img)
			qrWithText := overlayText(grayImg, "扫码成功")
			qrImage.Image = qrWithText
			qrImage.Refresh()
			mu.Unlock()

			// 获取登录状态
			isLogin, data, cookieStr, _ := is_login()
			if isLogin {
				uname := data.Get("data.uname").String()

				mu.Lock()
				cookieEntry.SetText(cookieStr)
				mu.Unlock()

				// 保存 Cookie
				filename := getCookieFilename(cookieStr)
				err := os.WriteFile(filename, []byte(cookieStr), 0644)
				if err != nil {
					dialog.ShowError(err, w)
				} else {
					message := fmt.Sprintf("%s 登录成功\n已保存到 %v", uname, filename)
					dialog.ShowInformation("登录成功", message, w)
				}
			}

			// 重置UI
			mu.Lock()
			statusLabel.SetText("")
			qrImage.Image = nil
			qrImage.Refresh()
			mu.Unlock()
			showStartButton(startButton, &mu)
		}()
	}

	w.ShowAndRun()
}

func showStartButton(button *widget.Button, mu *sync.Mutex) {
	mu.Lock()
	defer mu.Unlock()
	button.Show()
}

func toGrayScale(img image.Image) image.Image {
	bounds := img.Bounds()
	grayImg := image.NewGray(bounds)
	draw.Draw(grayImg, bounds, img, image.Point{}, draw.Src)
	return grayImg
}

func overlayText(img image.Image, text string) image.Image {
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, image.Point{}, draw.Src)

	col := color.RGBA{255, 255, 255, 255}

	textWidth := len(text) * 7
	x := (bounds.Dx() - textWidth) / 2
	y := bounds.Dy() / 2

	point := fixed.Point26_6{
		X: fixed.I(x),
		Y: fixed.I(y),
	}

	d := &font.Drawer{
		Dst:  rgba,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13,
		Dot:  point,
	}
	d.DrawString(text)

	return rgba
}

func getCookieFilename(cookieStr string) string {
	parts := strings.Split(cookieStr, ";")
	userID := ""
	for _, part := range parts {
		if strings.HasPrefix(strings.TrimSpace(part), "DedeUserID=") {
			userID = strings.TrimPrefix(strings.TrimSpace(part), "DedeUserID=")
			break
		}
	}
	if userID == "" {
		userID = "unknown_user"
	}
	return fmt.Sprintf("%s_cookie.txt", userID)
}
